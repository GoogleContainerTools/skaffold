package builder

import (
	"github.com/buildpacks/pack/pkg/dist"
)

type DetectionOrderEntry struct {
	dist.BuildpackRef   `yaml:",inline"`
	Cyclical            bool           `json:"cyclic,omitempty" yaml:"cyclic,omitempty" toml:"cyclic,omitempty"`
	GroupDetectionOrder DetectionOrder `json:"buildpacks,omitempty" yaml:"buildpacks,omitempty" toml:"buildpacks,omitempty"`
}

type DetectionOrder []DetectionOrderEntry

const (
	OrderDetectionMaxDepth = -1
	OrderDetectionNone     = 0
)
