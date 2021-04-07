/*
Copyright 2019 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/progress"
	"github.com/docker/docker/pkg/streamformatter"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

const (
	retries   = 5
	sleepTime = 1 * time.Second
)

type ContainerRun struct {
	Image       string
	User        string
	Command     []string
	Mounts      []mount.Mount
	Env         []string
	BeforeStart func(context.Context, string) error
}

// LocalDaemon talks to a local Docker API.
type LocalDaemon interface {
	Close() error
	ExtraEnv() []string
	ServerVersion(ctx context.Context) (types.Version, error)
	ConfigFile(ctx context.Context, image string) (*v1.ConfigFile, error)
	Build(ctx context.Context, out io.Writer, workspace string, artifact string, a *latest.DockerArtifact, opts BuildOptions) (string, error)
	Push(ctx context.Context, out io.Writer, ref string) (string, error)
	Pull(ctx context.Context, out io.Writer, ref string) error
	Load(ctx context.Context, out io.Writer, input io.Reader, ref string) (string, error)
	Tag(ctx context.Context, image, ref string) error
	TagWithImageID(ctx context.Context, ref string, imageID string) (string, error)
	ImageID(ctx context.Context, ref string) (string, error)
	ImageInspectWithRaw(ctx context.Context, image string) (types.ImageInspect, []byte, error)
	ImageRemove(ctx context.Context, image string, opts types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error)
	ImageExists(ctx context.Context, ref string) bool
	ImageList(ctx context.Context, ref string) ([]types.ImageSummary, error)
	Prune(ctx context.Context, images []string, pruneChildren bool) ([]string, error)
	DiskUsage(ctx context.Context) (uint64, error)
	RawClient() client.CommonAPIClient
}

// BuildOptions provides parameters related to the LocalDaemon build.
type BuildOptions struct {
	Tag            string
	Mode           config.RunMode
	ExtraBuildArgs map[string]*string
}

type localDaemon struct {
	cfg            Config
	forceRemove    bool
	apiClient      client.CommonAPIClient
	extraEnv       []string
	imageCache     map[string]*v1.ConfigFile
	imageCacheLock sync.Mutex
}

// NewLocalDaemon creates a new LocalDaemon.
func NewLocalDaemon(apiClient client.CommonAPIClient, extraEnv []string, forceRemove bool, cfg Config) LocalDaemon {
	return &localDaemon{
		cfg:         cfg,
		apiClient:   apiClient,
		extraEnv:    extraEnv,
		forceRemove: forceRemove,
		imageCache:  make(map[string]*v1.ConfigFile),
	}
}

// ExtraEnv returns the env variables needed to point at this local Docker
// eg. minikube. This has be set in addition to the current environment.
func (l *localDaemon) ExtraEnv() []string {
	return l.extraEnv
}

// PushResult gives the information on an image that has been pushed.
type PushResult struct {
	Digest string
}

// BuildResult gives the information on an image that has been built.
type BuildResult struct {
	ID string
}

func (l *localDaemon) RawClient() client.CommonAPIClient {
	return l.apiClient
}

// Close closes the connection with the local daemon.
func (l *localDaemon) Close() error {
	return l.apiClient.Close()
}

// ServerVersion retrieves the version information from the server.
func (l *localDaemon) ServerVersion(ctx context.Context) (types.Version, error) {
	return l.apiClient.ServerVersion(ctx)
}

// ConfigFile retrieves and caches image configurations.
func (l *localDaemon) ConfigFile(ctx context.Context, image string) (*v1.ConfigFile, error) {
	l.imageCacheLock.Lock()
	defer l.imageCacheLock.Unlock()

	cachedCfg, present := l.imageCache[image]
	if present {
		return cachedCfg, nil
	}

	cfg := &v1.ConfigFile{}

	_, raw, err := l.apiClient.ImageInspectWithRaw(ctx, image)
	if err == nil {
		if err := json.Unmarshal(raw, cfg); err != nil {
			return nil, err
		}
	} else {
		cfg, err = RetrieveRemoteConfig(image, l.cfg)
		if err != nil {
			return nil, err
		}
	}

	l.imageCache[image] = cfg

	return cfg, nil
}

func (l *localDaemon) CheckCompatible(a *latest.DockerArtifact) error {
	if a.Secret != nil || a.SSH != "" {
		return fmt.Errorf("docker build options, secrets and ssh, require BuildKit - set `useBuildkit: true` in your config, or run with `DOCKER_BUILDKIT=1`")
	}
	return nil
}

// Build performs a docker build and returns the imageID.
func (l *localDaemon) Build(ctx context.Context, out io.Writer, workspace string, artifact string, a *latest.DockerArtifact, opts BuildOptions) (string, error) {
	logrus.Debugf("Running docker build: context: %s, dockerfile: %s", workspace, a.DockerfilePath)

	if err := l.CheckCompatible(a); err != nil {
		return "", err
	}
	buildArgs, err := EvalBuildArgs(opts.Mode, workspace, a.DockerfilePath, a.BuildArgs, opts.ExtraBuildArgs)
	if err != nil {
		return "", fmt.Errorf("unable to evaluate build args: %w", err)
	}

	// Like `docker build`, we ignore the errors
	// See https://github.com/docker/cli/blob/75c1bb1f33d7cedbaf48404597d5bf9818199480/cli/command/image/build.go#L364
	authConfigs, _ := DefaultAuthHelper.GetAllAuthConfigs(ctx)

	buildCtx, buildCtxWriter := io.Pipe()
	go func() {
		err := CreateDockerTarContext(ctx, buildCtxWriter,
			NewBuildConfig(workspace, artifact, a.DockerfilePath, buildArgs), l.cfg)
		if err != nil {
			buildCtxWriter.CloseWithError(fmt.Errorf("creating docker context: %w", err))
			return
		}
		buildCtxWriter.Close()
	}()

	progressOutput := streamformatter.NewProgressOutput(out)
	body := progress.NewProgressReader(buildCtx, progressOutput, 0, "", "Sending build context to Docker daemon")

	resp, err := l.apiClient.ImageBuild(ctx, body, types.ImageBuildOptions{
		Tags:        []string{opts.Tag},
		Dockerfile:  a.DockerfilePath,
		BuildArgs:   buildArgs,
		CacheFrom:   a.CacheFrom,
		AuthConfigs: authConfigs,
		Target:      a.Target,
		ForceRemove: l.forceRemove,
		NetworkMode: strings.ToLower(a.NetworkMode),
		NoCache:     a.NoCache,
	})
	if err != nil {
		return "", fmt.Errorf("docker build: %w", err)
	}
	defer resp.Body.Close()

	var imageID string
	auxCallback := func(msg jsonmessage.JSONMessage) {
		if msg.Aux == nil {
			return
		}

		var result BuildResult
		if err := json.Unmarshal(*msg.Aux, &result); err != nil {
			logrus.Debugln("Unable to parse build output:", err)
			return
		}
		imageID = result.ID
	}

	if err := streamDockerMessages(out, resp.Body, auxCallback); err != nil {
		return "", fmt.Errorf("unable to stream build output: %w", err)
	}

	if imageID == "" {
		// Maybe this version of Docker doesn't return the digest of the image
		// that has been built.
		imageID, err = l.ImageID(ctx, opts.Tag)
		if err != nil {
			return "", fmt.Errorf("getting digest: %w", err)
		}
	}

	return imageID, nil
}

// streamDockerMessages streams formatted json output from the docker daemon
func streamDockerMessages(dst io.Writer, src io.Reader, auxCallback func(jsonmessage.JSONMessage)) error {
	termFd, isTerm := util.IsTerminal(dst)
	return jsonmessage.DisplayJSONMessagesStream(src, dst, termFd, isTerm, auxCallback)
}

// Push pushes an image reference to a registry. Returns the image digest.
func (l *localDaemon) Push(ctx context.Context, out io.Writer, ref string) (string, error) {
	registryAuth, err := l.encodedRegistryAuth(ctx, DefaultAuthHelper, ref)
	if err != nil {
		return "", fmt.Errorf("getting auth config for %q: %w", ref, err)
	}

	// Quick check if the image was already pushed (ignore any error).
	if alreadyPushed, digest, err := l.isAlreadyPushed(ctx, ref, registryAuth); alreadyPushed && err == nil {
		return digest, nil
	}

	rc, err := l.apiClient.ImagePush(ctx, ref, types.ImagePushOptions{
		RegistryAuth: registryAuth,
	})
	if err != nil {
		return "", fmt.Errorf("%s %q: %w", sErrors.PushImageErr, ref, err)
	}
	defer rc.Close()

	var digest string
	auxCallback := func(msg jsonmessage.JSONMessage) {
		if msg.Aux == nil {
			return
		}

		var result PushResult
		if err := json.Unmarshal(*msg.Aux, &result); err != nil {
			logrus.Debugln("Unable to parse push output:", err)
			return
		}
		digest = result.Digest
	}

	if err := streamDockerMessages(out, rc, auxCallback); err != nil {
		return "", fmt.Errorf("%s %q: %w", sErrors.PushImageErr, ref, err)
	}

	if digest == "" {
		// Maybe this version of Docker doesn't return the digest of the image
		// that has been pushed.
		digest, err = RemoteDigest(ref, l.cfg)
		if err != nil {
			return "", fmt.Errorf("getting digest: %w", err)
		}
	}

	return digest, nil
}

// isAlreadyPushed quickly checks if the local image has already been pushed.
func (l *localDaemon) isAlreadyPushed(ctx context.Context, ref, registryAuth string) (bool, string, error) {
	localImage, _, err := l.apiClient.ImageInspectWithRaw(ctx, ref)
	if err != nil {
		return false, "", err
	}

	if len(localImage.RepoDigests) == 0 {
		return false, "", nil
	}

	remoteImage, err := l.apiClient.DistributionInspect(ctx, ref, registryAuth)
	if err != nil {
		return false, "", err
	}
	digest := remoteImage.Descriptor.Digest.String()

	for _, repoDigest := range localImage.RepoDigests {
		if parsed, err := ParseReference(repoDigest); err == nil {
			if parsed.Digest == digest {
				return true, parsed.Digest, nil
			}
		}
	}

	return false, "", nil
}

// Pull pulls an image reference from a registry.
func (l *localDaemon) Pull(ctx context.Context, out io.Writer, ref string) error {
	// We first try pulling the image with credentials.  If that fails then retry
	// without credentials in case the image is public.

	// Set CLOUDSDK_CORE_VERBOSITY to suppress error messages emitted by docker-credential-gcloud
	// when the user is not authenticated or lacks credentials to pull the given image.  The errors
	// are irrelevant when the image is public (e.g., `gcr.io/buildpacks/builder:v1`).`
	// If the image is private, the error from GCR directs the user to the GCR authentication
	// page which provides steps to rememdy the situation.
	if v, found := os.LookupEnv("CLOUDSDK_CORE_VERBOSITY"); found {
		defer os.Setenv("CLOUDSDK_CORE_VERBOSITY", v)
	} else {
		defer os.Unsetenv("CLOUDSDK_CORE_VERBOSITY")
	}
	os.Setenv("CLOUDSDK_CORE_VERBOSITY", "critical")

	// Eargerly create credentials.
	registryAuth, err := l.encodedRegistryAuth(ctx, DefaultAuthHelper, ref)
	// Let's ignore the error because maybe the image is public
	// and can be pulled without credentials.
	rc, err := l.apiClient.ImagePull(ctx, ref, types.ImagePullOptions{
		RegistryAuth: registryAuth,
		PrivilegeFunc: func() (string, error) {
			// The first pull is unauthorized. There are two situations:
			//   1. if `encodedRegistryAuth()` errored, then `registryAuth == ""` and so we've
			//     tried an anonymous pull which has failed.  So return the original error from
			//     `encodedRegistryAuth()`.
			//   2. If `encodedRegistryAuth()` succeeded (so `err == nil`), then our credential was rejected, so
			//     return "" to retry as an anonymous pull.
			return "", err
		},
	})
	if err != nil {
		return fmt.Errorf("pulling image from repository: %w", err)
	}
	defer rc.Close()

	return streamDockerMessages(out, rc, nil)
}

// Load loads an image from a tar file. Returns the imageID for the loaded image.
func (l *localDaemon) Load(ctx context.Context, out io.Writer, input io.Reader, ref string) (string, error) {
	resp, err := l.apiClient.ImageLoad(ctx, input, false)
	if err != nil {
		return "", fmt.Errorf("loading image into docker daemon: %w", err)
	}
	defer resp.Body.Close()

	if err := streamDockerMessages(out, resp.Body, nil); err != nil {
		return "", fmt.Errorf("reading from image load response: %w", err)
	}

	return l.ImageID(ctx, ref)
}

// Tag adds a tag to an image.
func (l *localDaemon) Tag(ctx context.Context, image, ref string) error {
	return l.apiClient.ImageTag(ctx, image, ref)
}

// For k8s, we need a unique, immutable ID for the image.
// k8s doesn't recognize the imageID or any combination of the image name
// suffixed with the imageID, as a valid image name.
// So, the solution we chose is to create a tag, just for Skaffold, from
// the imageID, and use that in the manifests.
func (l *localDaemon) TagWithImageID(ctx context.Context, ref string, imageID string) (string, error) {
	parsed, err := ParseReference(ref)
	if err != nil {
		return "", err
	}

	uniqueTag := parsed.BaseName + ":" + strings.TrimPrefix(imageID, "sha256:")
	if err := l.Tag(ctx, imageID, uniqueTag); err != nil {
		return "", err
	}

	return uniqueTag, nil
}

// ImageID returns the image ID for a corresponding reference.
func (l *localDaemon) ImageID(ctx context.Context, ref string) (string, error) {
	image, _, err := l.apiClient.ImageInspectWithRaw(ctx, ref)
	if err != nil {
		if client.IsErrNotFound(err) {
			return "", nil
		}
		return "", localDigestGetErr(ref, err)
	}

	return image.ID, nil
}

func (l *localDaemon) ImageExists(ctx context.Context, ref string) bool {
	_, _, err := l.apiClient.ImageInspectWithRaw(ctx, ref)
	return err == nil
}

func (l *localDaemon) ImageInspectWithRaw(ctx context.Context, image string) (types.ImageInspect, []byte, error) {
	return l.apiClient.ImageInspectWithRaw(ctx, image)
}

func (l *localDaemon) ImageRemove(ctx context.Context, image string, opts types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error) {
	for i := 0; i < retries; i++ {
		resp, err := l.apiClient.ImageRemove(ctx, image, opts)
		if err == nil {
			return resp, nil
		}
		if _, ok := err.(errdefs.ErrConflict); !ok {
			return nil, err
		}
		time.Sleep(sleepTime)
	}
	return nil, fmt.Errorf("could not remove image %q after %d retries", image, retries)
}

func (l *localDaemon) ImageList(ctx context.Context, ref string) ([]types.ImageSummary, error) {
	return l.apiClient.ImageList(ctx, types.ImageListOptions{
		Filters: filters.NewArgs(filters.Arg("reference", ref)),
	})
}
func (l *localDaemon) DiskUsage(ctx context.Context) (uint64, error) {
	usage, err := l.apiClient.DiskUsage(ctx)
	if err != nil {
		return 0, err
	}
	return uint64(usage.LayersSize), nil
}

func ToCLIBuildArgs(a *latest.DockerArtifact, evaluatedArgs map[string]*string) ([]string, error) {
	var args []string
	var keys []string
	for k := range evaluatedArgs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		args = append(args, "--build-arg")

		v := evaluatedArgs[k]
		if v == nil {
			args = append(args, k)
		} else {
			args = append(args, fmt.Sprintf("%s=%s", k, *v))
		}
	}

	for _, from := range a.CacheFrom {
		args = append(args, "--cache-from", from)
	}

	if a.Target != "" {
		args = append(args, "--target", a.Target)
	}

	if a.NetworkMode != "" {
		args = append(args, "--network", strings.ToLower(a.NetworkMode))
	}

	if a.NoCache {
		args = append(args, "--no-cache")
	}

	if a.Squash {
		args = append(args, "--squash")
	}

	if a.Secret != nil {
		secretString := fmt.Sprintf("id=%s", a.Secret.ID)
		if a.Secret.Source != "" {
			secretString += ",src=" + a.Secret.Source
		}
		if a.Secret.Destination != "" {
			secretString += ",dst=" + a.Secret.Destination
		}
		args = append(args, "--secret", secretString)
	}

	if a.SSH != "" {
		args = append(args, "--ssh", a.SSH)
	}

	return args, nil
}

func (l *localDaemon) Prune(ctx context.Context, images []string, pruneChildren bool) ([]string, error) {
	var pruned []string
	var errRt error
	for _, id := range images {
		resp, err := l.ImageRemove(ctx, id, types.ImageRemoveOptions{
			Force:         true,
			PruneChildren: pruneChildren,
		})
		if err == nil {
			pruned = append(pruned, id)
		} else if errRt == nil {
			// save the first error
			errRt = fmt.Errorf("pruning images: %w", err)
		}

		for _, r := range resp {
			if r.Deleted != "" {
				logrus.Debugf("deleted image %s\n", r.Deleted)
			}
			if r.Untagged != "" {
				logrus.Debugf("untagged image %s\n", r.Untagged)
			}
		}
	}
	return pruned, errRt
}
