package v01

import (
	"github.com/BurntSushi/toml"
	"github.com/buildpacks/lifecycle/api"

	"github.com/buildpacks/pack/pkg/project/types"
)

type Descriptor struct {
	Project  types.Project          `toml:"project"`
	Build    types.Build            `toml:"build"`
	Metadata map[string]interface{} `toml:"metadata"`
}

func NewDescriptor(projectTomlContents string) (types.Descriptor, toml.MetaData, error) {
	versionedDescriptor := &Descriptor{}

	tomlMetaData, err := toml.Decode(projectTomlContents, versionedDescriptor)
	if err != nil {
		return types.Descriptor{}, tomlMetaData, err
	}

	return types.Descriptor{
		Project:       versionedDescriptor.Project,
		Build:         versionedDescriptor.Build,
		Metadata:      versionedDescriptor.Metadata,
		SchemaVersion: api.MustParse("0.1"),
	}, tomlMetaData, nil
}
