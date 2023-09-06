package cache

import (
	"encoding/json"
	"fmt"
	"io"
	"runtime"

	"github.com/buildpacks/imgutil"
	"github.com/buildpacks/imgutil/remote"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/pkg/errors"

	"github.com/buildpacks/lifecycle/image"
	"github.com/buildpacks/lifecycle/log"
	"github.com/buildpacks/lifecycle/platform"
)

const MetadataLabel = "io.buildpacks.lifecycle.cache.metadata"

type ImageCache struct {
	committed bool
	origImage imgutil.Image
	newImage  imgutil.Image
	logger    log.Logger
}

func NewImageCache(origImage imgutil.Image, newImage imgutil.Image, logger log.Logger) *ImageCache {
	return &ImageCache{
		origImage: origImage,
		newImage:  newImage,
		logger:    logger,
	}
}

func NewImageCacheFromName(name string, keychain authn.Keychain, logger log.Logger) (*ImageCache, error) {
	origImage, err := remote.NewImage(
		name,
		keychain,
		remote.FromBaseImage(name),
		remote.WithDefaultPlatform(imgutil.Platform{OS: runtime.GOOS}),
	)
	if err != nil {
		return nil, fmt.Errorf("accessing cache image %q: %v", name, err)
	}
	emptyImage, err := remote.NewImage(
		name,
		keychain,
		remote.WithPreviousImage(name),
		remote.WithDefaultPlatform(imgutil.Platform{OS: runtime.GOOS}),
		remote.AddEmptyLayerOnSave(),
	)
	if err != nil {
		return nil, fmt.Errorf("creating new cache image %q: %v", name, err)
	}

	return NewImageCache(origImage, emptyImage, logger), nil
}

func (c *ImageCache) Exists() bool {
	return c.origImage.Found()
}

func (c *ImageCache) Name() string {
	return c.origImage.Name()
}

func (c *ImageCache) SetMetadata(metadata platform.CacheMetadata) error {
	if c.committed {
		return errCacheCommitted
	}
	data, err := json.Marshal(metadata)
	if err != nil {
		return errors.Wrap(err, "serializing metadata")
	}
	return c.newImage.SetLabel(MetadataLabel, string(data))
}

func (c *ImageCache) RetrieveMetadata() (platform.CacheMetadata, error) {
	if !c.origImage.Valid() {
		c.logger.Infof("Ignoring cache image %q because it was corrupt", c.origImage.Name())
		return platform.CacheMetadata{}, nil
	}
	var meta platform.CacheMetadata
	if err := image.DecodeLabel(c.origImage, MetadataLabel, &meta); err != nil {
		return platform.CacheMetadata{}, nil
	}
	return meta, nil
}

func (c *ImageCache) AddLayerFile(tarPath string, diffID string) error {
	if c.committed {
		return errCacheCommitted
	}
	return c.newImage.AddLayerWithDiffID(tarPath, diffID)
}

func (c *ImageCache) ReuseLayer(diffID string) error {
	if c.committed {
		return errCacheCommitted
	}
	return c.newImage.ReuseLayer(diffID)
}

func (c *ImageCache) RetrieveLayer(diffID string) (io.ReadCloser, error) {
	return c.origImage.GetLayer(diffID)
}

func (c *ImageCache) Commit() error {
	if c.committed {
		return errCacheCommitted
	}

	// Check if the cache image exists prior to saving the new cache at that same location
	origImgExists := c.origImage.Found()

	if err := c.newImage.Save(); err != nil {
		return errors.Wrapf(err, "saving image '%s'", c.newImage.Name())
	}
	c.committed = true

	if origImgExists {
		// Deleting the original image is for cleanup only and should not fail the commit.
		if err := c.DeleteOrigImage(); err != nil {
			c.logger.Warnf("Unable to delete previous cache image: %v", err.Error())
		}
	}
	c.origImage = c.newImage

	return nil
}

func (c *ImageCache) DeleteOrigImage() error {
	origIdentifier, err := c.origImage.Identifier()
	if err != nil {
		return errors.Wrap(err, "getting identifier for original image")
	}
	newIdentifier, err := c.newImage.Identifier()
	if err != nil {
		return errors.Wrap(err, "getting identifier for new image")
	}
	if origIdentifier.String() == newIdentifier.String() {
		return nil
	}
	return c.origImage.Delete()
}
