package layer

import (
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/buildpacks/lifecycle/api"
	"github.com/buildpacks/lifecycle/buildpack"
	"github.com/buildpacks/lifecycle/internal/encoding"
	"github.com/buildpacks/lifecycle/launch"
	"github.com/buildpacks/lifecycle/log"
	"github.com/buildpacks/lifecycle/platform"
	"github.com/buildpacks/lifecycle/platform/files"
)

// MetadataRestorer given a group of buildpacks and metadata from the previous image and cache,
// will create `<layers>/<buildpack-id>/<layer>.toml` files containing `metadata` that the buildpack previously wrote.
// Note that layer `types` information is not persisted, as the buildpack must opt in to layer re-use
// by editing the provided `<layer>.toml` to configure the desired layer type.
//
//go:generate mockgen -package testmock -destination ../../phase/testmock/metadata_restorer.go github.com/buildpacks/lifecycle/internal/layer MetadataRestorer
type MetadataRestorer interface {
	Restore(buildpacks []buildpack.GroupElement, appMeta files.LayersMetadata, cacheMeta platform.CacheMetadata, layerSHAStore SHAStore) error
}

// NewDefaultMetadataRestorer returns an instance of the DefaultMetadataRestorer struct
func NewDefaultMetadataRestorer(layersDir string, skipLayers bool, logger log.Logger, platformAPI *api.Version) *DefaultMetadataRestorer {
	return &DefaultMetadataRestorer{
		Logger:      logger,
		LayersDir:   layersDir,
		SkipLayers:  skipLayers,
		PlatformAPI: platformAPI,
	}
}

type DefaultMetadataRestorer struct {
	LayersDir   string
	SkipLayers  bool
	Logger      log.Logger
	PlatformAPI *api.Version
}

func (r *DefaultMetadataRestorer) Restore(buildpacks []buildpack.GroupElement, appMeta files.LayersMetadata, cacheMeta platform.CacheMetadata, layerSHAStore SHAStore) error {
	if err := r.restoreStoreTOML(appMeta, buildpacks); err != nil {
		return err
	}

	if err := r.restoreLayerMetadata(layerSHAStore, appMeta, cacheMeta, buildpacks); err != nil {
		return err
	}

	return nil
}

