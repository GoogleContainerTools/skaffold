package client

import (
	"fmt"

	"github.com/buildpacks/imgutil"
	"github.com/google/go-containerregistry/pkg/v1/types"

	"github.com/buildpacks/pack/internal/style"
)

type PushManifestOptions struct {
	// Image index we want to update
	IndexRepoName string

	// Index media-type
	Format types.MediaType

	// true if we want to publish to an insecure registry
	Insecure bool

	// true if we want the index to be deleted from local storage after pushing it
	Purge bool
}

// PushManifest implements commands.PackClient.
func (c *Client) PushManifest(opts PushManifestOptions) (err error) {
	if opts.Format == "" {
		opts.Format = types.OCIImageIndex
	}
	ops := parseOptions(opts)

	idx, err := c.indexFactory.LoadIndex(opts.IndexRepoName)
	if err != nil {
		return
	}

	if err = idx.Push(ops...); err != nil {
		return fmt.Errorf("failed to push manifest list %s: %w", style.Symbol(opts.IndexRepoName), err)
	}

	if !opts.Purge {
		c.logger.Infof("Successfully pushed manifest list %s to registry", style.Symbol(opts.IndexRepoName))
		return nil
	}

	return idx.DeleteDir()
}

func parseOptions(opts PushManifestOptions) (idxOptions []imgutil.IndexOption) {
	if opts.Insecure {
		idxOptions = append(idxOptions, imgutil.WithInsecure())
	}

	if opts.Purge {
		idxOptions = append(idxOptions, imgutil.WithPurge(true))
	}

	return append(idxOptions, imgutil.WithMediaType(opts.Format))
}
