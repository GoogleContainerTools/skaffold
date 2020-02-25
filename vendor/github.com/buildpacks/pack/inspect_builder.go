package pack

import (
	"context"
	"strings"

	"github.com/pkg/errors"

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
	Buildpacks      []builder.BuildpackMetadata
	Order           dist.Order
	Lifecycle       builder.LifecycleDescriptor
	CreatedBy       builder.CreatorMetadata
}

type BuildpackInfo struct {
	ID      string
	Version string
}

func (c *Client) InspectBuilder(name string, daemon bool) (*BuilderInfo, error) {
	img, err := c.imageFetcher.Fetch(context.Background(), name, daemon, false)
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

	return &BuilderInfo{
		Description:     bldr.Description(),
		Stack:           bldr.StackID,
		Mixins:          append(commonMixins, buildMixins...),
		RunImage:        bldr.Stack().RunImage.Image,
		RunImageMirrors: bldr.Stack().RunImage.Mirrors,
		Buildpacks:      bldr.Buildpacks(),
		Order:           bldr.Order(),
		Lifecycle:       bldr.LifecycleDescriptor(),
		CreatedBy:       bldr.CreatedBy(),
	}, nil
}
