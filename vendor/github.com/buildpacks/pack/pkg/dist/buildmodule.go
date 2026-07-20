package dist

import (
	"fmt"
	"strings"

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
	ExecEnv     []string  `toml:"exec-env,omitempty" json:"exec-env,omitempty" yaml:"exec-env,omitempty"`
	Licenses    []License `toml:"licenses,omitempty" json:"licenses,omitempty" yaml:"licenses,omitempty"`
	ClearEnv    bool      `toml:"clear-env,omitempty" json:"clear-env,omitempty" yaml:"clear-env,omitempty"`
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
	ArchVariant   string         `json:"variant,omitempty" toml:"variant,omitempty"`
	Distributions []Distribution `json:"distros,omitempty" toml:"distros,omitempty"`
}

// ValuesAsSlice converts the internal representation of a target (os, arch, variant, etc.) into a string slice,
// where each value included in the final array must be not empty.
func (t *Target) ValuesAsSlice() []string {
	var targets []string
	if t.OS != "" {
		targets = append(targets, t.OS)
	}
	if t.Arch != "" {
		targets = append(targets, t.Arch)
	}
	if t.ArchVariant != "" {
		targets = append(targets, t.ArchVariant)
	}

	for _, d := range t.Distributions {
		targets = append(targets, fmt.Sprintf("%s@%s", d.Name, d.Version))
	}
	return targets
}

func (t *Target) ValuesAsPlatform() string {
	return strings.Join(t.ValuesAsSlice(), "/")
}

// ExpandTargetsDistributions expands each provided target (with multiple distribution versions) to multiple targets (each with a single distribution version).
// For example, given an array with ONE target with the format:
//
//	[
//	  {OS:"linux", Distributions: []dist.Distribution{{Name: "ubuntu", Version: "18.01"},{Name: "ubuntu", Version: "21.01"}}}
//	]
//
// it returns an array with TWO targets each with the format:
//
//	[
//	 {OS:"linux",Distributions: []dist.Distribution{{Name: "ubuntu", Version: "18.01"}}},
//	 {OS:"linux",Distributions: []dist.Distribution{{Name: "ubuntu", Version: "21.01"}}}
//	]
func ExpandTargetsDistributions(targets ...Target) []Target {
	var expandedTargets []Target
	for _, target := range targets {
		expandedTargets = append(expandedTargets, expandTargetDistributions(target)...)
	}
	return expandedTargets
}

func expandTargetDistributions(target Target) []Target {
	var expandedTargets []Target
	if (len(target.Distributions)) > 1 {
		originalDistros := target.Distributions
		for _, distro := range originalDistros {
			copyTarget := target
			copyTarget.Distributions = []Distribution{distro}
			expandedTargets = append(expandedTargets, copyTarget)
		}
	} else {
		expandedTargets = append(expandedTargets, target)
	}
	return expandedTargets
}

type Distribution struct {
	Name    string `json:"name,omitempty" toml:"name,omitempty"`
	Version string `json:"version,omitempty" toml:"version,omitempty"`
}
