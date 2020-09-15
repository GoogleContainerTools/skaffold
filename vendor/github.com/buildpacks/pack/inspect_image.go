package pack

import (
	"context"
	"strings"

	"github.com/buildpacks/pack/config"

	"github.com/Masterminds/semver"
	"github.com/buildpacks/lifecycle"
	"github.com/buildpacks/lifecycle/launch"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/dist"
	"github.com/buildpacks/pack/internal/image"
)

type ImageInfo struct {
	StackID    string
	Buildpacks []lifecycle.Buildpack
	Base       lifecycle.RunImageMetadata
	BOM        []lifecycle.BOMEntry
	Stack      lifecycle.StackMetadata
	Processes  ProcessDetails
}

type ProcessDetails struct {
	DefaultProcess *launch.Process
	OtherProcesses []launch.Process
}

// Deserialize just the subset of fields we need to avoid breaking changes
type layersMetadata struct {
	RunImage lifecycle.RunImageMetadata `json:"runImage" toml:"run-image"`
	Stack    lifecycle.StackMetadata    `json:"stack" toml:"stack"`
}

const (
	platformAPIEnv            = "CNB_PLATFORM_API"
	cnbProcessEnv             = "CNB_PROCESS_TYPE"
	launcherEntrypoint        = "/cnb/lifecycle/launcher"
	windowsLauncherEntrypoint = `c:\cnb\lifecycle\launcher.exe`
	entrypointPrefix          = "/cnb/process/"
	windowsEntrypointPrefix   = `c:\cnb\process\`
	defaultProcess            = "web"
	fallbackPlatformAPI       = "0.3"
	windowsPrefix             = "c:"
)

func (c *Client) InspectImage(name string, daemon bool) (*ImageInfo, error) {
	img, err := c.imageFetcher.Fetch(context.Background(), name, daemon, config.PullNever)
	if err != nil {
		if errors.Cause(err) == image.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	var layersMd layersMetadata
	if _, err := dist.GetLabel(img, lifecycle.LayerMetadataLabel, &layersMd); err != nil {
		return nil, err
	}

	var buildMD lifecycle.BuildMetadata
	if _, err := dist.GetLabel(img, lifecycle.BuildMetadataLabel, &buildMD); err != nil {
		return nil, err
	}

	minimumBaseImageReferenceVersion := semver.MustParse("0.5.0")
	actualLauncherVersion, err := semver.NewVersion(buildMD.Launcher.Version)

	if err == nil && actualLauncherVersion.LessThan(minimumBaseImageReferenceVersion) {
		layersMd.RunImage.Reference = ""
	}

	stackID, err := img.Label(lifecycle.StackIDLabel)
	if err != nil {
		return nil, err
	}

	platformAPI, err := img.Env(platformAPIEnv)
	if err != nil {
		return nil, errors.Wrap(err, "reading platform api")
	}

	if platformAPI == "" {
		platformAPI = fallbackPlatformAPI
	}

	platformAPIVersion, err := semver.NewVersion(platformAPI)
	if err != nil {
		return nil, errors.Wrap(err, "parsing platform api version")
	}

	var defaultProcessType string
	if platformAPIVersion.LessThan(semver.MustParse("0.4")) {
		defaultProcessType, err = img.Env(cnbProcessEnv)
		if err != nil || defaultProcessType == "" {
			defaultProcessType = defaultProcess
		}
	} else {
		inspect, _, err := c.docker.ImageInspectWithRaw(context.TODO(), name)
		if err != nil {
			return nil, errors.Wrap(err, "reading image")
		}

		entrypoint := inspect.Config.Entrypoint
		if len(entrypoint) > 0 && entrypoint[0] != launcherEntrypoint && entrypoint[0] != windowsLauncherEntrypoint {
			process := entrypoint[0]
			if strings.HasPrefix(process, windowsPrefix) {
				process = strings.TrimPrefix(process, windowsEntrypointPrefix)
				process = strings.TrimSuffix(process, ".exe") // Trim .exe for Windows support
			} else {
				process = strings.TrimPrefix(process, entrypointPrefix)
			}

			defaultProcessType = process
		}
	}

	var processDetails ProcessDetails
	for _, proc := range buildMD.Processes {
		proc := proc
		if proc.Type == defaultProcessType {
			processDetails.DefaultProcess = &proc
			continue
		}
		processDetails.OtherProcesses = append(processDetails.OtherProcesses, proc)
	}

	return &ImageInfo{
		StackID:    stackID,
		Stack:      layersMd.Stack,
		Base:       layersMd.RunImage,
		BOM:        buildMD.BOM,
		Buildpacks: buildMD.Buildpacks,
		Processes:  processDetails,
	}, nil
}
