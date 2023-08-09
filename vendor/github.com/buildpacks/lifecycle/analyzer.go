package lifecycle

import (
	"github.com/buildpacks/imgutil"
	"github.com/pkg/errors"

	"github.com/buildpacks/lifecycle/api"
	"github.com/buildpacks/lifecycle/buildpack"
	"github.com/buildpacks/lifecycle/image"
	"github.com/buildpacks/lifecycle/internal/layer"
	"github.com/buildpacks/lifecycle/platform"
)

type Platform interface {
	API() *api.Version
}

type Analyzer struct {
	PreviousImage imgutil.Image
	RunImage      imgutil.Image
	Logger        Logger
	Platform      Platform
	SBOMRestorer  layer.SBOMRestorer

	// Platform API < 0.7
	Buildpacks            []buildpack.GroupBuildpack
	Cache                 Cache
	LayerMetadataRestorer layer.MetadataRestorer
}

// Analyze fetches the layers metadata from the previous image and writes analyzed.toml.
func (a *Analyzer) Analyze() (platform.AnalyzedMetadata, error) {
	var (
		appMeta         platform.LayersMetadata
		cacheMeta       platform.CacheMetadata
		previousImageID *platform.ImageIdentifier
		runImageID      *platform.ImageIdentifier
		err             error
	)

	if a.PreviousImage != nil { // Previous image is optional in Platform API >= 0.7
		previousImageID, err = a.getImageIdentifier(a.PreviousImage)
		if err != nil {
			return platform.AnalyzedMetadata{}, errors.Wrap(err, "retrieving image identifier")
		}

		// continue even if the label cannot be decoded
		if err := image.DecodeLabel(a.PreviousImage, platform.LayerMetadataLabel, &appMeta); err != nil {
			appMeta = platform.LayersMetadata{}
		}

		if a.Platform.API().AtLeast("0.8") {
			if appMeta.BOM != nil && appMeta.BOM.SHA != "" {
				a.Logger.Infof("Restoring data for sbom from previous image")
				if err := a.SBOMRestorer.RestoreFromPrevious(a.PreviousImage, appMeta.BOM.SHA); err != nil {
					return platform.AnalyzedMetadata{}, errors.Wrap(err, "retrieving launch sBOM layer")
				}
			}
		}
	} else {
		appMeta = platform.LayersMetadata{}
	}

	if a.RunImage != nil {
		runImageID, err = a.getImageIdentifier(a.RunImage)
		if err != nil {
			return platform.AnalyzedMetadata{}, errors.Wrap(err, "retrieving image identifier")
		}
	}

	if a.restoresLayerMetadata() {
		cacheMeta, err = retrieveCacheMetadata(a.Cache, a.Logger)
		if err != nil {
			return platform.AnalyzedMetadata{}, err
		}

		useShaFiles := true
		if err := a.LayerMetadataRestorer.Restore(a.Buildpacks, appMeta, cacheMeta, layer.NewSHAStore(useShaFiles)); err != nil {
			return platform.AnalyzedMetadata{}, err
		}
	}

	return platform.AnalyzedMetadata{
		PreviousImage: previousImageID,
		RunImage:      runImageID,
		Metadata:      appMeta,
	}, nil
}

func (a *Analyzer) restoresLayerMetadata() bool {
	return a.Platform.API().LessThan("0.7")
}

func (a *Analyzer) getImageIdentifier(image imgutil.Image) (*platform.ImageIdentifier, error) {
	if !image.Found() {
		a.Logger.Infof("Previous image with name %q not found", image.Name())
		return nil, nil
	}
	identifier, err := image.Identifier()
	if err != nil {
		return nil, err
	}
	a.Logger.Debugf("Analyzing image %q", identifier.String())
	return &platform.ImageIdentifier{
		Reference: identifier.String(),
	}, nil
}

func retrieveCacheMetadata(cache Cache, logger Logger) (platform.CacheMetadata, error) {
	// Create empty cache metadata in case a usable cache is not provided.
	var cacheMeta platform.CacheMetadata
	if cache != nil {
		var err error
		if !cache.Exists() {
			logger.Info("Layer cache not found")
		}
		cacheMeta, err = cache.RetrieveMetadata()
		if err != nil {
			return cacheMeta, errors.Wrap(err, "retrieving cache metadata")
		}
	} else {
		logger.Debug("Usable cache not provided, using empty cache metadata")
	}

	return cacheMeta, nil
}
