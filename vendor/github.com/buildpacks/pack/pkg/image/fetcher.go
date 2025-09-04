package image

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/buildpacks/imgutil/layout"
	"github.com/buildpacks/imgutil/layout/sparse"
	cerrdefs "github.com/containerd/errdefs"

	"github.com/buildpacks/imgutil"
	"github.com/buildpacks/imgutil/local"
	"github.com/buildpacks/imgutil/remote"
	"github.com/buildpacks/lifecycle/auth"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/pkg/errors"

	pname "github.com/buildpacks/pack/internal/name"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/internal/term"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/logging"
)

// FetcherOption is a type of function that mutate settings on the client.
// Values in these functions are set through currying.
type FetcherOption func(c *Fetcher)

type LayoutOption struct {
	Path   string
	Sparse bool
}

// WithRegistryMirrors supply your own mirrors for registry.
func WithRegistryMirrors(registryMirrors map[string]string) FetcherOption {
	return func(c *Fetcher) {
		c.registryMirrors = registryMirrors
	}
}

func WithKeychain(keychain authn.Keychain) FetcherOption {
	return func(c *Fetcher) {
		c.keychain = keychain
	}
}

type DockerClient interface {
	local.DockerClient
	ImagePull(ctx context.Context, ref string, options image.PullOptions) (io.ReadCloser, error)
}

type Fetcher struct {
	docker          DockerClient
	logger          logging.Logger
	registryMirrors map[string]string
	keychain        authn.Keychain
}

type FetchOptions struct {
	Daemon             bool
	Target             *dist.Target
	PullPolicy         PullPolicy
	LayoutOption       LayoutOption
	InsecureRegistries []string
}

func NewFetcher(logger logging.Logger, docker DockerClient, opts ...FetcherOption) *Fetcher {
	fetcher := &Fetcher{
		logger:   logger,
		docker:   docker,
		keychain: authn.DefaultKeychain,
	}

	for _, opt := range opts {
		opt(fetcher)
	}

	return fetcher
}

var ErrNotFound = errors.New("not found")

func (f *Fetcher) Fetch(ctx context.Context, name string, options FetchOptions) (imgutil.Image, error) {
	name, err := pname.TranslateRegistry(name, f.registryMirrors, f.logger)
	if err != nil {
		return nil, err
	}

	if (options.LayoutOption != LayoutOption{}) {
		return f.fetchLayoutImage(name, options.LayoutOption)
	}

	if !options.Daemon {
		return f.fetchRemoteImage(name, options.Target, options.InsecureRegistries)
	}

	switch options.PullPolicy {
	case PullNever:
		img, err := f.fetchDaemonImage(name)
		return img, err
	case PullIfNotPresent:
		img, err := f.fetchDaemonImage(name)
		if err == nil || !errors.Is(err, ErrNotFound) {
			return img, err
		}
	}

	platform := ""
	msg := fmt.Sprintf("Pulling image %s", style.Symbol(name))
	if options.Target != nil {
		platform = options.Target.ValuesAsPlatform()
		msg = fmt.Sprintf("Pulling image %s with platform %s", style.Symbol(name), style.Symbol(platform))
	}
	f.logger.Debug(msg)
	if err = f.pullImage(ctx, name, platform); err != nil {
		// FIXME: this matching is brittle and the fallback should be removed when https://github.com/buildpacks/pack/issues/2079
		// has been fixed for a sufficient amount of time.
		// Sample error from docker engine:
		// `image with reference <image> was found but does not match the specified platform: wanted linux/amd64, actual: linux` or
		// `image with reference <image> was found but its platform (linux) does not match the specified platform (linux/amd64)`
		if strings.Contains(err.Error(), "does not match the specified platform") {
			f.logger.Debugf(fmt.Sprintf("Pulling image %s", style.Symbol(name)))
			err = f.pullImage(ctx, name, "")
		}
	}
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	return f.fetchDaemonImage(name)
}

func (f *Fetcher) CheckReadAccess(repo string, options FetchOptions) bool {
	if !options.Daemon || options.PullPolicy == PullAlways {
		return f.checkRemoteReadAccess(repo)
	}
	if _, err := f.fetchDaemonImage(repo); err != nil {
		if errors.Is(err, ErrNotFound) {
			// Image doesn't exist in the daemon
			// 	Pull Never: should fail
			// 	Pull If Not Present: need to check the registry
			if options.PullPolicy == PullNever {
				return false
			}
			return f.checkRemoteReadAccess(repo)
		}
		f.logger.Debugf("failed reading image '%s' from the daemon, error: %s", repo, err.Error())
		return false
	}
	return true
}