func (r *DefaultMetadataRestorer) restoreStoreTOML(appMeta files.LayersMetadata, buildpacks []buildpack.GroupElement) error {
	for _, bp := range buildpacks {
		if store := appMeta.LayersMetadataFor(bp.ID).Store; store != nil {
			if err := encoding.WriteTOML(filepath.Join(r.LayersDir, launch.EscapeID(bp.ID), "store.toml"), store); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *DefaultMetadataRestorer) restoreLayerMetadata(layerSHAStore SHAStore, appMeta files.LayersMetadata, cacheMeta platform.CacheMetadata, buildpacks []buildpack.GroupElement) error {
	if r.SkipLayers {
		r.Logger.Infof("Skipping buildpack layer analysis")
		return nil
	}

	for _, bp := range buildpacks {
		buildpackDir, err := buildpack.ReadLayersDir(r.LayersDir, bp, r.Logger)
		if err != nil {
			return errors.Wrap(err, "reading buildpack layer directory")
		}

		// Restore metadata for launch=true layers.
		// The restorer step will restore the layer data for cache=true layers if possible or delete the layer.
		appLayers := appMeta.LayersMetadataFor(bp.ID).Layers
		cachedLayers := cacheMeta.MetadataForBuildpack(bp.ID).Layers
		for layerName, layer := range appLayers {
			identifier := fmt.Sprintf("%s:%s", bp.ID, layerName)
			if !layer.Launch {
				r.Logger.Debugf("Not restoring metadata for %q, marked as launch=false", identifier)
				continue
			}
			if layer.Build && !layer.Cache {
				// layer is launch=true, build=true. Because build=true, the layer contents must be present in the build container.
				// There is no reason to restore the metadata file, because the buildpack will always recreate the layer.
				r.Logger.Debugf("Not restoring metadata for %q, marked as build=true, cache=false", identifier)
				continue
			}
			if layer.Cache {
				if cacheLayer, ok := cachedLayers[layerName]; !ok || !cacheLayer.Cache {
					// The layer is not cache=true in the cache metadata and will not be restored.
					// Do not write the metadata file so that it is clear to the buildpack that it needs to recreate the layer.
					// Although a launch=true (only) layer still needs a metadata file,
					// the restorer will remove the file anyway when it does its cleanup.
					r.Logger.Debugf("Not restoring metadata for %q, marked as cache=true, but not found in cache", identifier)
					continue
				}
			}
			r.Logger.Infof("Restoring metadata for %q from app image", identifier)
			if err := r.writeLayerMetadata(layerSHAStore, buildpackDir, layerName, layer, bp.ID); err != nil {
				return err
			}
		}

		// Restore metadata for cache=true layers.
		// The restorer step will restore the layer data if possible or delete the layer.
		for layerName, layer := range cachedLayers {
			identifier := fmt.Sprintf("%s:%s", bp.ID, layerName)
			if !layer.Cache {
				r.Logger.Debugf("Not restoring %q from cache, marked as cache=false", identifier)
				continue
			}
			// If launch=true, the metadata was restored from the appLayers if present.
			if layer.Launch {
				if _, ok := appLayers[layerName]; ok || r.PlatformAPI.LessThan("0.14") {
					r.Logger.Debugf("Not restoring %q from cache, marked as launch=true", identifier)
					continue
				}
			}
			r.Logger.Infof("Restoring metadata for %q from cache", identifier)
			if err := r.writeLayerMetadata(layerSHAStore, buildpackDir, layerName, layer, bp.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *DefaultMetadataRestorer) writeLayerMetadata(layerSHAStore SHAStore, buildpackDir buildpack.LayersDir, layerName string, metadata buildpack.LayerMetadata, buildpackID string) error {
	layer := buildpackDir.NewLayer(layerName, buildpackDir.Buildpack.API, r.Logger)
	r.Logger.Debugf("Writing layer metadata for %q", layer.Identifier())
	if err := layer.WriteMetadata(metadata.LayerMetadataFile); err != nil {
		return err
	}
	return layerSHAStore.add(buildpackID, metadata.SHA, layer)
}

type NopMetadataRestorer struct{}

func (r *NopMetadataRestorer) Restore(_ []buildpack.GroupElement, _ files.LayersMetadata, _ platform.CacheMetadata, _ SHAStore) error {
	return nil
}

type SHAStore interface {
	add(buildpackID, sha string, layer *buildpack.Layer) error
	Get(buildpackID string, layer buildpack.Layer) (string, error)
}

// NewSHAStore returns a new SHAStore for mapping buildpack IDs to layer names and their SHAs.
func NewSHAStore() SHAStore {
	return &memoryStore{make(map[string]layerToSha)}
}

type memoryStore struct {
	buildpacksToLayersShaMap map[string]layerToSha
}

type layerToSha struct {
	layerToShaMap map[string]string
}

func (ms *memoryStore) add(buildpackID, sha string, layer *buildpack.Layer) error {
	ms.addLayerToMap(buildpackID, layer.Name(), sha)
	return nil
}

func (ms *memoryStore) Get(buildpackID string, layer buildpack.Layer) (string, error) {
	return ms.getShaByBuildpackLayer(buildpackID, layer.Name()), nil
}

func (ms *memoryStore) addLayerToMap(buildpackID, layerName, sha string) {
	_, exists := ms.buildpacksToLayersShaMap[buildpackID]
	if !exists {
		ms.buildpacksToLayersShaMap[buildpackID] = layerToSha{make(map[string]string)}
	}
	ms.buildpacksToLayersShaMap[buildpackID].layerToShaMap[layerName] = sha
}

// if the layer exists for the buildpack ID, its SHA will be returned. Otherwise, an empty string will be returned.
func (ms *memoryStore) getShaByBuildpackLayer(buildpackID, layerName string) string {
	if layerToSha, buildpackExists := ms.buildpacksToLayersShaMap[buildpackID]; buildpackExists {
		if sha, layerExists := layerToSha.layerToShaMap[layerName]; layerExists {
			return sha
		}
	}
	return ""
}
