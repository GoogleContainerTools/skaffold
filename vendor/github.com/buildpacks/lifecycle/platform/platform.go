package platform

import (
	"github.com/buildpacks/lifecycle/api"
)

type Platform struct {
	Exiter
	api *api.Version
}

func NewPlatform(apiStr string) *Platform {
	platform := Platform{
		api: api.MustParse(apiStr),
	}
	switch apiStr {
	case "0.3", "0.4", "0.5":
		platform.Exiter = &LegacyExiter{}
	default:
		platform.Exiter = &DefaultExiter{}
	}
	return &platform
}

func (p *Platform) API() *api.Version {
	return p.api
}
