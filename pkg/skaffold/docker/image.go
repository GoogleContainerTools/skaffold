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
	"sort"
	"strings"
	"sync"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/progress"
	"github.com/docker/docker/pkg/streamformatter"
	"github.com/docker/docker/pkg/term"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// LocalDaemon talks to a local Docker API.
type LocalDaemon interface {
	Close() error
	ExtraEnv() []string
	ServerVersion(ctx context.Context) (types.Version, error)
	ConfigFile(ctx context.Context, image string) (*v1.ConfigFile, error)
	Build(ctx context.Context, out io.Writer, workspace string, a *latest.DockerArtifact, ref string) (string, error)
	Push(ctx context.Context, out io.Writer, ref string) (string, error)
	Pull(ctx context.Context, out io.Writer, ref string) error
	Load(ctx context.Context, out io.Writer, input io.Reader, ref string) (string, error)
	Tag(ctx context.Context, image, ref string) error
	ImageID(ctx context.Context, ref string) (string, error)
	ImageInspectWithRaw(ctx context.Context, image string) (types.ImageInspect, []byte, error)
	ImageRemove(ctx context.Context, image string, opts types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error)
	RepoDigest(ctx context.Context, ref string) (string, error)
	ImageList(ctx context.Context, options types.ImageListOptions) ([]types.ImageSummary, error)
	ImageExists(ctx context.Context, ref string) bool
}

type localDaemon struct {
	forceRemove        bool
	insecureRegistries map[string]bool
	apiClient          client.CommonAPIClient
	extraEnv           []string
	imageCache         map[string]*v1.ConfigFile
	imageCacheLock     sync.Mutex
}

// NewLocalDaemon creates a new LocalDaemon.
func NewLocalDaemon(apiClient client.CommonAPIClient, extraEnv []string, forceRemove bool, insecureRegistries map[string]bool) LocalDaemon {
	return &localDaemon{
		apiClient:          apiClient,
		extraEnv:           extraEnv,
		forceRemove:        forceRemove,
		insecureRegistries: insecureRegistries,
		imageCache:         make(map[string]*v1.ConfigFile),
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
		cfg, err = RetrieveRemoteConfig(image, l.insecureRegistries)
		if err != nil {
			return nil, errors.Wrap(err, "getting remote config")
		}
	}

	l.imageCache[image] = cfg

	return cfg, nil
}

