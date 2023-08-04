package builder

import (
	"context"
	"fmt"
	"sort"
	"strings"

	pubbldr "github.com/buildpacks/pack/builder"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/image"
)

type Info struct {
	Description     string
	StackID         string
	Mixins          []string
	RunImage        string
	RunImageMirrors []string
	Buildpacks      []dist.BuildpackInfo
	Order           pubbldr.DetectionOrder
	BuildpackLayers dist.BuildpackLayers
	Lifecycle       LifecycleDescriptor
	CreatedBy       CreatorMetadata
}

type Inspectable interface {
	Label(name string) (string, error)
}

type InspectableFetcher interface {
	Fetch(ctx context.Context, name string, options image.FetchOptions) (Inspectable, error)
}

type LabelManagerFactory interface {
	BuilderLabelManager(inspectable Inspectable) LabelInspector
}

type LabelInspector interface {
	Metadata() (Metadata, error)
	StackID() (string, error)
	Mixins() ([]string, error)
	Order() (dist.Order, error)
	BuildpackLayers() (dist.BuildpackLayers, error)
}

type DetectionCalculator interface {
	Order(topOrder dist.Order, layers dist.BuildpackLayers, depth int) (pubbldr.DetectionOrder, error)
}

type Inspector struct {
	imageFetcher             InspectableFetcher
	labelManagerFactory      LabelManagerFactory
	detectionOrderCalculator DetectionCalculator
}

func NewInspector(fetcher InspectableFetcher, factory LabelManagerFactory, calculator DetectionCalculator) *Inspector {
	return &Inspector{
		imageFetcher:             fetcher,
		labelManagerFactory:      factory,
		detectionOrderCalculator: calculator,
	}
}

func (i *Inspector) Inspect(name string, daemon bool, orderDetectionDepth int) (Info, error) {
	inspectable, err := i.imageFetcher.Fetch(context.Background(), name, image.FetchOptions{Daemon: daemon, PullPolicy: image.PullNever})
	if err != nil {
		return Info{}, fmt.Errorf("fetching builder image: %w", err)
	}

	labelManager := i.labelManagerFactory.BuilderLabelManager(inspectable)

	metadata, err := labelManager.Metadata()
	if err != nil {
		return Info{}, fmt.Errorf("reading image metadata: %w", err)
	}

	stackID, err := labelManager.StackID()
	if err != nil {
		return Info{}, fmt.Errorf("reading image stack id: %w", err)
	}

	mixins, err := labelManager.Mixins()
	if err != nil {
		return Info{}, fmt.Errorf("reading image mixins: %w", err)
	}

	var commonMixins, buildMixins []string
	commonMixins = []string{}
	for _, mixin := range mixins {
		if strings.HasPrefix(mixin, "build:") {
			buildMixins = append(buildMixins, mixin)
		} else {
			commonMixins = append(commonMixins, mixin)
		}
	}

	order, err := labelManager.Order()
	if err != nil {
		return Info{}, fmt.Errorf("reading image order: %w", err)
	}

	layers, err := labelManager.BuildpackLayers()
	if err != nil {
		return Info{}, fmt.Errorf("reading image buildpack layers: %w", err)
	}

	detectionOrder, err := i.detectionOrderCalculator.Order(order, layers, orderDetectionDepth)
	if err != nil {
		return Info{}, fmt.Errorf("calculating detection order: %w", err)
	}

	lifecycle := CompatDescriptor(LifecycleDescriptor{
		Info: LifecycleInfo{Version: metadata.Lifecycle.Version},
		API:  metadata.Lifecycle.API,
		APIs: metadata.Lifecycle.APIs,
	})

	return Info{
		Description:     metadata.Description,
		StackID:         stackID,
		Mixins:          append(commonMixins, buildMixins...),
		RunImage:        metadata.Stack.RunImage.Image,
		RunImageMirrors: metadata.Stack.RunImage.Mirrors,
		Buildpacks:      sortBuildPacksByID(uniqueBuildpacks(metadata.Buildpacks)),
		Order:           detectionOrder,
		BuildpackLayers: layers,
		Lifecycle:       lifecycle,
		CreatedBy:       metadata.CreatedBy,
	}, nil
}

func uniqueBuildpacks(buildpacks []dist.BuildpackInfo) []dist.BuildpackInfo {
	foundBuildpacks := map[string]interface{}{}
	var uniqueBuildpacks []dist.BuildpackInfo

	for _, bp := range buildpacks {
		_, ok := foundBuildpacks[bp.FullName()]
		if !ok {
			uniqueBuildpacks = append(uniqueBuildpacks, bp)
			foundBuildpacks[bp.FullName()] = true
		}
	}

	return uniqueBuildpacks
}

func sortBuildPacksByID(buildpacks []dist.BuildpackInfo) []dist.BuildpackInfo {
	sort.Slice(buildpacks, func(i int, j int) bool {
		if buildpacks[i].ID == buildpacks[j].ID {
			return buildpacks[i].Version < buildpacks[j].Version
		}

		return buildpacks[i].ID < buildpacks[j].ID
	})

	return buildpacks
}
