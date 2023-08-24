package platform

import (
	"github.com/buildpacks/lifecycle/api"
)

type LifecyclePhase int

const (
	Analyze LifecyclePhase = iota
	Detect
	Restore
	Extend
	Build
	Export
	Create
	Rebase
)

// Platform holds lifecycle inputs and outputs for a given Platform API version and lifecycle phase.
type Platform struct {
	*LifecycleInputs
	Exiter
}

// NewPlatformFor accepts a Platform API version and a layers directory, and returns a Platform with default lifecycle inputs and an exiter service.
func NewPlatformFor(platformAPI string) *Platform {
	return &Platform{
		LifecycleInputs: NewLifecycleInputs(api.MustParse(platformAPI)),
		Exiter:          NewExiter(platformAPI),
	}
}

func (p *Platform) API() *api.Version {
	return p.PlatformAPI
}
