package client

import (
	"context"
	"fmt"

	"github.com/buildpacks/imgutil"
	"github.com/google/go-containerregistry/pkg/v1/types"

	"github.com/buildpacks/pack/internal/style"
)

type CreateManifestOptions struct {
	// Image index we want to create
	IndexRepoName string

	// Name of images we wish to add into the image index
	RepoNames []string

	// Media type of the index
	Format types.MediaType

	// true if we want to publish to an insecure registry
	Insecure bool

	// true if we want to push the index to a registry after creating
	Publish bool
}

// CreateManifest implements commands.PackClient.
func (c *Client) CreateManifest(ctx context.Context, opts CreateManifestOptions) (err error) {
	ops := parseOptsToIndexOptions(opts)

	if c.indexFactory.Exists(opts.IndexRepoName) {
		return fmt.Errorf("manifest list '%s' already exists in local storage; use 'pack manifest remove' to "+
			"remove it before creating a new manifest list with the same name", style.Symbol(opts.IndexRepoName))
	}

	index, err := c.indexFactory.CreateIndex(opts.IndexRepoName, ops...)
	if err != nil {
		return err
	}

	for _, repoName := range opts.RepoNames {
		if err = c.addManifestToIndex(ctx, repoName, index); err != nil {
			return err
		}
	}

	if opts.Publish {
		// push to a registry without saving a local copy
		ops = append(ops, imgutil.WithPurge(true))
		if err = index.Push(ops...); err != nil {
			return err
		}

		c.logger.Infof("Successfully pushed manifest list %s to registry", style.Symbol(opts.IndexRepoName))
		return nil
	}

	if err = index.SaveDir(); err != nil {
		return fmt.Errorf("manifest list %s could not be saved to local storage: %w", style.Symbol(opts.IndexRepoName), err)
	}

	c.logger.Infof("Successfully created manifest list %s", style.Symbol(opts.IndexRepoName))
	return nil
}

func parseOptsToIndexOptions(opts CreateManifestOptions) (idxOpts []imgutil.IndexOption) {
	if opts.Insecure {
		return []imgutil.IndexOption{
			imgutil.WithMediaType(opts.Format),
			imgutil.WithInsecure(),
		}
	}
	if opts.Format == "" {
		opts.Format = types.OCIImageIndex
	}
	return []imgutil.IndexOption{
		imgutil.WithMediaType(opts.Format),
	}
}
