/*
Package client provides all the functionality provided by pack as a library through a go api.

# Prerequisites

In order to use most functionality, you will need an OCI runtime such as Docker or podman installed.

# References

This package provides functionality to create and manipulate all artifacts outlined in the Cloud Native Buildpacks specification.
An introduction to these artifacts and their usage can be found at https://buildpacks.io/docs/.

The formal specification of the pack platform provides can be found at: https://github.com/buildpacks/spec.
*/
package client

import (
	"context"
	"os"
	"path/filepath"

	"github.com/buildpacks/imgutil"
	"github.com/buildpacks/imgutil/local"
	"github.com/buildpacks/imgutil/remote"
	"github.com/google/go-containerregistry/pkg/authn"
	dockerClient "github.com/moby/moby/client"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/build"
	iconfig "github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/blob"
	"github.com/buildpacks/pack/pkg/buildpack"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/image"
	"github.com/buildpacks/pack/pkg/index"
	"github.com/buildpacks/pack/pkg/logging"
)

const (
	// Env variable to set the root folder for manifest list local storage
	xdgRuntimePath = "XDG_RUNTIME_DIR"
)

var (
	// Version is the version of `pack`. It is injected at compile time.
	Version = "0.0.0"
)

//go:generate mockgen -package testmocks -destination ../testmocks/mock_docker_client.go github.com/moby/moby/client APIClient

//go:generate mockgen -package testmocks -destination ../testmocks/mock_image_fetcher.go github.com/buildpacks/pack/pkg/client ImageFetcher

// ImageFetcher is an interface representing the ability to fetch local and remote images.
type ImageFetcher interface {
	// Fetch fetches an image by resolving it both remotely and locally depending on provided parameters.
	// The pull behavior is dictated by the pullPolicy, which can have the following behavior
	//   - PullNever: try to use the daemon to return a `local.Image`.
	//   - PullIfNotPResent: try look to use the daemon to return a `local.Image`, if none is found  fetch a remote image.
	//   - PullAlways: it will only try to fetch a remote image.
	//
	// These PullPolicies that these interact with the daemon argument.
	// PullIfNotPresent and daemon = false, gives us the same behavior as PullAlways.
	// There is a single invalid configuration, PullNever and daemon = false, this will always fail.
	Fetch(ctx context.Context, name string, options image.FetchOptions) (imgutil.Image, error)

	// CheckReadAccess verifies if an image is accessible with read permissions
	// When FetchOptions.Daemon is true and the image doesn't exist in the daemon,
	// the behavior is dictated by the pull policy, which can have the following behavior
	//   - PullNever: returns false
	//   - PullAlways Or PullIfNotPresent: it will check read access for the remote image.
	// When FetchOptions.Daemon is false it will check read access for the remote image.
	CheckReadAccess(repo string, options image.FetchOptions) bool

	// FetchForPlatform fetches an image and resolves it to a platform-specific digest before fetching.
	// This ensures that multi-platform images are always resolved to the correct platform-specific manifest.
	FetchForPlatform(ctx context.Context, name string, options image.FetchOptions) (imgutil.Image, error)
}

//go:generate mockgen -package testmocks -destination ../testmocks/mock_blob_downloader.go github.com/buildpacks/pack/pkg/client BlobDownloader

// BlobDownloader is an interface for collecting both remote and local assets as blobs.
type BlobDownloader interface {
	// Download collects both local and remote assets and provides a blob object
	// used to read asset contents.
	Download(ctx context.Context, pathOrURI string) (blob.Blob, error)
}

//go:generate mockgen -package testmocks -destination ../testmocks/mock_image_factory.go github.com/buildpacks/pack/pkg/client ImageFactory

// ImageFactory is an interface representing the ability to create a new OCI image.
type ImageFactory interface {
	// NewImage initializes an image object with required settings so that it
	// can be written either locally or to a registry.
	NewImage(repoName string, local bool, target dist.Target) (imgutil.Image, error)
}

//go:generate mockgen -package testmocks -destination ../testmocks/mock_index_factory.go github.com/buildpacks/pack/pkg/client IndexFactory

