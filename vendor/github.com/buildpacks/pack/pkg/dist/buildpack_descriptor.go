package dist

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/buildpacks/lifecycle/api"

	"github.com/buildpacks/pack/internal/stringset"
	"github.com/buildpacks/pack/internal/style"
)

type BuildpackDescriptor struct {
	WithAPI          *api.Version `toml:"api"`
	WithInfo         ModuleInfo   `toml:"buildpack"`
	WithStacks       []Stack      `toml:"stacks,omitempty"`
	WithTargets      []Target     `toml:"targets,omitempty"`
	WithOrder        Order        `toml:"order"`
	WithWindowsBuild bool
	WithLinuxBuild   bool
}

func (b *BuildpackDescriptor) EscapedID() string {
	return strings.ReplaceAll(b.Info().ID, "/", "_")
}

func (b *BuildpackDescriptor) EnsureStackSupport(stackID string, providedMixins []string, validateRunStageMixins bool) error {
	if len(b.Stacks()) == 0 {
		return nil // Order buildpack or a buildpack using Targets, no validation required
	}

	bpMixins, err := b.findMixinsForStack(stackID)
	if err != nil {
		return err
	}

	if !validateRunStageMixins {
		var filtered []string
		for _, m := range bpMixins {
			if !strings.HasPrefix(m, "run:") {
				filtered = append(filtered, m)
			}
		}
		bpMixins = filtered
	}

	_, missing, _ := stringset.Compare(providedMixins, bpMixins)
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("buildpack %s requires missing mixin(s): %s", style.Symbol(b.Info().FullName()), strings.Join(missing, ", "))
	}
	return nil
}

func (b *BuildpackDescriptor) EnsureTargetSupport(givenOS, givenArch, givenDistroName, givenDistroVersion string) error {
	if len(b.Targets()) == 0 {
		if (!b.WithLinuxBuild && !b.WithWindowsBuild) || len(b.Stacks()) > 0 { // nolint
			return nil // Order buildpack or stack buildpack, no validation required
		} else if b.WithLinuxBuild && givenOS == DefaultTargetOSLinux && givenArch == DefaultTargetArch {
			return nil
		} else if b.WithWindowsBuild && givenOS == DefaultTargetOSWindows && givenArch == DefaultTargetArch {
			return nil
		}
	}
	for _, bpTarget := range b.Targets() {
		if bpTarget.OS == givenOS {
			if bpTarget.Arch == "" || givenArch == "" || bpTarget.Arch == givenArch {
				if len(bpTarget.Distributions) == 0 || givenDistroName == "" || givenDistroVersion == "" {
					return nil
				}
				for _, bpDistro := range bpTarget.Distributions {
					if bpDistro.Name == givenDistroName && bpDistro.Version == givenDistroVersion {
						return nil
					}
				}
			}
		}
	}
	type osDistribution struct {
		Name    string `json:"name,omitempty"`
		Version string `json:"version,omitempty"`
	}
	type target struct {
		OS           string         `json:"os"`
		Arch         string         `json:"arch"`
		Distribution osDistribution `json:"distribution"`
	}
	return fmt.Errorf(
		"unable to satisfy target os/arch constraints; build image: %s, buildpack %s: %s",
		toJSONMaybe(target{
			OS:           givenOS,
			Arch:         givenArch,
			Distribution: osDistribution{Name: givenDistroName, Version: givenDistroVersion},
		}),
		style.Symbol(b.Info().FullName()),
		toJSONMaybe(b.Targets()),
	)
}

func toJSONMaybe(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%s", v) // hopefully v is a Stringer
	}
	return string(b)
}

func (b *BuildpackDescriptor) Kind() string {
	return "buildpack"
}

func (b *BuildpackDescriptor) API() *api.Version {
	return b.WithAPI
}

func (b *BuildpackDescriptor) Info() ModuleInfo {
	return b.WithInfo
}

func (b *BuildpackDescriptor) Order() Order {
	return b.WithOrder
}

func (b *BuildpackDescriptor) Stacks() []Stack {
	return b.WithStacks
}

func (b *BuildpackDescriptor) Targets() []Target {
	return b.WithTargets
}

func (b *BuildpackDescriptor) findMixinsForStack(stackID string) ([]string, error) {
	for _, s := range b.Stacks() {
		if s.ID == stackID || s.ID == "*" {
			return s.Mixins, nil
		}
	}
	return nil, fmt.Errorf("buildpack %s does not support stack %s", style.Symbol(b.Info().FullName()), style.Symbol(stackID))
}
