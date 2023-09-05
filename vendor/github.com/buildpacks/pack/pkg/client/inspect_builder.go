package client

import (
	"errors"

	pubbldr "github.com/buildpacks/pack/builder"

	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/image"
)

// BuilderInfo is a collection of metadata describing a builder created using client.
type BuilderInfo struct {
	// Human readable, description of a builder.
	Description string

	// Stack name used by the builder.
	Stack string

	// List of Stack mixins, this information is provided by Stack variable.
	Mixins []string

	// RunImage provided by the builder.
	RunImages []pubbldr.RunImageConfig

	// All buildpacks included within the builder.
	Buildpacks []dist.ModuleInfo

	// Detailed ordering of buildpacks and nested buildpacks where depth is specified.
	Order pubbldr.DetectionOrder

	// Listing of all buildpack layers in a builder.
	// All elements in the Buildpacks variable are represented in this
	// object.
	BuildpackLayers dist.ModuleLayers

	// Lifecycle provides the following API versioning information for a builder:
	// - Lifecycle Version used in this builder,
	// - Platform API,
	// - Buildpack API.
	Lifecycle builder.LifecycleDescriptor

	// Name and Version information from tooling used
	// to produce this builder.
	CreatedBy builder.CreatorMetadata

	// All extensions included within the builder.
	Extensions []dist.ModuleInfo

	// Detailed ordering of extensions.
	OrderExtensions pubbldr.DetectionOrder
}

// BuildpackInfoKey contains all information needed to determine buildpack equivalence.
type BuildpackInfoKey struct {
	ID      string
	Version string
}

type BuilderInspectionConfig struct {
	OrderDetectionDepth int
}

type BuilderInspectionModifier func(config *BuilderInspectionConfig)

func WithDetectionOrderDepth(depth int) BuilderInspectionModifier {
	return func(config *BuilderInspectionConfig) {
		config.OrderDetectionDepth = depth
	}
}

// InspectBuilder reads label metadata of a local or remote builder image. It initializes a BuilderInfo
// object with this metadata, and returns it. This method will error if the name image cannot be found
// both locally and remotely, or if the found image does not contain the proper labels.
func (c *Client) InspectBuilder(name string, daemon bool, modifiers ...BuilderInspectionModifier) (*BuilderInfo, error) {
	inspector := builder.NewInspector(
		builder.NewImageFetcherWrapper(c.imageFetcher),
		builder.NewLabelManagerProvider(),
		builder.NewDetectionOrderCalculator(),
	)

	inspectionConfig := BuilderInspectionConfig{OrderDetectionDepth: pubbldr.OrderDetectionNone}
	for _, mod := range modifiers {
		mod(&inspectionConfig)
	}

	info, err := inspector.Inspect(name, daemon, inspectionConfig.OrderDetectionDepth)
	if err != nil {
		if errors.Is(err, image.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &BuilderInfo{
		Description:     info.Description,
		Stack:           info.StackID,
		Mixins:          info.Mixins,
		RunImages:       info.RunImages,
		Buildpacks:      info.Buildpacks,
		Order:           info.Order,
		BuildpackLayers: info.BuildpackLayers,
		Lifecycle:       info.Lifecycle,
		CreatedBy:       info.CreatedBy,
		Extensions:      info.Extensions,
		OrderExtensions: info.OrderExtensions,
	}, nil
}