// IndexFactory is an interface representing the ability to create a ImageIndex/ManifestList.
type IndexFactory interface {
	// Exists return true if the given index exits in the local storage
	Exists(repoName string) bool
	// CreateIndex creates ManifestList locally
	CreateIndex(repoName string, opts ...imgutil.IndexOption) (imgutil.ImageIndex, error)
	// LoadIndex loads ManifestList from local storage with the given name
	LoadIndex(reponame string, opts ...imgutil.IndexOption) (imgutil.ImageIndex, error)
	// FetchIndex fetches ManifestList from Registry with the given name
	FetchIndex(name string, opts ...imgutil.IndexOption) (imgutil.ImageIndex, error)
	// FindIndex will find Index locally then on remote
	FindIndex(name string, opts ...imgutil.IndexOption) (imgutil.ImageIndex, error)
}

//go:generate mockgen -package testmocks -destination ../testmocks/mock_buildpack_downloader.go github.com/buildpacks/pack/pkg/client BuildpackDownloader

// BuildpackDownloader is an interface for downloading and extracting buildpacks from various sources
type BuildpackDownloader interface {
	// Download parses a buildpack URI and downloads the buildpack and any dependencies buildpacks from the appropriate source
	Download(ctx context.Context, buildpackURI string, opts buildpack.DownloadOptions) (buildpack.BuildModule, []buildpack.BuildModule, error)
}

// Client is an orchestration object, it contains all parameters needed to
// build an app image using Cloud Native Buildpacks.
// All settings on this object should be changed through ClientOption functions.
type Client struct {
	logger logging.Logger
	docker DockerClient

	keychain            authn.Keychain
	imageFactory        ImageFactory
	imageFetcher        ImageFetcher
	indexFactory        IndexFactory
	downloader          BlobDownloader
	lifecycleExecutor   LifecycleExecutor
	buildpackDownloader BuildpackDownloader

	experimental    bool
	registryMirrors map[string]string
	version         string
}

func (c *Client) processSystem(system dist.System, buildpacks []buildpack.BuildModule, disableSystem bool) (dist.System, error) {
	if disableSystem {
		return dist.System{}, nil
	}

	if len(buildpacks) == 0 {
		return system, nil
	}

	resolved := dist.System{}

	// Create a map of available buildpacks for faster lookup
	availableBPs := make(map[string]bool)
	for _, bp := range buildpacks {
		bpInfo := bp.Descriptor().Info()
		availableBPs[bpInfo.ID+"@"+bpInfo.Version] = true
	}

	// Process pre-buildpacks
	for _, preBp := range system.Pre.Buildpacks {
		key := preBp.ID + "@" + preBp.Version
		if availableBPs[key] {
			resolved.Pre.Buildpacks = append(resolved.Pre.Buildpacks, preBp)
		} else if !preBp.Optional {
			return dist.System{}, errors.Errorf("required system buildpack %s@%s is not available", preBp.ID, preBp.Version)
		}
	}

	// Process post-buildpacks
	for _, postBp := range system.Post.Buildpacks {
		key := postBp.ID + "@" + postBp.Version
		if availableBPs[key] {
			resolved.Post.Buildpacks = append(resolved.Post.Buildpacks, postBp)
		} else if !postBp.Optional {
			return dist.System{}, errors.Errorf("required system buildpack %s@%s is not available", postBp.ID, postBp.Version)
		}
	}

	return resolved, nil
}

// Option is a type of function that mutate settings on the client.
// Values in these functions are set through currying.
type Option func(c *Client)

// WithLogger supply your own logger.
func WithLogger(l logging.Logger) Option {
	return func(c *Client) {
		c.logger = l
	}
}

// WithImageFactory supply your own image factory.
func WithImageFactory(f ImageFactory) Option {
	return func(c *Client) {
		c.imageFactory = f
	}
}

// WithIndexFactory supply your own index factory
func WithIndexFactory(f IndexFactory) Option {
	return func(c *Client) {
		c.indexFactory = f
	}
}

// WithFetcher supply your own Fetcher.
// A Fetcher retrieves both local and remote images to make them available.
func WithFetcher(f ImageFetcher) Option {
	return func(c *Client) {
		c.imageFetcher = f
	}
}

// WithDownloader supply your own downloader.
// A Downloader is used to gather buildpacks from both remote urls, or local sources.
func WithDownloader(d BlobDownloader) Option {
	return func(c *Client) {
		c.downloader = d
	}
}

// WithBuildpackDownloader supply your own BuildpackDownloader.
// A BuildpackDownloader is used to gather buildpacks from both remote urls, or local sources.
func WithBuildpackDownloader(d BuildpackDownloader) Option {
	return func(c *Client) {
		c.buildpackDownloader = d
	}
}

