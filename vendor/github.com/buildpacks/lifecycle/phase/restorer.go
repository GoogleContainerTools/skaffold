package phase

import (
	"path/filepath"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	c "github.com/buildpacks/lifecycle/cache"

	"github.com/buildpacks/lifecycle/api"
	"github.com/buildpacks/lifecycle/buildpack"
	"github.com/buildpacks/lifecycle/internal/layer"
	"github.com/buildpacks/lifecycle/layers"
	"github.com/buildpacks/lifecycle/log"
	"github.com/buildpacks/lifecycle/platform"
	"github.com/buildpacks/lifecycle/platform/files"
)

type Restorer struct {
	LayersDir string
	Logger    log.Logger

	Buildpacks            []buildpack.GroupElement
	LayerMetadataRestorer layer.MetadataRestorer
	LayersMetadata        files.LayersMetadata
	PlatformAPI           *api.Version
	SBOMRestorer          layer.SBOMRestorer
}

// Restore restores metadata for launch and cache layers into the layers directory and attempts to restore layer data for cache=true layers, removing the layer when unsuccessful.
// If a usable cache is not provided, Restore will not restore any cache=true layer metadata.
func (r *Restorer) Restore(cache Cache) error {
	defer log.NewMeasurement("Restorer", r.Logger)()
	cacheMeta, err := retrieveCacheMetadata(cache, r.Logger)
	if err != nil {
		return err
	}

	if r.LayerMetadataRestorer == nil {
		r.LayerMetadataRestorer = layer.NewDefaultMetadataRestorer(r.LayersDir, false, r.Logger, r.PlatformAPI)
	}

	if r.SBOMRestorer == nil {
		r.SBOMRestorer = layer.NewSBOMRestorer(layer.SBOMRestorerOpts{
			LayersDir: r.LayersDir,
			Logger:    r.Logger,
			Nop:       false,
		}, r.PlatformAPI)
	}

	layerSHAStore := layer.NewSHAStore()
	r.Logger.Debug("Restoring Layer Metadata")
	if err := r.LayerMetadataRestorer.Restore(r.Buildpacks, r.LayersMetadata, cacheMeta, layerSHAStore); err != nil {
		return err
	}

	var g errgroup.Group
	for _, bp := range r.Buildpacks {
		cachedLayers := cacheMeta.MetadataForBuildpack(bp.ID).Layers

		var cachedFn func(buildpack.Layer) bool
		// At this point in the build, <layer>.toml files never contain layer types information
		// (this information is added by buildpacks during the `build` phase).
		// The cache metadata is the only way to identify cache=true layers.
		cachedFn = func(l buildpack.Layer) bool {
			bpLayer, ok := cachedLayers[filepath.Base(l.Path())]
			return ok && bpLayer.Cache
		}

		r.Logger.Debugf("Reading Buildpack Layers directory %s", r.LayersDir)
		buildpackDir, err := buildpack.ReadLayersDir(r.LayersDir, bp, r.Logger)
		if err != nil {
			return errors.Wrapf(err, "reading buildpack layer directory")
		}
		foundLayers := buildpackDir.FindLayers(cachedFn)

		for _, bpLayer := range foundLayers {
			cachedLayer, exists := cachedLayers[bpLayer.Name()]
			if !exists {
				// This should be unreachable, as "find layers" uses the same cache metadata as the map
				r.Logger.Infof("Removing %q, not in cache", bpLayer.Identifier())
				if err := bpLayer.Remove(); err != nil {
					return errors.Wrapf(err, "removing layer")
				}
				continue
			}

			layerSha, err := layerSHAStore.Get(bp.ID, bpLayer)
			if err != nil {
				return err
			}

			if layerSha != cachedLayer.SHA {
				r.Logger.Infof("Removing %q, wrong sha", bpLayer.Identifier())
				r.Logger.Debugf("Layer sha: %q, cache sha: %q", layerSha, cachedLayer.SHA)
				if err := bpLayer.Remove(); err != nil {
					return errors.Wrapf(err, "removing layer")
				}
			} else {
				r.Logger.Infof("Restoring data for %q from cache", bpLayer.Identifier())
				g.Go(func() error {
					err = r.restoreCacheLayer(cache, cachedLayer.SHA)
					if err != nil {
						isReadErr, readErr := c.IsReadErr(err)
						if isReadErr {
							r.Logger.Warnf("Skipping restore for layer %s: %s", bpLayer.Identifier(), readErr.Error())
							return nil
						}
						return errors.Wrapf(err, "restoring layer %s", bpLayer.Identifier())
					}
					return nil
				})
			}
		}
	}

	if r.PlatformAPI.AtLeast("0.8") {
		g.Go(func() error {
			if cacheMeta.BOM.SHA != "" {
				r.Logger.Infof("Restoring data for SBOM from cache")
				if err := r.SBOMRestorer.RestoreFromCache(cache, cacheMeta.BOM.SHA); err != nil {
					return err
				}
			}
			return r.SBOMRestorer.RestoreToBuildpackLayers(r.Buildpacks)
		})
	}

	if err := g.Wait(); err != nil {
		return errors.Wrap(err, "restoring data")
	}

	return nil
}

func (r *Restorer) restoreCacheLayer(cache Cache, sha string) error {
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

	return layers.Extract(rc, "")
}

func retrieveCacheMetadata(fromCache Cache, logger log.Logger) (platform.CacheMetadata, error) {
	// Create empty cache metadata in case a usable cache is not provided.
	var cacheMeta platform.CacheMetadata
	if fromCache != nil {
		var err error
		if !fromCache.Exists() {
			logger.Info("Layer cache not found")
		}
		cacheMeta, err = fromCache.RetrieveMetadata()
		if err != nil {
			return cacheMeta, errors.Wrap(err, "retrieving cache metadata")
		}
	} else {
		logger.Debug("Usable cache not provided, using empty cache metadata")
	}

	return cacheMeta, nil
}
