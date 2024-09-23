package buildpack

import (
	"fmt"

	"github.com/buildpacks/lifecycle/log"
)

type BOMValidator interface {
	ValidateBOM(GroupElement, []BOMEntry) ([]BOMEntry, error)
}

// NewBOMValidator returns a validator for the legacy unstructured BOM.
func NewBOMValidator(_ string, layersDir string, logger log.Logger) BOMValidator {
	return &defaultBOMValidator{logger: logger, layersDir: layersDir}
}

type defaultBOMValidator struct {
	logger    log.Logger
	layersDir string
}

func (v *defaultBOMValidator) ValidateBOM(bp GroupElement, bom []BOMEntry) ([]BOMEntry, error) {
	if err := v.validateBOM(bom); err != nil {
		return []BOMEntry{}, err
	}
	return v.processBOM(bp, bom), nil
}

func (v *defaultBOMValidator) validateBOM(bom []BOMEntry) error {
	sbomMatches, err := sbomGlob(v.layersDir)
	if err != nil {
		return err
	}

	switch {
	case len(bom) > 0 && len(sbomMatches) > 0:
		// no-op: Don't show a warning here.
		// This code path represents buildpack authors providing a
		// migration path from old BOM to new SBOM.
	case len(bom) > 0:
		v.logger.Warn("BOM table is deprecated in this buildpack api version, though it remains supported for backwards compatibility. Buildpack authors should write BOM information to <layer>.sbom.<ext>, launch.sbom.<ext>, or build.sbom.<ext>.")
	}

	for _, entry := range bom {
		if entry.Version != "" {
			return fmt.Errorf("bom entry '%s' has a top level version which is not allowed. The buildpack should instead set metadata.version", entry.Name)
		}
	}

	return nil
}

func (v *defaultBOMValidator) processBOM(buildpack GroupElement, bom []BOMEntry) []BOMEntry {
	return WithBuildpack(buildpack, bom)
}

func WithBuildpack(bp GroupElement, bom []BOMEntry) []BOMEntry {
	var out []BOMEntry
	for _, entry := range bom {
		entry.Buildpack = bp.NoAPI().NoHomepage()
		out = append(out, entry)
	}
	return out
}