// Deprecated: use WithDownloader instead.
//
// WithCacheDir supply your own cache directory.
func WithCacheDir(path string) Option {
	return func(c *Client) {
		c.downloader = blob.NewDownloader(c.logger, path)
	}
}

// WithDockerClient supply your own docker client.
func WithDockerClient(docker DockerClient) Option {
	return func(c *Client) {
		c.docker = docker
	}
}

// WithExperimental sets whether experimental features should be enabled.
func WithExperimental(experimental bool) Option {
	return func(c *Client) {
		c.experimental = experimental
	}
}

// WithRegistryMirrors sets mirrors to pull images from.
func WithRegistryMirrors(registryMirrors map[string]string) Option {
	return func(c *Client) {
		c.registryMirrors = registryMirrors
	}
}

// WithKeychain sets keychain of credentials to image registries
func WithKeychain(keychain authn.Keychain) Option {
	return func(c *Client) {
		c.keychain = keychain
	}
}

const DockerAPIVersion = "1.38"

// NewClient allocates and returns a Client configured with the specified options.
func NewClient(opts ...Option) (*Client, error) {
	client := &Client{
		version:  Version,
		keychain: authn.DefaultKeychain,
	}

	for _, opt := range opts {
		opt(client)
	}

	if client.logger == nil {
		client.logger = logging.NewSimpleLogger(os.Stderr)
	}

	if client.docker == nil {
		var err error
		client.docker, err = dockerClient.New(
			dockerClient.FromEnv,
		)
		if err != nil {
			return nil, errors.Wrap(err, "creating docker client")
		}
	}

	if client.downloader == nil {
		packHome, err := iconfig.PackHome()
		if err != nil {
			return nil, errors.Wrap(err, "getting pack home")
		}
		client.downloader = blob.NewDownloader(client.logger, filepath.Join(packHome, "download-cache"))
	}

	if client.imageFetcher == nil {
		client.imageFetcher = image.NewFetcher(client.logger, client.docker, image.WithRegistryMirrors(client.registryMirrors), image.WithKeychain(client.keychain))
	}

	if client.imageFactory == nil {
		client.imageFactory = &imageFactory{
			dockerClient: client.docker,
			keychain:     client.keychain,
		}
	}

	if client.indexFactory == nil {
		packHome, err := iconfig.PackHome()
		if err != nil {
			return nil, errors.Wrap(err, "getting pack home")
		}
		indexRootStoragePath := filepath.Join(packHome, "manifests")
		if xdgPath, ok := os.LookupEnv(xdgRuntimePath); ok {
			indexRootStoragePath = xdgPath
		}
		client.indexFactory = index.NewIndexFactory(client.keychain, indexRootStoragePath)
	}

	if client.buildpackDownloader == nil {
		client.buildpackDownloader = buildpack.NewDownloader(
			client.logger,
			client.imageFetcher,
			client.downloader,
			&registryResolver{
				logger: client.logger,
			},
		)
	}

	client.lifecycleExecutor = build.NewLifecycleExecutor(client.logger, client.docker)

	return client, nil
}

type registryResolver struct {
	logger logging.Logger
}

func (r *registryResolver) Resolve(registryName, bpName string) (string, error) {
	cache, err := getRegistry(r.logger, registryName)
	if err != nil {
		return "", errors.Wrapf(err, "lookup registry %s", style.Symbol(registryName))
	}

	regBuildpack, err := cache.LocateBuildpack(bpName)
	if err != nil {
		return "", errors.Wrapf(err, "lookup buildpack %s", style.Symbol(bpName))
	}

	return regBuildpack.Address, nil
}

type imageFactory struct {
	dockerClient local.DockerClient
	keychain     authn.Keychain
}

func (f *imageFactory) NewImage(repoName string, daemon bool, target dist.Target) (imgutil.Image, error) {
	platform := imgutil.Platform{OS: target.OS, Architecture: target.Arch, Variant: target.ArchVariant}

	if len(target.Distributions) > 0 {
		// We need to set platform distribution information so that it will be reflected in the image config.
		// We assume the given target's distributions were already expanded, we should be dealing with just 1 distribution name and version.
		platform.OSVersion = target.Distributions[0].Version
	}

	if daemon {
		return local.NewImage(repoName, f.dockerClient, local.WithDefaultPlatform(platform))
	}

	return remote.NewImage(repoName, f.keychain, remote.WithDefaultPlatform(platform))
}
