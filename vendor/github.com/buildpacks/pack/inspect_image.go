package pack

import (
	"context"

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

func (c *Client) InspectImage(name string, daemon bool) (*ImageInfo, error) {
	img, err := c.imageFetcher.Fetch(context.Background(), name, daemon, false)
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

	defaultProcessType, err := img.Env("CNB_PROCESS_TYPE")
	if err != nil || defaultProcessType == "" {
		defaultProcessType = "web"
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
