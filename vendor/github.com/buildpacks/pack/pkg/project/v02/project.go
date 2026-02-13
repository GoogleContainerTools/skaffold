package v02

import (
	"github.com/BurntSushi/toml"
	"github.com/buildpacks/lifecycle/api"

	"github.com/buildpacks/pack/pkg/project/types"
)

type Buildpacks struct {
	Include []string      `toml:"include"`
	Exclude []string      `toml:"exclude"`
	Group   []Buildpack   `toml:"group"`
	Env     Env           `toml:"env"`
	Build   Build         `toml:"build"`
	Builder string        `toml:"builder"`
	Pre     GroupAddition `toml:"pre"`
	Post    GroupAddition `toml:"post"`
}

type Build struct {
	Env []EnvVar `toml:"env"`
}

// Deprecated: use `[[io.buildpacks.build.env]]` instead. see https://github.com/buildpacks/pack/pull/1479
type Env struct {
	Build []EnvVar `toml:"build"`
}

type Project struct {
	SchemaVersion    string                 `toml:"schema-version"`
	ID               string                 `toml:"id"`
	Name             string                 `toml:"name"`
	Version          string                 `toml:"version"`
	Authors          []string               `toml:"authors"`
	Licenses         []types.License        `toml:"licenses"`
	DocumentationURL string                 `toml:"documentation-url"`
	SourceURL        string                 `toml:"source-url"`
	Metadata         map[string]interface{} `toml:"metadata"`
}

type IO struct {
	Buildpacks Buildpacks `toml:"buildpacks"`
}

type Descriptor struct {
	Project Project `toml:"_"`
	IO      IO      `toml:"io"`
}

type Buildpack struct {
	ID      string       `toml:"id"`
	Version string       `toml:"version"`
	URI     string       `toml:"uri"`
	Script  types.Script `toml:"script"`
}

type EnvVar struct {
	Name  string `toml:"name"`
	Value string `toml:"value"`
}

type GroupAddition struct {
	Buildpacks []Buildpack `toml:"group"`
}

func NewDescriptor(projectTomlContents string) (types.Descriptor, toml.MetaData, error) {
	versionedDescriptor := &Descriptor{}
	tomlMetaData, err := toml.Decode(projectTomlContents, &versionedDescriptor)
	if err != nil {
		return types.Descriptor{}, tomlMetaData, err
	}

	// backward compatibility for incorrect key
	env := versionedDescriptor.IO.Buildpacks.Build.Env
	if env == nil {
		env = versionedDescriptor.IO.Buildpacks.Env.Build
	}

	return types.Descriptor{
		Project: types.Project{
			Name:     versionedDescriptor.Project.Name,
			Licenses: versionedDescriptor.Project.Licenses,
		},
		Build: types.Build{
			Include:    versionedDescriptor.IO.Buildpacks.Include,
			Exclude:    versionedDescriptor.IO.Buildpacks.Exclude,
			Buildpacks: mapToBuildPacksDescriptor(versionedDescriptor.IO.Buildpacks.Group),
			Env:        mapToEnvVarsDescriptor(env),
			Builder:    versionedDescriptor.IO.Buildpacks.Builder,
			Pre: types.GroupAddition{
				Buildpacks: mapToBuildPacksDescriptor(versionedDescriptor.IO.Buildpacks.Pre.Buildpacks),
			},
			Post: types.GroupAddition{
				Buildpacks: mapToBuildPacksDescriptor(versionedDescriptor.IO.Buildpacks.Post.Buildpacks),
			},
		},
		Metadata:      versionedDescriptor.Project.Metadata,
		SchemaVersion: api.MustParse("0.2"),
	}, tomlMetaData, nil
}

func mapToBuildPacksDescriptor(v2BuildPacks []Buildpack) []types.Buildpack {
	var buildPacks []types.Buildpack
	for _, v2BuildPack := range v2BuildPacks {
		buildPacks = append(buildPacks, mapToBuildPackDescriptor(v2BuildPack))
	}
	return buildPacks
}

func mapToBuildPackDescriptor(v2BuildPack Buildpack) types.Buildpack {
	return types.Buildpack{
		ID:      v2BuildPack.ID,
		Version: v2BuildPack.Version,
		URI:     v2BuildPack.URI,
		Script:  v2BuildPack.Script,
		ExecEnv: []string{}, // schema v2 doesn't handle execution environments variables
	}
}

func mapToEnvVarsDescriptor(v2EnvVars []EnvVar) []types.EnvVar {
	var envVars []types.EnvVar
	for _, v2EnvVar := range v2EnvVars {
		envVars = append(envVars, mapToEnVarDescriptor(v2EnvVar))
	}
	return envVars
}

func mapToEnVarDescriptor(v2EnVar EnvVar) types.EnvVar {
	return types.EnvVar{
		Name:    v2EnVar.Name,
		Value:   v2EnVar.Value,
		ExecEnv: []string{}, // schema v2 doesn't handle execution environments variables
	}
}
