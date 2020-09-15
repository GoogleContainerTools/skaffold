package pack

import (
	"context"
	"os"
	"path/filepath"

	"github.com/buildpacks/imgutil"
	dockerClient "github.com/docker/docker/client"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/pkg/errors"

	pubcfg "github.com/buildpacks/pack/config"
	"github.com/buildpacks/pack/internal/blob"
	"github.com/buildpacks/pack/internal/build"
	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/image"
	"github.com/buildpacks/pack/logging"
)

//go:generate mockgen -package testmocks -destination testmocks/mock_docker_client.go github.com/docker/docker/client CommonAPIClient

//go:generate mockgen -package testmocks -destination testmocks/mock_image_fetcher.go github.com/buildpacks/pack ImageFetcher
type ImageFetcher interface {
	// Fetch fetches an image by resolving it both remotely and locally depending on provided parameters.
	// If daemon is true, it will look return a `local.Image`. Pull, applicable only when daemon is true, will
	// attempt to pull a remote image first.
	Fetch(ctx context.Context, name string, daemon bool, pullPolicy pubcfg.PullPolicy) (imgutil.Image, error)
}

//go:generate mockgen -package testmocks -destination testmocks/mock_downloader.go github.com/buildpacks/pack Downloader
type Downloader interface {
	Download(ctx context.Context, pathOrURI string) (blob.Blob, error)
}

//go:generate mockgen -package testmocks -destination testmocks/mock_image_factory.go github.com/buildpacks/pack ImageFactory
type ImageFactory interface {
	NewImage(repoName string, local bool) (imgutil.Image, error)
}

type Client struct {
	logger            logging.Logger
	imageFetcher      ImageFetcher
	downloader        Downloader
	lifecycleExecutor LifecycleExecutor
	docker            dockerClient.CommonAPIClient
	imageFactory      ImageFactory
	experimental      bool
}

type ClientOption func(c *Client)

// WithLogger supply your own logger.
func WithLogger(l logging.Logger) ClientOption {
	return func(c *Client) {
		c.logger = l
	}
}

// WithImageFactory supply your own image factory.
func WithImageFactory(f ImageFactory) ClientOption {
	return func(c *Client) {
		c.imageFactory = f
	}
}

// WithFetcher supply your own fetcher.
func WithFetcher(f ImageFetcher) ClientOption {
	return func(c *Client) {
		c.imageFetcher = f
	}
}

// WithDownloader supply your own downloader.
func WithDownloader(d Downloader) ClientOption {
	return func(c *Client) {
		c.downloader = d
	}
}

// WithCacheDir supply your own cache directory.
//
// Deprecated: use WithDownloader instead.
func WithCacheDir(path string) ClientOption {
	return func(c *Client) {
		c.downloader = blob.NewDownloader(c.logger, path)
	}
}

// WithDockerClient supply your own docker client.
func WithDockerClient(docker dockerClient.CommonAPIClient) ClientOption {
	return func(c *Client) {
		c.docker = docker
	}
}

// WithExperimental sets whether experimental features should be enabled
func WithExperimental(experimental bool) ClientOption {
	return func(c *Client) {
		c.experimental = experimental
	}
}

func NewClient(opts ...ClientOption) (*Client, error) {
	var client Client

	for _, opt := range opts {
		opt(&client)
	}

	if client.logger == nil {
		client.logger = logging.New(os.Stderr)
	}

	if client.docker == nil {
		var err error
		client.docker, err = dockerClient.NewClientWithOpts(
			dockerClient.FromEnv,
			dockerClient.WithVersion("1.38"),
		)
		if err != nil {
			return nil, errors.Wrap(err, "creating docker client")
		}
	}

	if client.downloader == nil {
		packHome, err := config.PackHome()
		if err != nil {
			return nil, errors.Wrap(err, "getting pack home")
		}
		client.downloader = blob.NewDownloader(client.logger, filepath.Join(packHome, "download-cache"))
	}

	if client.imageFetcher == nil {
		client.imageFetcher = image.NewFetcher(client.logger, client.docker)
	}

	if client.imageFactory == nil {
		client.imageFactory = image.NewFactory(client.docker, authn.DefaultKeychain)
	}

	client.lifecycleExecutor = build.NewLifecycleExecutor(client.logger, client.docker)

	return &client, nil
}
