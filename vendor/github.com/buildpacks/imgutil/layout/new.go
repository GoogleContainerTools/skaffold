package layout

import (
	"fmt"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/pkg/errors"

	"github.com/buildpacks/imgutil"
)

func NewImage(path string, ops ...ImageOption) (*Image, error) {
	imageOpts := &options{}
	for _, op := range ops {
		if err := op(imageOpts); err != nil {
			return nil, err
		}
	}

	platform := defaultPlatform()
	if (imageOpts.platform != imgutil.Platform{}) {
		platform = imageOpts.platform
	}

	image, err := emptyImage(platform)
	if err != nil {
		return nil, err
	}

	ri := &Image{
		Image:       image,
		path:        path,
		withHistory: imageOpts.withHistory,
	}

	if imageOpts.prevImagePath != "" {
		if err := processPreviousImageOption(ri, imageOpts.prevImagePath, platform); err != nil {
			return nil, err
		}
	}

	if imageOpts.baseImagePath != "" {
		if err := processBaseImageOption(ri, imageOpts.baseImagePath, platform); err != nil {
			return nil, err
		}
	} else if imageOpts.baseImage != nil {
		if err := ri.setUnderlyingImage(imageOpts.baseImage); err != nil {
			return nil, err
		}
	}

	if imageOpts.createdAt.IsZero() {
		ri.createdAt = imgutil.NormalizedDateTime
	} else {
		ri.createdAt = imageOpts.createdAt
	}

	if imageOpts.mediaTypes == imgutil.MissingTypes {
		ri.requestedMediaTypes = imgutil.OCITypes
	} else {
		ri.requestedMediaTypes = imageOpts.mediaTypes
	}
	if err = ri.setUnderlyingImage(ri.Image); err != nil { // update media types
		return nil, err
	}

	return ri, nil
}

func defaultPlatform() imgutil.Platform {
	return imgutil.Platform{
		OS:           "linux",
		Architecture: "amd64",
	}
}

func emptyImage(platform imgutil.Platform) (v1.Image, error) {
	cfg := &v1.ConfigFile{
		Architecture: platform.Architecture,
		History:      []v1.History{},
		OS:           platform.OS,
		OSVersion:    platform.OSVersion,
		RootFS: v1.RootFS{
			Type:    "layers",
			DiffIDs: []v1.Hash{},
		},
	}
	image := mutate.MediaType(empty.Image, types.OCIManifestSchema1)
	image = mutate.ConfigMediaType(image, types.OCIConfigJSON)
	return mutate.ConfigFile(image, cfg)
}

func processPreviousImageOption(ri *Image, prevImagePath string, platform imgutil.Platform) error {
	prevImage, err := newV1Image(prevImagePath, platform, ri.withHistory)
	if err != nil {
		return err
	}

	prevLayers, err := prevImage.Layers()
	if err != nil {
		return errors.Wrapf(err, "getting layers for previous image with path %q", prevImagePath)
	}

	ri.prevLayers = prevLayers
	configFile, err := prevImage.ConfigFile()
	if err != nil {
		return err
	}
	ri.prevHistory = configFile.History

	return nil
}

// newV1Image creates a layout image from the given path.
//   - If a ImageIndex for multiples platforms exists, then it will try to select the image
//     according to the platform provided
//   - If the image does not exist, then an empty image is returned
func newV1Image(path string, platform imgutil.Platform, withHistory bool) (v1.Image, error) {
	var (
		image  v1.Image
		layout Path
		err    error
	)

	if ImageExists(path) {
		layout, err = FromPath(path)
		if err != nil {
			return nil, fmt.Errorf("loading layout from path new: %w", err)
		}

		index, err := layout.ImageIndex()
		if err != nil {
			return nil, fmt.Errorf("reading index: %w", err)
		}

		image, err = imageFromIndex(index, platform)
		if err != nil {
			return nil, fmt.Errorf("getting image from index: %w", err)
		}
	} else {
		image, err = emptyImage(platform)
		if err != nil {
			return nil, fmt.Errorf("initializing empty image: %w", err)
		}
	}

	if withHistory {
		if image, err = imgutil.OverrideHistoryIfNeeded(image); err != nil {
			return nil, fmt.Errorf("overriding history: %w", err)
		}
	}

	return &Image{
		Image: image,
		path:  path,
	}, nil
}

// imageFromIndex creates a v1.Image from the given Image Index, selecting the image manifest
// that matches the given OS and architecture.
func imageFromIndex(index v1.ImageIndex, platform imgutil.Platform) (v1.Image, error) {
	indexManifest, err := index.IndexManifest()
	if err != nil {
		return nil, err
	}

	if len(indexManifest.Manifests) == 0 {
		return nil, errors.New("no underlyingImage indexManifest found")
	}

	manifest := indexManifest.Manifests[0]
	if len(indexManifest.Manifests) > 1 {
		// Find based on platform (os/arch)
		for _, m := range indexManifest.Manifests {
			if m.Platform.OS == platform.OS && m.Platform.Architecture == platform.OS {
				manifest = m
				break
			}
		}
		return nil, fmt.Errorf("manifest matching platform %v not found", platform)
	}

	image, err := index.Image(manifest.Digest)
	if err != nil {
		return nil, err
	}

	return image, nil
}

func processBaseImageOption(ri *Image, baseImagePath string, platform imgutil.Platform) error {
	baseImage, err := newV1Image(baseImagePath, platform, ri.withHistory)
	if err != nil {
		return err
	}

	return ri.setUnderlyingImage(baseImage)
}

// setUnderlyingImage wraps the provided v1.Image into a layout.Image and sets it as the underlying image for the receiving layout.Image
func (i *Image) setUnderlyingImage(base v1.Image) error {
	manifest, err := base.Manifest()
	if err != nil {
		return err
	}
	if i.requestedMediaTypesMatch(manifest) {
		i.Image = &Image{Image: base}
		return nil
	}
	// provided v1.Image media types differ from requested, override them
	newBase, err := imgutil.OverrideMediaTypes(base, i.requestedMediaTypes)
	if err != nil {
		return err
	}
	i.Image = &Image{Image: newBase}
	return nil
}

// requestedMediaTypesMatch returns true if the manifest and config file use the requested media types
func (i *Image) requestedMediaTypesMatch(manifest *v1.Manifest) bool {
	return manifest.MediaType == i.requestedMediaTypes.ManifestType() &&
		manifest.Config.MediaType == i.requestedMediaTypes.ConfigType()
}
