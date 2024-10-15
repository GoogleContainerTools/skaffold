package client

import (
	"context"

	"github.com/buildpacks/lifecycle/layers"
	"github.com/buildpacks/lifecycle/platform"
	"github.com/buildpacks/lifecycle/platform/files"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/image"
)

type DownloadSBOMOptions struct {
	Daemon         bool
	DestinationDir string
}

// Deserialize just the subset of fields we need to avoid breaking changes
type sbomMetadata struct {
	BOM *files.LayerMetadata `json:"sbom" toml:"sbom"`
}

func (s *sbomMetadata) isMissing() bool {
	return s == nil ||
		s.BOM == nil ||
		s.BOM.SHA == ""
}

const (
	Local = iota
	Remote
)

// DownloadSBOM pulls SBOM layer from an image.
// It reads the SBOM metadata of an image then
// pulls the corresponding diffId, if it exists
func (c *Client) DownloadSBOM(name string, options DownloadSBOMOptions) error {
	img, err := c.imageFetcher.Fetch(context.Background(), name, image.FetchOptions{Daemon: options.Daemon, PullPolicy: image.PullNever})
	if err != nil {
		if errors.Cause(err) == image.ErrNotFound {
			c.logger.Warnf("if the image is saved on a registry run with the flag '--remote', for example: 'pack sbom download --remote %s'", name)
			return errors.Wrapf(image.ErrNotFound, "image '%s' cannot be found", name)
		}
		return err
	}

	var sbomMD sbomMetadata
	if _, err := dist.GetLabel(img, platform.LifecycleMetadataLabel, &sbomMD); err != nil {
		return err
	}

	if sbomMD.isMissing() {
		return errors.Errorf("could not find SBoM information on '%s'", name)
	}

	rc, err := img.GetLayer(sbomMD.BOM.SHA)
	if err != nil {
		return err
	}
	defer rc.Close()

	return layers.Extract(rc, options.DestinationDir)
}