// Build performs a docker build and returns the imageID.
func (l *localDaemon) Build(ctx context.Context, out io.Writer, workspace string, a *latest.DockerArtifact, ref string) (string, error) {
	logrus.Debugf("Running docker build: context: %s, dockerfile: %s", workspace, a.DockerfilePath)

	// Like `docker build`, we ignore the errors
	// See https://github.com/docker/cli/blob/75c1bb1f33d7cedbaf48404597d5bf9818199480/cli/command/image/build.go#L364
	authConfigs, _ := DefaultAuthHelper.GetAllAuthConfigs()

	buildArgs, err := EvaluateBuildArgs(a.BuildArgs)
	if err != nil {
		return "", errors.Wrap(err, "unable to evaluate build args")
	}

	buildCtx, buildCtxWriter := io.Pipe()
	go func() {
		err := CreateDockerTarContext(ctx, buildCtxWriter, workspace, a, l.insecureRegistries)
		if err != nil {
			buildCtxWriter.CloseWithError(errors.Wrap(err, "creating docker context"))
			return
		}
		buildCtxWriter.Close()
	}()

	progressOutput := streamformatter.NewProgressOutput(out)
	body := progress.NewProgressReader(buildCtx, progressOutput, 0, "", "Sending build context to Docker daemon")

	resp, err := l.apiClient.ImageBuild(ctx, body, types.ImageBuildOptions{
		Tags:        []string{ref},
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
		return "", errors.Wrap(err, "docker build")
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
		return "", errors.Wrap(err, "unable to stream build output")
	}

	if imageID == "" {
		// Maybe this version of Docker doesn't return the digest of the image
		// that has been built.
		imageID, err = l.ImageID(ctx, ref)
		if err != nil {
			return "", errors.Wrap(err, "getting digest")
		}
	}

	return imageID, nil
}

// streamDockerMessages streams formatted json output from the docker daemon
// TODO(@r2d4): Make this output much better, this is the bare minimum
func streamDockerMessages(dst io.Writer, src io.Reader, auxCallback func(jsonmessage.JSONMessage)) error {
	fd, _ := term.GetFdInfo(dst)
	return jsonmessage.DisplayJSONMessagesStream(src, dst, fd, false, auxCallback)
}

// Push pushes an image reference to a registry. Returns the image digest.
func (l *localDaemon) Push(ctx context.Context, out io.Writer, ref string) (string, error) {
	registryAuth, err := l.encodedRegistryAuth(ctx, DefaultAuthHelper, ref)
	if err != nil {
		return "", errors.Wrapf(err, "getting auth config for %s", ref)
	}

	rc, err := l.apiClient.ImagePush(ctx, ref, types.ImagePushOptions{
		RegistryAuth: registryAuth,
	})
	if err != nil {
		return "", errors.Wrap(err, "pushing image to repository")
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
		return "", err
	}

	if digest == "" {
		// Maybe this version of Docker doesn't return the digest of the image
		// that has been pushed.
		digest, err = RemoteDigest(ref, l.insecureRegistries)
		if err != nil {
			return "", errors.Wrap(err, "getting digest")
		}
	}

	return digest, nil
}

// Pull pulls an image reference from a registry.
func (l *localDaemon) Pull(ctx context.Context, out io.Writer, ref string) error {
	registryAuth, err := l.encodedRegistryAuth(ctx, DefaultAuthHelper, ref)
	if err != nil {
		return errors.Wrapf(err, "getting auth config for %s", ref)
	}

	rc, err := l.apiClient.ImagePull(ctx, ref, types.ImagePullOptions{
		RegistryAuth: registryAuth,
	})
	if err != nil {
		return errors.Wrap(err, "pulling image from repository")
	}
	defer rc.Close()

	return streamDockerMessages(out, rc, nil)
}

// Load loads an image from a tar file. Returns the imageID for the loaded image.
func (l *localDaemon) Load(ctx context.Context, out io.Writer, input io.Reader, ref string) (string, error) {
	resp, err := l.apiClient.ImageLoad(ctx, input, false)
	if err != nil {
		return "", errors.Wrap(err, "loading image into docker daemon")
	}
	defer resp.Body.Close()

	err = streamDockerMessages(out, resp.Body, nil)
	if err != nil {
		return "", errors.Wrap(err, "reading from image load response")
	}

	return l.ImageID(ctx, ref)
}

// Tag adds a tag to an image.
func (l *localDaemon) Tag(ctx context.Context, image, ref string) error {
	return l.apiClient.ImageTag(ctx, image, ref)
}

// ImageID returns the image ID for a corresponding reference.
func (l *localDaemon) ImageID(ctx context.Context, ref string) (string, error) {
	image, _, err := l.apiClient.ImageInspectWithRaw(ctx, ref)
	if err != nil {
		if client.IsErrNotFound(err) {
			return "", nil
		}
		return "", errors.Wrap(err, "inspecting image")
	}

	return image.ID, nil
}

// ImageList returns a list of all images in the local daemon
func (l *localDaemon) ImageList(ctx context.Context, options types.ImageListOptions) ([]types.ImageSummary, error) {
	return l.apiClient.ImageList(ctx, options)
}

// RepoDigest returns a repo digest for the given ref
func (l *localDaemon) RepoDigest(ctx context.Context, ref string) (string, error) {
	image, _, err := l.apiClient.ImageInspectWithRaw(ctx, ref)
	if err != nil {
		return "", errors.Wrap(err, "inspecting image")
	}
	if len(image.RepoDigests) == 0 {
		return "", nil
	}
	return image.RepoDigests[0], nil
}

func (l *localDaemon) ImageExists(ctx context.Context, ref string) bool {
	_, _, err := l.apiClient.ImageInspectWithRaw(ctx, ref)
	return err == nil
}

func (l *localDaemon) ImageInspectWithRaw(ctx context.Context, image string) (types.ImageInspect, []byte, error) {
	return l.apiClient.ImageInspectWithRaw(ctx, image)
}

func (l *localDaemon) ImageRemove(ctx context.Context, image string, opts types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error) {
	return l.apiClient.ImageRemove(ctx, image, opts)
}

// GetBuildArgs gives the build args flags for docker build.
func GetBuildArgs(a *latest.DockerArtifact) ([]string, error) {
	var args []string

	buildArgs, err := EvaluateBuildArgs(a.BuildArgs)
	if err != nil {
		return nil, errors.Wrap(err, "unable to evaluate build args")
	}

	var keys []string
	for k := range buildArgs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		args = append(args, "--build-arg")

		v := buildArgs[k]
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

	return args, nil
}

// EvaluateBuildArgs evaluates templated build args.
func EvaluateBuildArgs(args map[string]*string) (map[string]*string, error) {
	if args == nil {
		return nil, nil
	}

	evaluated := map[string]*string{}
	for k, v := range args {
		if v == nil {
			evaluated[k] = nil
			continue
		}

		tmpl, err := util.ParseEnvTemplate(*v)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to parse template for build arg: %s=%s", k, *v)
		}

		value, err := util.ExecuteEnvTemplate(tmpl, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to get value for build arg: %s", k)
		}
		evaluated[k] = &value
	}

	return evaluated, nil
}