func (f *Fetcher) checkRemoteReadAccess(repo string) bool {
	img, err := remote.NewImage(repo, f.keychain)
	if err != nil {
		f.logger.Debugf("failed accessing remote image %s, error: %s", repo, err.Error())
		return false
	}
	if ok, err := img.CheckReadAccess(); ok {
		f.logger.Debugf("CheckReadAccess succeeded for the run image %s", repo)
		return true
	} else {
		f.logger.Debugf("CheckReadAccess failed for the run image %s, error: %s", repo, err.Error())
		return false
	}
}

func (f *Fetcher) fetchDaemonImage(name string) (imgutil.Image, error) {
	image, err := local.NewImage(name, f.docker, local.FromBaseImage(name))
	if err != nil {
		return nil, err
	}

	if !image.Found() {
		return nil, errors.Wrapf(ErrNotFound, "image %s does not exist on the daemon", style.Symbol(name))
	}

	return image, nil
}

func (f *Fetcher) fetchRemoteImage(name string, target *dist.Target, insecureRegistries []string) (imgutil.Image, error) {
	var (
		image   imgutil.Image
		options []imgutil.ImageOption
		err     error
	)

	if len(insecureRegistries) > 0 {
		for _, registry := range insecureRegistries {
			options = append(options, remote.WithRegistrySetting(registry, true))
		}
	}

	if target == nil {
		image, err = remote.NewImage(name, f.keychain, append(options, remote.FromBaseImage(name))...)
	} else {
		platform := imgutil.Platform{OS: target.OS, Architecture: target.Arch, Variant: target.ArchVariant}
		image, err = remote.NewImage(name, f.keychain, append(append(options, remote.FromBaseImage(name)), remote.WithDefaultPlatform(platform))...)
	}

	if err != nil {
		return nil, err
	}

	if !image.Found() {
		return nil, errors.Wrapf(ErrNotFound, "image %s does not exist in registry", style.Symbol(name))
	}

	return image, nil
}

func (f *Fetcher) fetchLayoutImage(name string, options LayoutOption) (imgutil.Image, error) {
	var (
		image imgutil.Image
		err   error
	)

	v1Image, err := remote.NewV1Image(name, f.keychain)
	if err != nil {
		return nil, err
	}

	if options.Sparse {
		image, err = sparse.NewImage(options.Path, v1Image)
	} else {
		image, err = layout.NewImage(options.Path, layout.FromBaseImageInstance(v1Image))
	}

	if err != nil {
		return nil, err
	}

	err = image.Save()
	if err != nil {
		return nil, err
	}

	return image, nil
}

func (f *Fetcher) pullImage(ctx context.Context, imageID string, platform string) error {
	regAuth, err := f.registryAuth(imageID)
	if err != nil {
		return err
	}

	rc, err := f.docker.ImagePull(ctx, imageID, image.PullOptions{RegistryAuth: regAuth, Platform: platform})
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return errors.Wrapf(ErrNotFound, "image %s does not exist on the daemon", style.Symbol(imageID))
		}

		return err
	}

	writer := logging.GetWriterForLevel(f.logger, logging.InfoLevel)
	termFd, isTerm := term.IsTerminal(writer)

	err = jsonmessage.DisplayJSONMessagesStream(rc, &colorizedWriter{writer}, termFd, isTerm, nil)
	if err != nil {
		return err
	}

	return rc.Close()
}

func (f *Fetcher) registryAuth(ref string) (string, error) {
	_, a, err := auth.ReferenceForRepoName(f.keychain, ref)
	if err != nil {
		return "", errors.Wrapf(err, "resolve auth for ref %s", ref)
	}
	authConfig, err := a.Authorization()
	if err != nil {
		return "", err
	}

	dataJSON, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(dataJSON), nil
}

type colorizedWriter struct {
	writer io.Writer
}

type colorFunc = func(string, ...interface{}) string

func (w *colorizedWriter) Write(p []byte) (n int, err error) {
	msg := string(p)
	colorizers := map[string]colorFunc{
		"Waiting":           style.Waiting,
		"Pulling fs layer":  style.Waiting,
		"Downloading":       style.Working,
		"Download complete": style.Working,
		"Extracting":        style.Working,
		"Pull complete":     style.Complete,
		"Already exists":    style.Complete,
		"=":                 style.ProgressBar,
		">":                 style.ProgressBar,
	}
	for pattern, colorize := range colorizers {
		msg = strings.ReplaceAll(msg, pattern, colorize(pattern))
	}
	return w.writer.Write([]byte(msg))
}
