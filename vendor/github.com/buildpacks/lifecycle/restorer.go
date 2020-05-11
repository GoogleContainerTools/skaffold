package lifecycle

import (
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/buildpacks/lifecycle/archive"
)

type Restorer struct {
	LayersDir  string
	Buildpacks []Buildpack
	Logger     Logger
}

// Restore attempts to restore layer data for cache=true layers, removing the layer when unsuccessful.
// If a usable cache is not provided, Restore will remove all cache=true layer metadata.
func (r *Restorer) Restore(cache Cache) error {
	// Create empty cache metadata in case a usable cache is not provided.
	var meta CacheMetadata
	if cache != nil {
		var err error
		meta, err = cache.RetrieveMetadata()
		if err != nil {
			return errors.Wrapf(err, "retrieving cache metadata")
		}
	} else {
		r.Logger.Debug("Usable cache not provided, using empty cache metadata.")
	}

	var g errgroup.Group
	for _, buildpack := range r.Buildpacks {
		buildpackDir, err := readBuildpackLayersDir(r.LayersDir, buildpack)
		if err != nil {
			return errors.Wrapf(err, "reading buildpack layer directory")
		}

		cachedLayers := meta.MetadataForBuildpack(buildpack.ID).Layers
		for _, bpLayer := range buildpackDir.findLayers(forCached) {
			name := bpLayer.name()
			cachedLayer, exists := cachedLayers[name]
			if !exists {
				r.Logger.Infof("Removing %q, not in cache", bpLayer.Identifier())
				if err := bpLayer.remove(); err != nil {
					return errors.Wrapf(err, "removing layer")
				}
				continue
			}
			data, err := bpLayer.read()
			if err != nil {
				return errors.Wrapf(err, "reading layer")
			}
			if data.SHA != cachedLayer.SHA {
				r.Logger.Infof("Removing %q, wrong sha", bpLayer.Identifier())
				r.Logger.Debugf("Layer sha: %q, cache sha: %q", data.SHA, cachedLayer.SHA)
				if err := bpLayer.remove(); err != nil {
					return errors.Wrapf(err, "removing layer")
				}
			} else {
				r.Logger.Infof("Restoring data for %q from cache", bpLayer.Identifier())
				g.Go(func() error {
					return r.restoreLayer(cache, cachedLayer.SHA)
				})
			}
		}
	}
	if err := g.Wait(); err != nil {
		return errors.Wrap(err, "restoring data")
	}
	return nil
}

func (r *Restorer) restoreLayer(cache Cache, sha string) error {
	// Sanity check to prevent panic.
	if cache == nil {
		return errors.New("restoring layer: cache not provided")
	}
	r.Logger.Debugf("Retrieving data for %q", sha)
	rc, err := cache.RetrieveLayer(sha)
	if err != nil {
		return err
	}
	defer rc.Close()

	return archive.Untar(rc, "/")
}
