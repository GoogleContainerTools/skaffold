// Data Format Files for the buildpack api spec (https://github.com/buildpacks/spec/blob/main/buildpack.md#data-format).

package buildpack

import (
	"fmt"

	"github.com/BurntSushi/toml"

	"github.com/buildpacks/lifecycle/api"
	"github.com/buildpacks/lifecycle/launch"
	"github.com/buildpacks/lifecycle/layers"
)

// launch.toml

type LaunchTOML struct {
	BOM       []BOMEntry
	Labels    []Label
	Processes []ProcessEntry `toml:"processes"`
	Slices    []layers.Slice `toml:"slices"`
}

type ProcessEntry struct {
	Type             string         `toml:"type" json:"type"`
	Command          []string       `toml:"-"` // ignored
	RawCommandValue  toml.Primitive `toml:"command" json:"command"`
	Args             []string       `toml:"args" json:"args"`
	Direct           *bool          `toml:"direct" json:"direct"`
	Default          bool           `toml:"default,omitempty" json:"default,omitempty"`
	WorkingDirectory string         `toml:"working-dir,omitempty" json:"working-dir,omitempty"`
}

// DecodeLaunchTOML reads a launch.toml file
func DecodeLaunchTOML(launchPath string, bpAPI string, launchTOML *LaunchTOML) error {
	// decode the common bits
	md, err := toml.DecodeFile(launchPath, &launchTOML)
	if err != nil {
		return err
	}

	// decode the process.commands, which differ based on buildpack API
	commandsAreStrings := api.MustParse(bpAPI).LessThan("0.9")

	// processes are defined differently depending on API version
	// and will be decoded into different values
	for i, process := range launchTOML.Processes {
		if commandsAreStrings {
			var commandString string
			if err = md.PrimitiveDecode(process.RawCommandValue, &commandString); err != nil {
				return err
			}
			// legacy Direct defaults to false
			if process.Direct == nil {
				direct := false
				launchTOML.Processes[i].Direct = &direct
			}
			launchTOML.Processes[i].Command = []string{commandString}
		} else {
			// direct is no longer allowed as a key
			if process.Direct != nil {
				return fmt.Errorf("process.direct is not supported on this buildpack version")
			}
			var command []string
			if err = md.PrimitiveDecode(process.RawCommandValue, &command); err != nil {
				return err
			}
			launchTOML.Processes[i].Command = command
		}
	}

	return nil
}

// ToLaunchProcess converts a buildpack.ProcessEntry to a launch.Process
func (p *ProcessEntry) ToLaunchProcess(bpID string) launch.Process {
	// legacy processes will always have a value
	// new processes will have a nil value but are always direct processes
	var direct bool
	if p.Direct == nil {
		direct = true
	} else {
		direct = *p.Direct
	}

	return launch.Process{
		Type:             p.Type,
		Command:          launch.NewRawCommand(p.Command),
		Args:             p.Args,
		Direct:           direct, // launch.Process requires a value
		Default:          p.Default,
		BuildpackID:      bpID,
		WorkingDirectory: p.WorkingDirectory,
	}
}

// converts launch.toml processes to launch.Processes
func (lt LaunchTOML) ToLaunchProcessesForBuildpack(bpID string) []launch.Process {
	var processes []launch.Process
	for _, process := range lt.Processes {
		processes = append(processes, process.ToLaunchProcess(bpID))
	}
	return processes
}

type BOMEntry struct {
	Require
	Buildpack GroupElement `toml:"buildpack" json:"buildpack"`
}

type Require struct {
	Name     string                 `toml:"name" json:"name"`
	Version  string                 `toml:"version,omitempty" json:"version,omitempty"`
	Metadata map[string]interface{} `toml:"metadata" json:"metadata"`
}

func (r *Require) hasDoublySpecifiedVersions() bool {
	if _, ok := r.Metadata["version"]; ok {
		return r.Version != ""
	}
	return false
}

func (r *Require) hasInconsistentVersions() bool {
	if version, ok := r.Metadata["version"]; ok {
		return r.Version != "" && r.Version != version
	}
	return false
}

func (r *Require) hasTopLevelVersions() bool {
	return r.Version != ""
}

type Label struct {
	Key   string `toml:"key"`
	Value string `toml:"value"`
}

// build.toml

type BuildTOML struct {
	BOM   []BOMEntry `toml:"bom"`
	Unmet []Unmet    `toml:"unmet"`
}

type Unmet struct {
	Name string `toml:"name"`
}

// store.toml

type StoreTOML struct {
	Data map[string]interface{} `json:"metadata" toml:"metadata"`
}

// build plan

type BuildPlan struct {
	PlanSections
	Or planSectionsList `toml:"or"`
}

func (p *PlanSections) hasInconsistentVersions() bool {
	for _, req := range p.Requires {
		if req.hasInconsistentVersions() {
			return true
		}
	}
	return false
}

func (p *PlanSections) hasDoublySpecifiedVersions() bool {
	for _, req := range p.Requires {
		if req.hasDoublySpecifiedVersions() {
			return true
		}
	}
	return false
}

func (p *PlanSections) hasTopLevelVersions() bool {
	for _, req := range p.Requires {
		if req.hasTopLevelVersions() {
			return true
		}
	}
	return false
}

func (p *PlanSections) hasRequires() bool {
	return len(p.Requires) > 0
}

type planSectionsList []PlanSections

func (p *planSectionsList) hasInconsistentVersions() bool {
	for _, planSection := range *p {
		if planSection.hasInconsistentVersions() {
			return true
		}
	}
	return false
}

func (p *planSectionsList) hasDoublySpecifiedVersions() bool {
	for _, planSection := range *p {
		if planSection.hasDoublySpecifiedVersions() {
			return true
		}
	}
	return false
}

func (p *planSectionsList) hasTopLevelVersions() bool {
	for _, planSection := range *p {
		if planSection.hasTopLevelVersions() {
			return true
		}
	}
	return false
}

func (p *planSectionsList) hasRequires() bool {
	for _, planSection := range *p {
		if planSection.hasRequires() {
			return true
		}
	}
	return false
}

type PlanSections struct {
	Requires []Require `toml:"requires"`
	Provides []Provide `toml:"provides"`
}

type Provide struct {
	Name string `toml:"name"`
}

// buildpack plan

type Plan struct {
	Entries []Require `toml:"entries"`
}

func (p Plan) filter(unmet []Unmet) Plan {
	var out []Require
	for _, entry := range p.Entries {
		if !containsName(unmet, entry.Name) {
			out = append(out, entry)
		}
	}
	return Plan{Entries: out}
}

func (p Plan) toBOM() []BOMEntry {
	var bom []BOMEntry
	for _, entry := range p.Entries {
		bom = append(bom, BOMEntry{Require: entry})
	}
	return bom
}

func containsName(unmet []Unmet, name string) bool {
	for _, u := range unmet {
		if u.Name == name {
			return true
		}
	}
	return false
}

// layer content metadata

type LayersMetadata struct {
	ID      string                   `json:"key" toml:"key"`
	Version string                   `json:"version" toml:"version"`
	Layers  map[string]LayerMetadata `json:"layers" toml:"layers"`
	Store   *StoreTOML               `json:"store,omitempty" toml:"store"`
}

type LayerMetadata struct {
	SHA string `json:"sha" toml:"sha"`
	LayerMetadataFile
}
