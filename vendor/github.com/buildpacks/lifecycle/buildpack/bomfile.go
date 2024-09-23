package buildpack

import (
	"fmt"
	"mime"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/buildpacks/lifecycle/log"
)

const (
	LayerTypeBuild LayerType = iota
	LayerTypeCache
	LayerTypeLaunch
)

const (
	MediaTypeCycloneDX   = "application/vnd.cyclonedx+json"
	MediaTypeSPDX        = "application/spdx+json"
	MediaTypeSyft        = "application/vnd.syft+json"
	mediaTypeUnsupported = "unsupported"

	ExtensionCycloneDX = "sbom.cdx.json"
	ExtensionSPDX      = "sbom.spdx.json"
	ExtensionSyft      = "sbom.syft.json"
)

type LayerType int

type BOMFile struct {
	BuildpackID string
	LayerName   string
	LayerType   LayerType
	Path        string
}

// Name() returns the destination filename for a given BOM file
// cdx files should be renamed to "sbom.cdx.json"
// spdx files should be renamed to "sbom.spdx.json"
// syft files should be renamed to "sbom.syft.json"
// If the BOM is neither cdx, spdx, nor syft, the 2nd return argument
// will return an error to indicate an unsupported format
func (b *BOMFile) Name() (string, error) {
	switch b.mediaType() {
	case MediaTypeCycloneDX:
		return ExtensionCycloneDX, nil
	case MediaTypeSPDX:
		return ExtensionSPDX, nil
	case MediaTypeSyft:
		return ExtensionSyft, nil
	default:
		return "", errors.Errorf("unsupported SBOM format: '%s'", b.Path)
	}
}

func (b *BOMFile) mediaType() string {
	name := filepath.Base(b.Path)

	switch {
	case strings.HasSuffix(name, "."+ExtensionCycloneDX):
		return MediaTypeCycloneDX
	case strings.HasSuffix(name, "."+ExtensionSPDX):
		return MediaTypeSPDX
	case strings.HasSuffix(name, "."+ExtensionSyft):
		return MediaTypeSyft
	default:
		return mediaTypeUnsupported
	}
}

func validateMediaTypes(bp GroupElement, bomfiles []BOMFile, declaredTypes []string) error {
	ensureDeclared := func(declaredTypes []string, foundType string) error {
		for _, declaredType := range declaredTypes {
			dType, _, err := mime.ParseMediaType(declaredType)
			if err != nil {
				return errors.Wrap(err, "parsing declared media type")
			}
			if foundType == dType {
				return nil
			}
		}
		return errors.Errorf("undeclared SBOM media type: '%s'", foundType)
	}

	for _, bomFile := range bomfiles {
		fileType := bomFile.mediaType()
		switch fileType {
		case mediaTypeUnsupported:
			return errors.Errorf("unsupported SBOM file format: '%s'", bomFile.Path)
		default:
			if err := ensureDeclared(declaredTypes, fileType); err != nil {
				return errors.Wrap(err, fmt.Sprintf("validating SBOM file '%s' for buildpack: '%s'", bomFile.Path, bp.String()))
			}
		}
	}

	return nil
}

func sbomGlob(layersDir string) (matches []string, err error) {
	layerGlob := filepath.Join(layersDir, "*.sbom.*.json")
	matches, err = filepath.Glob(layerGlob)
	return
}

func (d *BpDescriptor) processSBOMFiles(layersDir string, bp GroupElement, bpLayers map[string]LayerMetadataFile, logger log.Logger) ([]BOMFile, error) {
	var (
		files []BOMFile
	)

	matches, err := sbomGlob(layersDir)
	if err != nil {
		return nil, err
	}

	for _, m := range matches {
		layerDir, file := filepath.Split(m)
		layerName := strings.SplitN(file, ".", 2)[0]

		if layerName == "launch" {
			files = append(files, BOMFile{
				BuildpackID: bp.ID,
				LayerType:   LayerTypeLaunch,
				Path:        m,
			})

			continue
		}

		if layerName == "build" {
			files = append(files, BOMFile{
				BuildpackID: bp.ID,
				LayerType:   LayerTypeBuild,
				Path:        m,
			})

			continue
		}

		meta, ok := bpLayers[filepath.Join(layerDir, layerName)]
		if !ok {
			continue
		}

		if meta.Launch {
			files = append(files, BOMFile{
				BuildpackID: bp.ID,
				LayerName:   layerName,
				LayerType:   LayerTypeLaunch,
				Path:        m,
			})
		} else {
			files = append(files, BOMFile{
				BuildpackID: bp.ID,
				LayerName:   layerName,
				LayerType:   LayerTypeBuild,
				Path:        m,
			})
		}

		if meta.Cache {
			files = append(files, BOMFile{
				BuildpackID: bp.ID,
				LayerName:   layerName,
				LayerType:   LayerTypeCache,
				Path:        m,
			})
		}
	}

	return files, validateMediaTypes(bp, files, d.Buildpack.SBOM)
}
