package client

import (
	"errors"
	"fmt"

	gccrName "github.com/google/go-containerregistry/pkg/name"
)

// RemoveManifest implements commands.PackClient.
func (c *Client) RemoveManifest(name string, images []string) error {
	var allErrors error

	imgIndex, err := c.indexFactory.LoadIndex(name)
	if err != nil {
		return err
	}

	for _, image := range images {
		ref, err := gccrName.NewDigest(image, gccrName.WeakValidation, gccrName.Insecure)
		if err != nil {
			allErrors = errors.Join(allErrors, fmt.Errorf("invalid instance '%s': %w", image, err))
		}

		if err = imgIndex.RemoveManifest(ref); err != nil {
			allErrors = errors.Join(allErrors, err)
		}

		if err = imgIndex.SaveDir(); err != nil {
			allErrors = errors.Join(allErrors, err)
		}
	}

	if allErrors == nil {
		c.logger.Infof("Successfully removed image(s) from index: '%s'", name)
	}

	return allErrors
}
