package pack

import (
	"context"
	"strings"

	"github.com/pkg/errors"

	"github.com/buildpacks/pack/config"
	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/internal/dist"
	"github.com/buildpacks/pack/internal/image"
	"github.com/buildpacks/pack/internal/style"
)

type BuilderInfo struct {
	Description     string
	Stack           string
	Mixins          []string
	RunImage        string
	RunImageMirrors []string
	Buildpacks      []dist.BuildpackInfo
	Order           dist.Order
	BuildpackLayers dist.BuildpackLayers
	Lifecycle       builder.LifecycleDescriptor
	CreatedBy       builder.CreatorMetadata
}

type BuildpackInfoKey struct {
	ID      string
	Version string
}

func (c *Client) InspectBuilder(name string, daemon bool) (*BuilderInfo, error) {
	img, err := c.imageFetcher.Fetch(context.Background(), name, daemon, config.PullNever)
	if err != nil {
		if errors.Cause(err) == image.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	bldr, err := builder.FromImage(img)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid builder %s", style.Symbol(name))
	}

	var commonMixins, buildMixins []string
	commonMixins = []string{}
	for _, mixin := range bldr.Mixins() {
		if strings.HasPrefix(mixin, "build:") {
			buildMixins = append(buildMixins, mixin)
		} else {
			commonMixins = append(commonMixins, mixin)
		}
	}

	var bpLayers dist.BuildpackLayers
	if _, err := dist.GetLabel(img, dist.BuildpackLayersLabel, &bpLayers); err != nil {
		return nil, err
	}

	return &BuilderInfo{
		Description:     bldr.Description(),
		Stack:           bldr.StackID,
		Mixins:          append(commonMixins, buildMixins...),
		RunImage:        bldr.Stack().RunImage.Image,
		RunImageMirrors: bldr.Stack().RunImage.Mirrors,
		Buildpacks:      uniqueBuildpacks(bldr.Buildpacks()),
		Order:           bldr.Order(),
		BuildpackLayers: bpLayers,
		Lifecycle:       bldr.LifecycleDescriptor(),
		CreatedBy:       bldr.CreatedBy(),
	}, nil
}

func uniqueBuildpacks(buildpacks []dist.BuildpackInfo) []dist.BuildpackInfo {
	buildpacksSet := map[BuildpackInfoKey]int{}
	homePageSet := map[BuildpackInfoKey]string{}
	for _, buildpack := range buildpacks {
		key := BuildpackInfoKey{
			ID:      buildpack.ID,
			Version: buildpack.Version,
		}
		_, ok := buildpacksSet[key]
		if !ok {
			buildpacksSet[key] = len(buildpacksSet)
			homePageSet[key] = buildpack.Homepage
		}
	}
	result := make([]dist.BuildpackInfo, len(buildpacksSet))
	for buildpackKey, index := range buildpacksSet {
		result[index] = dist.BuildpackInfo{
			ID:       buildpackKey.ID,
			Version:  buildpackKey.Version,
			Homepage: homePageSet[buildpackKey],
		}
	}

	return result
}
