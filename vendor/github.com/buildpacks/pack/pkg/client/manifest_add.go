package client

import (
	"context"
	"fmt"

	"github.com/buildpacks/pack/internal/style"
)

type ManifestAddOptions struct {
	// Image index we want to update
	IndexRepoName string

	// Name of image we wish to add into the image index
	RepoName string
}

// AddManifest implements commands.PackClient.
func (c *Client) AddManifest(ctx context.Context, opts ManifestAddOptions) (err error) {
	idx, err := c.indexFactory.LoadIndex(opts.IndexRepoName)
	if err != nil {
		return err
	}

	if err = c.addManifestToIndex(ctx, opts.RepoName, idx); err != nil {
		return err
	}

	if err = idx.SaveDir(); err != nil {
		return fmt.Errorf("failed to save manifest list %s to local storage: %w", style.Symbol(opts.RepoName), err)
	}

	c.logger.Infof("Successfully added image %s to index", style.Symbol(opts.RepoName))
	return nil
}
