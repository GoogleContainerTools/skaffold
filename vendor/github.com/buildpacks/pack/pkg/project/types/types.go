package types

import (
	"github.com/buildpacks/lifecycle/api"
)

type Script struct {
	API    string `toml:"api"`
	Inline string `toml:"inline"`
	Shell  string `toml:"shell"`
}

type Buildpack struct {
	ID      string `toml:"id"`
	Version string `toml:"version"`
	URI     string `toml:"uri"`
	Script  Script `toml:"script"`
}

type EnvVar struct {
	Name  string `toml:"name"`
	Value string `toml:"value"`
}

type Build struct {
	Include    []string    `toml:"include"`
	Exclude    []string    `toml:"exclude"`
	Buildpacks []Buildpack `toml:"buildpacks"`
	Env        []EnvVar    `toml:"env"`
	Builder    string      `toml:"builder"`
	Pre        GroupAddition
	Post       GroupAddition
}

type Project struct {
	Name      string    `toml:"name"`
	Version   string    `toml:"version"`
	SourceURL string    `toml:"source-url"`
	Licenses  []License `toml:"licenses"`
}

type License struct {
	Type string `toml:"type"`
	URI  string `toml:"uri"`
}

type Descriptor struct {
	Project       Project                `toml:"project"`
	Build         Build                  `toml:"build"`
	Metadata      map[string]interface{} `toml:"metadata"`
	SchemaVersion *api.Version
}

type GroupAddition struct {
	Buildpacks []Buildpack `toml:"group"`
}
