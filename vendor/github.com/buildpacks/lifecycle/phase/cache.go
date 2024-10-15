package phase

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/buildpacks/lifecycle/buildpack"
	c "github.com/buildpacks/lifecycle/cache"
	"github.com/buildpacks/lifecycle/layers"
	"github.com/buildpacks/lifecycle/log"
	"github.com/buildpacks/lifecycle/platform"
)

type LayerDir interface {
	Identifier() string
	Path() string
}

func (e *Exporter) Cache(layersDir string, cacheStore Cache) error {
	defer log.NewMeasurement("Cache", e.Logger)()
	var err error
	if !cacheStore.Exists() {
		e.Logger.Info("Layer cache not found")
	}
	origMeta, err := cacheStore.RetrieveMetadata()
	if err != nil {
		return errors.Wrap(err, "metadata for previous cache")
	}
	meta := platform.CacheMetadata{}

	for _, bp := range e.Buildpacks {
		bpDir, err := buildpack.ReadLayersDir(layersDir, bp, e.Logger)
		if err != nil {
			return errors.Wrapf(err, "reading layers for buildpack '%s'", bp.ID)
		}

		bpMD := buildpack.LayersMetadata{
			ID:      bp.ID,
			Version: bp.Version,
			Layers:  map[string]buildpack.LayerMetadata{},
		}
		for _, layer := range bpDir.FindLayers(buildpack.MadeCached) {
			layer := layer
			if !layer.HasLocalContents() {
				e.Logger.Warnf("Failed to cache layer '%s' because it has no contents", layer.Identifier())
				continue
			}
			lmd, err := layer.Read()
			if err != nil {
				e.Logger.Warnf("Failed to cache layer '%s' because of error reading metadata: %s", layer.Identifier(), err)
				continue
			}
			origLayerMetadata := origMeta.MetadataForBuildpack(bp.ID).Layers[layer.Name()]
			createdBy := fmt.Sprintf(layers.BuildpackLayerName, layer.Name(), fmt.Sprintf("%s@%s", bp.ID, bp.Version))
			if lmd.SHA, err = e.addOrReuseCacheLayer(cacheStore, &layer, origLayerMetadata.SHA, createdBy); err != nil {
				e.Logger.Warnf("Failed to cache layer '%s': %s", layer.Identifier(), err)
				continue
			}
			bpMD.Layers[layer.Name()] = lmd
		}
		meta.Buildpacks = append(meta.Buildpacks, bpMD)
	}

	if e.PlatformAPI.AtLeast("0.8") {
		if err := e.addSBOMCacheLayer(layersDir, cacheStore, origMeta, &meta); err != nil {
			return err
		}
	}

	if err := cacheStore.SetMetadata(meta); err != nil {
		return errors.Wrap(err, "setting cache metadata")
	}
	if err := cacheStore.Commit(); err != nil {
		return errors.Wrap(err, "committing cache")
	}

	return nil
}

type layerDir struct {
	path       string
	identifier string
}

func (l *layerDir) Identifier() string {
	return l.identifier
}

func (l *layerDir) Path() string {
	return l.path
}

func (e *Exporter) addOrReuseCacheLayer(cache Cache, layerDir LayerDir, previousSHA, createdBy string) (string, error) {
	layer, err := e.LayerFactory.DirLayer(layerDir.Identifier(), layerDir.Path(), createdBy)
	if err != nil {
		return "", errors.Wrapf(err, "creating layer '%s'", layerDir.Identifier())
	}
	if layer.Digest == previousSHA {
		e.Logger.Infof("Reusing cache layer '%s'\n", layer.ID)
		e.Logger.Debugf("Layer '%s' SHA: %s\n", layer.ID, layer.Digest)
		err = cache.ReuseLayer(previousSHA)
		if err != nil {
			isReadErr, readErr := c.IsReadErr(err)
			if !isReadErr {
				return "", errors.Wrapf(err, "reusing layer %s", layer.ID)
			}
			e.Logger.Warnf("Skipping re-use for layer %s: %s", layer.ID, readErr.Error())
		}
	}
	e.Logger.Infof("Adding cache layer '%s'\n", layer.ID)
	e.Logger.Debugf("Layer '%s' SHA: %s\n", layer.ID, layer.Digest)
	return layer.Digest, cache.AddLayerFile(layer.TarPath, layer.Digest)
}

func (e *Exporter) addSBOMCacheLayer(layersDir string, cacheStore Cache, origMetadata platform.CacheMetadata, meta *platform.CacheMetadata) error {
	sbomCacheDir, err := readLayersSBOM(layersDir, "cache", e.Logger)
	if err != nil {
		return errors.Wrap(err, "failed to read layers SBOM")
	}

	if sbomCacheDir != nil {
		l, err := e.LayerFactory.DirLayer(sbomCacheDir.Identifier(), sbomCacheDir.Path(), layers.SBOMLayerName)
		if err != nil {
			return errors.Wrapf(err, "creating layer '%s', path: '%s'", sbomCacheDir.Identifier(), sbomCacheDir.Path())
		}

		lyr := &layerDir{path: l.TarPath, identifier: l.ID}

		meta.BOM.SHA, err = e.addOrReuseCacheLayer(cacheStore, lyr, origMetadata.BOM.SHA, layers.SBOMLayerName)
		if err != nil {
			return err
		}
	}

	return nil
}

func readLayersSBOM(layersDir string, bomType string, logger log.Logger) (LayerDir, error) {
	path := filepath.Join(layersDir, "sbom", bomType)
	_, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}

	logger.Debugf("Found SBOM of type %s for at %s", bomType, path)
	return &layerDir{
		path:       path,
		identifier: fmt.Sprintf("buildpacksio/lifecycle:%s.sbom", bomType),
	}, nil
}
