package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"

	"github.com/buildpacks/imgutil"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"
)

type CachingImage struct {
	imgutil.Image
	cache *VolumeCache
}

func NewCachingImage(image imgutil.Image, cache *VolumeCache) imgutil.Image {
	return &CachingImage{
		Image: image,
		cache: cache,
	}
}

func (c *CachingImage) AddLayer(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return errors.Wrap(err, "opening layer file")
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return errors.Wrap(err, "hashing layer")
	}
	diffID := "sha256:" + hex.EncodeToString(hasher.Sum(make([]byte, 0, hasher.Size())))
	return c.AddLayerWithDiffID(path, diffID)
}

func (c *CachingImage) AddLayerWithDiffID(path string, diffID string) error {
	if err := c.cache.AddLayerFile(path, diffID); err != nil {
		return err
	}

	return c.Image.AddLayerWithDiffID(path, diffID)
}

func (c *CachingImage) AddLayerWithDiffIDAndHistory(path string, diffID string, history v1.History) error {
	if err := c.cache.AddLayerFile(path, diffID); err != nil {
		return err
	}

	return c.Image.AddLayerWithDiffIDAndHistory(path, diffID, history)
}

// cacheHit reports whether diffID can be served from the cache. A launch-cache hit is
// served from a directory the build user can write, so the contents are re-hashed
// before they are trusted; on a mismatch the cache is treated as a miss and the layer
// is re-fetched from the image.
func (c *CachingImage) cacheHit(diffID string) bool {
	found, err := c.cache.HasLayer(diffID)
	if err != nil {
		c.logf("Skipping cache for layer %s: %s", diffID, err)
		return false
	}
	if !found {
		return false
	}
	if err := c.cache.VerifyLayer(diffID); err != nil {
		c.logf("Skipping cache for layer %s: %s", diffID, err)
		return false
	}
	return true
}

func (c *CachingImage) logf(format string, v ...interface{}) {
	if c.cache.logger != nil {
		c.cache.logger.Warnf(format, v...)
	}
}

// ReuseLayer adds the layer from the cache on a cache hit; otherwise it reuses the layer from the image and adds it to the cache.
func (c *CachingImage) ReuseLayer(diffID string) error {
	if c.cacheHit(diffID) {
		if err := c.cache.ReuseLayer(diffID); err != nil {
			return err
		}
		path, err := c.cache.RetrieveLayerFile(diffID)
		if err != nil {
			return err
		}
		return c.Image.AddLayerWithDiffID(path, diffID)
	}

	if err := c.Image.ReuseLayer(diffID); err != nil {
		return err
	}
	rc, err := c.Image.GetLayer(diffID)
	if err != nil {
		return err
	}
	return c.cache.AddLayer(rc, diffID)
}

func (c *CachingImage) ReuseLayerWithHistory(diffID string, history v1.History) error {
	if c.cacheHit(diffID) {
		if err := c.cache.ReuseLayer(diffID); err != nil {
			return err
		}
		path, err := c.cache.RetrieveLayerFile(diffID)
		if err != nil {
			return err
		}
		return c.Image.AddLayerWithDiffIDAndHistory(path, diffID, history)
	}

	if err := c.Image.ReuseLayerWithHistory(diffID, history); err != nil {
		return err
	}
	rc, err := c.Image.GetLayer(diffID)
	if err != nil {
		return err
	}
	return c.cache.AddLayer(rc, diffID)
}

func (c *CachingImage) GetLayer(diffID string) (io.ReadCloser, error) {
	if c.cacheHit(diffID) {
		return c.cache.RetrieveLayer(diffID)
	}
	return c.Image.GetLayer(diffID)
}

func (c *CachingImage) Save(additionalNames ...string) error {
	err := c.Image.Save(additionalNames...)

	if saveSucceededFor(c.Name(), err) {
		if err := c.cache.Commit(); err != nil {
			return errors.Wrap(err, "failed to commit cache")
		}
	}
	return err
}

func (c *CachingImage) SaveAs(name string, additionalNames ...string) error {
	err := c.Image.SaveAs(name, additionalNames...)

	if saveSucceededFor(c.Name(), err) {
		if err := c.cache.Commit(); err != nil {
			return errors.Wrap(err, "failed to commit cache")
		}
	}
	return err
}

func saveSucceededFor(imageName string, err error) bool {
	if err == nil {
		return true
	}

	if saveErr, isSaveErr := err.(imgutil.SaveError); isSaveErr {
		for _, d := range saveErr.Errors {
			if d.ImageName == imageName {
				return false
			}
		}
		return true
	}
	return false
}
