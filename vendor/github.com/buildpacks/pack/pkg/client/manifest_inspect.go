package client

import (
	"fmt"

	"github.com/buildpacks/imgutil"
)

// InspectManifest implements commands.PackClient.
func (c *Client) InspectManifest(indexRepoName string) error {
	var (
		index    imgutil.ImageIndex
		indexStr string
		err      error
	)

	index, err = c.indexFactory.FindIndex(indexRepoName)
	if err != nil {
		return err
	}

	if indexStr, err = index.Inspect(); err != nil {
		return fmt.Errorf("failed to inspect manifest list '%s': %w", indexRepoName, err)
	}

	c.logger.Info(indexStr)
	return nil
}
