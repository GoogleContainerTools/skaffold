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
	RunImages       []pubbldr.RunImageConfig
	Buildpacks      []dist.ModuleInfo
	Order           pubbldr.DetectionOrder
	BuildpackLayers dist.ModuleLayers
	Lifecycle       LifecycleDescriptor
	CreatedBy       CreatorMetadata
	Extensions      []dist.ModuleInfo
	OrderExtensions pubbldr.DetectionOrder
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
	BuildpackLayers() (dist.ModuleLayers, error)
	OrderExtensions() (dist.Order, error)
}

type DetectionCalculator interface {
	Order(topOrder dist.Order, layers dist.ModuleLayers, depth int) (pubbldr.DetectionOrder, error)
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

	stackID, _ := labelManager.StackID() // ignore error because stack is optional

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

	orderExtensions, err := labelManager.OrderExtensions()
	if err != nil {
		return Info{}, fmt.Errorf("reading image order extensions: %w", err)
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

	detectionOrderExtensions := orderExttoPubbldrDetectionOrderExt(orderExtensions)

	lifecycle := CompatDescriptor(LifecycleDescriptor{
		Info: LifecycleInfo{Version: metadata.Lifecycle.Version},
		API:  metadata.Lifecycle.API,
		APIs: metadata.Lifecycle.APIs,
	})

	var runImages []pubbldr.RunImageConfig
	for _, ri := range metadata.RunImages {
		runImages = append(runImages, pubbldr.RunImageConfig{
			Image:   ri.Image,
			Mirrors: ri.Mirrors,
		})
	}
	addStackRunImage := true
	for _, ri := range runImages {
		if ri.Image == metadata.Stack.RunImage.Image {
			addStackRunImage = false
		}
	}
	if addStackRunImage && metadata.Stack.RunImage.Image != "" {
		runImages = append(runImages, pubbldr.RunImageConfig{
			Image:   metadata.Stack.RunImage.Image,
			Mirrors: metadata.Stack.RunImage.Mirrors,
		})
	}

	return Info{
		Description:     metadata.Description,
		StackID:         stackID,
		Mixins:          append(commonMixins, buildMixins...),
		RunImages:       runImages,
		Buildpacks:      sortBuildPacksByID(uniqueBuildpacks(metadata.Buildpacks)),
		Order:           detectionOrder,
		BuildpackLayers: layers,
		Lifecycle:       lifecycle,
		CreatedBy:       metadata.CreatedBy,
		Extensions:      metadata.Extensions,
		OrderExtensions: detectionOrderExtensions,
	}, nil
}

func orderExttoPubbldrDetectionOrderExt(orderExt dist.Order) pubbldr.DetectionOrder {
	var detectionOrderExt pubbldr.DetectionOrder

	for _, orderEntry := range orderExt {
		var detectionOrderEntry pubbldr.DetectionOrderEntry
		for _, moduleRef := range orderEntry.Group {
			detectionOrderEntry.ModuleRef = moduleRef
		}
		detectionOrderExt = append(detectionOrderExt, detectionOrderEntry)
	}

	return detectionOrderExt
}

func uniqueBuildpacks(buildpacks []dist.ModuleInfo) []dist.ModuleInfo {
	foundBuildpacks := map[string]interface{}{}
	var uniqueBuildpacks []dist.ModuleInfo

	for _, bp := range buildpacks {
		_, ok := foundBuildpacks[bp.FullName()]
		if !ok {
			uniqueBuildpacks = append(uniqueBuildpacks, bp)
			foundBuildpacks[bp.FullName()] = true
		}
	}

	return uniqueBuildpacks
}

func sortBuildPacksByID(buildpacks []dist.ModuleInfo) []dist.ModuleInfo {
	sort.Slice(buildpacks, func(i int, j int) bool {
		if buildpacks[i].ID == buildpacks[j].ID {
			return buildpacks[i].Version < buildpacks[j].Version
		}

		return buildpacks[i].ID < buildpacks[j].ID
	})

	return buildpacks
}
