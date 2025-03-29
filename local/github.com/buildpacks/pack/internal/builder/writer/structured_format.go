package writer

import (
	"fmt"

	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/client"

	pubbldr "github.com/buildpacks/pack/builder"
	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/logging"
)

type InspectOutput struct {
	SharedBuilderInfo
	RemoteInfo *BuilderInfo `json:"remote_info" yaml:"remote_info" toml:"remote_info"`
	LocalInfo  *BuilderInfo `json:"local_info" yaml:"local_info" toml:"local_info"`
}

type RunImage struct {
	Name           string `json:"name" yaml:"name" toml:"name"`
	UserConfigured bool   `json:"user_configured,omitempty" yaml:"user_configured,omitempty" toml:"user_configured,omitempty"`
}

type Lifecycle struct {
	builder.LifecycleInfo `yaml:"lifecycleinfo,inline"`
	BuildpackAPIs         builder.APIVersions `json:"buildpack_apis" yaml:"buildpack_apis" toml:"buildpack_apis"`
	PlatformAPIs          builder.APIVersions `json:"platform_apis" yaml:"platform_apis" toml:"platform_apis"`
}

type Stack struct {
	ID     string   `json:"id" yaml:"id" toml:"id"`
	Mixins []string `json:"mixins,omitempty" yaml:"mixins,omitempty" toml:"mixins,omitempty"`
}

type BuilderInfo struct {
	Description            string                  `json:"description,omitempty" yaml:"description,omitempty" toml:"description,omitempty"`
	CreatedBy              builder.CreatorMetadata `json:"created_by" yaml:"created_by" toml:"created_by"`
	Stack                  *Stack                  `json:"stack,omitempty" yaml:"stack,omitempty" toml:"stack,omitempty"`
	Lifecycle              Lifecycle               `json:"lifecycle" yaml:"lifecycle" toml:"lifecycle"`
	RunImages              []RunImage              `json:"run_images" yaml:"run_images" toml:"run_images"`
	Buildpacks             []dist.ModuleInfo       `json:"buildpacks" yaml:"buildpacks" toml:"buildpacks"`
	pubbldr.DetectionOrder `json:"detection_order" yaml:"detection_order" toml:"detection_order"`
	Extensions             []dist.ModuleInfo      `json:"extensions,omitempty" yaml:"extensions,omitempty" toml:"extensions,omitempty"`
	OrderExtensions        pubbldr.DetectionOrder `json:"order_extensions,omitempty" yaml:"order_extensions,omitempty" toml:"order_extensions,omitempty"`
}

type StructuredFormat struct {
	MarshalFunc func(interface{}) ([]byte, error)
}

func (w *StructuredFormat) Print(
	logger logging.Logger,
	localRunImages []config.RunImage,
	local, remote *client.BuilderInfo,
	localErr, remoteErr error,
	builderInfo SharedBuilderInfo,
) error {
	if localErr != nil {
		return fmt.Errorf("preparing output for %s: %w", style.Symbol(builderInfo.Name), localErr)
	}

	if remoteErr != nil {
		return fmt.Errorf("preparing output for %s: %w", style.Symbol(builderInfo.Name), remoteErr)
	}

	outputInfo := InspectOutput{SharedBuilderInfo: builderInfo}

	if local != nil {
		var stack *Stack
		if local.Stack != "" {
			stack = &Stack{ID: local.Stack}
		}

		if logger.IsVerbose() {
			stack.Mixins = local.Mixins
		}

		outputInfo.LocalInfo = &BuilderInfo{
			Description: local.Description,
			CreatedBy:   local.CreatedBy,
			Stack:       stack,
			Lifecycle: Lifecycle{
				LifecycleInfo: local.Lifecycle.Info,
				BuildpackAPIs: local.Lifecycle.APIs.Buildpack,
				PlatformAPIs:  local.Lifecycle.APIs.Platform,
			},
			RunImages:       runImages(local.RunImages, localRunImages),
			Buildpacks:      local.Buildpacks,
			DetectionOrder:  local.Order,
			Extensions:      local.Extensions,
			OrderExtensions: local.OrderExtensions,
		}
	}

	if remote != nil {
		var stack *Stack
		if remote.Stack != "" {
			stack = &Stack{ID: remote.Stack}
		}

		if logger.IsVerbose() {
			stack.Mixins = remote.Mixins
		}

		outputInfo.RemoteInfo = &BuilderInfo{
			Description: remote.Description,
			CreatedBy:   remote.CreatedBy,
			Stack:       stack,
			Lifecycle: Lifecycle{
				LifecycleInfo: remote.Lifecycle.Info,
				BuildpackAPIs: remote.Lifecycle.APIs.Buildpack,
				PlatformAPIs:  remote.Lifecycle.APIs.Platform,
			},
			RunImages:       runImages(remote.RunImages, localRunImages),
			Buildpacks:      remote.Buildpacks,
			DetectionOrder:  remote.Order,
			Extensions:      remote.Extensions,
			OrderExtensions: remote.OrderExtensions,
		}
	}

	if outputInfo.LocalInfo == nil && outputInfo.RemoteInfo == nil {
		return fmt.Errorf("unable to find builder %s locally or remotely", style.Symbol(builderInfo.Name))
	}

	var (
		output []byte
		err    error
	)
	if output, err = w.MarshalFunc(outputInfo); err != nil {
		return fmt.Errorf("untested, unexpected failure while marshaling: %w", err)
	}

	logger.Info(string(output))

	return nil
}

func runImages(runImages []pubbldr.RunImageConfig, localRunImages []config.RunImage) []RunImage {
	images := []RunImage{}

	for _, i := range localRunImages {
		for _, runImage := range runImages {
			if i.Image == runImage.Image {
				for _, m := range i.Mirrors {
					images = append(images, RunImage{Name: m, UserConfigured: true})
				}
			}
		}
	}

	for _, runImage := range runImages {
		images = append(images, RunImage{Name: runImage.Image})
		for _, m := range runImage.Mirrors {
			images = append(images, RunImage{Name: m})
		}
	}

	return images
}
