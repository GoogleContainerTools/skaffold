package inspectimage

import (
	"github.com/buildpacks/lifecycle/buildpack"
	"github.com/buildpacks/lifecycle/launch"
	"github.com/buildpacks/lifecycle/platform/files"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/dist"
)

type GeneralInfo struct {
	Name            string
	RunImageMirrors []config.RunImage
}

type RunImageMirrorDisplay struct {
	Name           string `json:"name" yaml:"name" toml:"name"`
	UserConfigured bool   `json:"user_configured,omitempty" yaml:"user_configured,omitempty" toml:"user_configured,omitempty"`
}

type StackDisplay struct {
	ID     string   `json:"id" yaml:"id" toml:"id"`
	Mixins []string `json:"mixins,omitempty" yaml:"mixins,omitempty" toml:"mixins,omitempty"`
}

type ProcessDisplay struct {
	Type    string   `json:"type" yaml:"type" toml:"type"`
	Shell   string   `json:"shell" yaml:"shell" toml:"shell"`
	Command string   `json:"command" yaml:"command" toml:"command"`
	Default bool     `json:"default" yaml:"default" toml:"default"`
	Args    []string `json:"args" yaml:"args" toml:"args"`
	WorkDir string   `json:"working-dir" yaml:"working-dir" toml:"working-dir"`
}

type BaseDisplay struct {
	TopLayer  string `json:"top_layer" yaml:"top_layer" toml:"top_layer"`
	Reference string `json:"reference" yaml:"reference" toml:"reference"`
}

type InfoDisplay struct {
	StackID         string                  `json:"stack" yaml:"stack" toml:"stack"`
	Base            BaseDisplay             `json:"base_image" yaml:"base_image" toml:"base_image"`
	RunImageMirrors []RunImageMirrorDisplay `json:"run_images" yaml:"run_images" toml:"run_images"`
	Buildpacks      []dist.ModuleInfo       `json:"buildpacks" yaml:"buildpacks" toml:"buildpacks"`
	Extensions      []dist.ModuleInfo       `json:"extensions" yaml:"extensions" toml:"extensions"`
	Processes       []ProcessDisplay        `json:"processes" yaml:"processes" toml:"processes"`
	Rebasable       bool                    `json:"rebasable" yaml:"rebasable" toml:"rebasable"`
}

type InspectOutput struct {
	ImageName string       `json:"image_name" yaml:"image_name" toml:"image_name"`
	Remote    *InfoDisplay `json:"remote_info" yaml:"remote_info" toml:"remote_info"`
	Local     *InfoDisplay `json:"local_info" yaml:"local_info" toml:"local_info"`
}

func NewInfoDisplay(info *client.ImageInfo, generalInfo GeneralInfo) *InfoDisplay {
	if info == nil {
		return nil
	}
	if info != nil && info.Extensions != nil {
		return &InfoDisplay{
			StackID:         info.StackID,
			Base:            displayBase(info.Base),
			RunImageMirrors: displayMirrors(info, generalInfo),
			Buildpacks:      displayBuildpacks(info.Buildpacks),
			Extensions:      displayExtensions(info.Extensions),
			Processes:       displayProcesses(info.Processes),
			Rebasable:       info.Rebasable,
		}
	}
	return &InfoDisplay{
		StackID:         info.StackID,
		Base:            displayBase(info.Base),
		RunImageMirrors: displayMirrors(info, generalInfo),
		Buildpacks:      displayBuildpacks(info.Buildpacks),
		Processes:       displayProcesses(info.Processes),
		Rebasable:       info.Rebasable,
	}
}

//
// private functions
//

func getConfigMirrors(info *client.ImageInfo, imageMirrors []config.RunImage) []string {
	var runImage string
	if info != nil {
		runImage = info.Stack.RunImage.Image
	}

	for _, ri := range imageMirrors {
		if ri.Image == runImage {
			return ri.Mirrors
		}
	}
	return nil
}

func displayBase(base files.RunImageForRebase) BaseDisplay {
	return BaseDisplay{
		TopLayer:  base.TopLayer,
		Reference: base.Reference,
	}
}

func displayMirrors(info *client.ImageInfo, generalInfo GeneralInfo) []RunImageMirrorDisplay {
	// add all user configured run images, then add run images provided by info
	var result []RunImageMirrorDisplay
	if info == nil {
		return result
	}

	cfgMirrors := getConfigMirrors(info, generalInfo.RunImageMirrors)
	for _, mirror := range cfgMirrors {
		if mirror != "" {
			result = append(result, RunImageMirrorDisplay{
				Name:           mirror,
				UserConfigured: true,
			})
		}
	}

	// Add run image as named by the stack.
	if info.Stack.RunImage.Image != "" {
		result = append(result, RunImageMirrorDisplay{
			Name:           info.Stack.RunImage.Image,
			UserConfigured: false,
		})
	}

	for _, mirror := range info.Stack.RunImage.Mirrors {
		if mirror != "" {
			result = append(result, RunImageMirrorDisplay{
				Name:           mirror,
				UserConfigured: false,
			})
		}
	}

	return result
}

func displayBuildpacks(buildpacks []buildpack.GroupElement) []dist.ModuleInfo {
	var result []dist.ModuleInfo
	for _, buildpack := range buildpacks {
		result = append(result, dist.ModuleInfo{
			ID:       buildpack.ID,
			Version:  buildpack.Version,
			Homepage: buildpack.Homepage,
		})
	}
	return result
}

func displayExtensions(extensions []buildpack.GroupElement) []dist.ModuleInfo {
	var result []dist.ModuleInfo
	for _, extension := range extensions {
		result = append(result, dist.ModuleInfo{
			ID:       extension.ID,
			Version:  extension.Version,
			Homepage: extension.Homepage,
		})
	}
	return result
}

func displayProcesses(details client.ProcessDetails) []ProcessDisplay {
	var result []ProcessDisplay
	detailsArray := details.OtherProcesses
	if details.DefaultProcess != nil {
		result = append(result, convertToDisplay(*details.DefaultProcess, true))
	}

	for _, detail := range detailsArray {
		result = append(result, convertToDisplay(detail, false))
	}
	return result
}

func convertToDisplay(proc launch.Process, isDefault bool) ProcessDisplay {
	var shell string
	switch proc.Direct {
	case true:
		shell = ""
	case false:
		shell = "bash"
	}
	var argsToUse []string
	if len(proc.Command.Entries) > 1 {
		argsToUse = proc.Command.Entries[1:]
	}
	argsToUse = append(argsToUse, proc.Args...)
	result := ProcessDisplay{
		Type:    proc.Type,
		Shell:   shell,
		Command: proc.Command.Entries[0],
		Default: isDefault,
		Args:    argsToUse, // overridable args are supported for platform API >= 0.10 with buildpack API >= 0.9, but we can't determine the buildpack API from the metadata label (to be fixed in platform 0.11)
		WorkDir: proc.WorkingDirectory,
	}

	return result
}
