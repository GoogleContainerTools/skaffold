package client

import "errors"

// DeleteManifest implements commands.PackClient.
func (c *Client) DeleteManifest(names []string) error {
	var allErrors error
	for _, name := range names {
		imgIndex, err := c.indexFactory.LoadIndex(name)
		if err != nil {
			allErrors = errors.Join(allErrors, err)
			continue
		}

		if err := imgIndex.DeleteDir(); err != nil {
			allErrors = errors.Join(allErrors, err)
		}
	}

	if allErrors == nil {
		c.logger.Info("Successfully deleted manifest list(s) from local storage")
	}
	return allErrors
}
