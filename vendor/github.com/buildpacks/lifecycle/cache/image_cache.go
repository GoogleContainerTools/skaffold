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
	committed    bool
	origImage    imgutil.Image
	newImage     imgutil.Image
	logger       log.Logger
	imageDeleter ImageDeleter
}

// NewImageCache creates a new ImageCache instance
func NewImageCache(origImage imgutil.Image, newImage imgutil.Image, logger log.Logger, imageDeleter ImageDeleter) *ImageCache {
	return &ImageCache{
		origImage:    origImage,
		newImage:     newImage,
		logger:       logger,
		imageDeleter: imageDeleter,
	}
}

// NewImageCacheFromName creates a new ImageCache from the name that has been provided
func NewImageCacheFromName(name string, keychain authn.Keychain, logger log.Logger, imageDeleter ImageDeleter) (*ImageCache, error) {
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

	return NewImageCache(origImage, emptyImage, logger, imageDeleter), nil
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
	if c.origImage.Found() && !c.origImage.Valid() {
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

// isLayerNotFound checks if the error is a layer not found error
//
// FIXME: we should not have to rely on trapping ErrUnexpectedEOF.
// If a blob is not present in the registry, we should get imgutil.ErrLayerNotFound,
// but we do not and instead get io.ErrUnexpectedEOF
func isLayerNotFound(err error) bool {
	var e imgutil.ErrLayerNotFound
	return errors.As(err, &e) || errors.Is(err, io.ErrUnexpectedEOF)
}

func (c *ImageCache) ReuseLayer(diffID string) error {
	if c.committed {
		return errCacheCommitted
	}
	err := c.newImage.ReuseLayer(diffID)
	if err != nil {
		// FIXME: this path is not currently executed.
		// If a blob is not present in the registry, we should get imgutil.ErrLayerNotFound.
		// We should then skip attempting to reuse the layer.
		// However, we do not get imgutil.ErrLayerNotFound when the blob is not present.
		if isLayerNotFound(err) {
			return NewReadErr(fmt.Sprintf("failed to find cache layer with SHA '%s'", diffID))
		}
		return fmt.Errorf("failed to reuse cache layer with SHA '%s'", diffID)
	}
	return nil
}

// RetrieveLayer retrieves a layer from the cache
func (c *ImageCache) RetrieveLayer(diffID string) (io.ReadCloser, error) {
	closer, err := c.origImage.GetLayer(diffID)
	if err != nil {
		if isLayerNotFound(err) {
			return nil, NewReadErr(fmt.Sprintf("failed to find cache layer with SHA '%s'", diffID))
		}
		return nil, fmt.Errorf("failed to get cache layer with SHA '%s'", diffID)
	}
	return closer, nil
}

func (c *ImageCache) Commit() error {
	if c.committed {
		return errCacheCommitted
	}

	if err := c.newImage.Save(); err != nil {
		return errors.Wrapf(err, "saving image '%s'", c.newImage.Name())
	}
	c.committed = true

	// Check if the cache image exists prior to saving the new cache at that same location
	if c.origImage.Found() {
		c.imageDeleter.DeleteOrigImageIfDifferentFromNewImage(c.origImage, c.newImage)
	}

	c.origImage = c.newImage

	return nil
}
