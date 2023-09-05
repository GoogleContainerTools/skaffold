package dist

import (
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/style"
)

const AssumedBuildpackAPIVersion = "0.1"
const BuildpacksDir = "/cnb/buildpacks"
const ExtensionsDir = "/cnb/extensions"

type ModuleInfo struct {
	ID          string    `toml:"id,omitempty" json:"id,omitempty" yaml:"id,omitempty"`
	Name        string    `toml:"name,omitempty" json:"name,omitempty" yaml:"name,omitempty"`
	Version     string    `toml:"version,omitempty" json:"version,omitempty" yaml:"version,omitempty"`
	Description string    `toml:"description,omitempty" json:"description,omitempty" yaml:"description,omitempty"`
	Homepage    string    `toml:"homepage,omitempty" json:"homepage,omitempty" yaml:"homepage,omitempty"`
	Keywords    []string  `toml:"keywords,omitempty" json:"keywords,omitempty" yaml:"keywords,omitempty"`
	Licenses    []License `toml:"licenses,omitempty" json:"licenses,omitempty" yaml:"licenses,omitempty"`
}

func (b ModuleInfo) FullName() string {
	if b.Version != "" {
		return b.ID + "@" + b.Version
	}
	return b.ID
}

func (b ModuleInfo) FullNameWithVersion() (string, error) {
	if b.Version == "" {
		return b.ID, errors.Errorf("buildpack %s does not have a version defined", style.Symbol(b.ID))
	}
	return b.ID + "@" + b.Version, nil
}

// Satisfy stringer
func (b ModuleInfo) String() string { return b.FullName() }

// Match compares two buildpacks by ID and Version
func (b ModuleInfo) Match(o ModuleInfo) bool {
	return b.ID == o.ID && b.Version == o.Version
}

type License struct {
	Type string `toml:"type"`
	URI  string `toml:"uri"`
}

type Stack struct {
	ID     string   `json:"id" toml:"id"`
	Mixins []string `json:"mixins,omitempty" toml:"mixins,omitempty"`
}

type Target struct {
	OS            string         `json:"os" toml:"os"`
	Arch          string         `json:"arch" toml:"arch"`
	Distributions []Distribution `json:"distributions,omitempty" toml:"distributions,omitempty"`
}

type Distribution struct {
	Name     string   `json:"name,omitempty" toml:"name,omitempty"`
	Versions []string `json:"versions,omitempty" toml:"versions,omitempty"`
}
