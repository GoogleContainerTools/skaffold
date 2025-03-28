package builder_test

import (
	"errors"
	"testing"

	"github.com/buildpacks/lifecycle/api"

	pubbldr "github.com/buildpacks/pack/builder"
	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/internal/builder/fakes"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/image"
	h "github.com/buildpacks/pack/testhelpers"

	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

const (
	testBuilderName        = "test-builder"
	testBuilderDescription = "Test Builder Description"
	testStackID            = "test-builder-stack-id"
	testRunImage           = "test/run-image"
	testStackRunImage      = "test/stack-run-image"
)

var (
	testTopNestedBuildpack = dist.ModuleInfo{
		ID:      "test.top.nested",
		Version: "test.top.nested.version",
	}
	testNestedBuildpack = dist.ModuleInfo{
		ID:       "test.nested",
		Version:  "test.nested.version",
		Homepage: "http://geocities.com/top-bp",
	}
	testBuildpack = dist.ModuleInfo{
		ID:      "test.bp.two",
		Version: "test.bp.two.version",
	}
	testBuildpackVersionTwo = dist.ModuleInfo{
		ID:      "test.bp.two",
		Version: "test.bp.two.version-2",
	}
	testBuildpacks = []dist.ModuleInfo{
		testBuildpack,
		testNestedBuildpack,
		testTopNestedBuildpack,
	}
	testLifecycleInfo = builder.LifecycleInfo{
		Version: builder.VersionMustParse("1.2.3"),
	}
	testBuildpackVersions = builder.APIVersions{
		Deprecated: builder.APISet{api.MustParse("0.1")},
		Supported:  builder.APISet{api.MustParse("1.2"), api.MustParse("1.3")},
	}
	testPlatformVersions = builder.APIVersions{
		Supported: builder.APISet{api.MustParse("2.3"), api.MustParse("2.4")},
	}
	inspectTestLifecycle = builder.LifecycleMetadata{
		LifecycleInfo: testLifecycleInfo,
		APIs: builder.LifecycleAPIs{
			Buildpack: testBuildpackVersions,
			Platform:  testPlatformVersions,
		},
	}
	testCreatorData = builder.CreatorMetadata{
		Name:    "pack",
		Version: "1.2.3",
	}
	testMetadata = builder.Metadata{
		Description: testBuilderDescription,
		Buildpacks:  testBuildpacks,
		Stack:       testStack,
		Lifecycle:   inspectTestLifecycle,
		CreatedBy:   testCreatorData,
		RunImages: []builder.RunImageMetadata{
			{
				Image:   testRunImage,
				Mirrors: testRunImageMirrors,
			},
		},
	}
	testMixins          = []string{"build:mixinA", "mixinX", "mixinY"}
	expectedTestMixins  = []string{"mixinX", "mixinY", "build:mixinA"}
	testRunImageMirrors = []string{"test/first-run-image-mirror", "test/second-run-image-mirror"}
	testStack           = builder.StackMetadata{
		RunImage: builder.RunImageMetadata{
			Image:   testStackRunImage,
			Mirrors: testRunImageMirrors,
		},
	}
	testOrder = dist.Order{
		dist.OrderEntry{Group: []dist.ModuleRef{
			{ModuleInfo: testBuildpack, Optional: false},
		}},
		dist.OrderEntry{Group: []dist.ModuleRef{
			{ModuleInfo: testNestedBuildpack, Optional: false},
			{ModuleInfo: testTopNestedBuildpack, Optional: true},
		}},
	}
	testOrderExtensions = dist.Order{
		dist.OrderEntry{Group: []dist.ModuleRef{
			{ModuleInfo: testBuildpack, Optional: false},
		}},
		dist.OrderEntry{Group: []dist.ModuleRef{
			{ModuleInfo: testNestedBuildpack, Optional: false},
			{ModuleInfo: testTopNestedBuildpack, Optional: true},
		}},
	}
	testLayers = dist.ModuleLayers{
		"test.top.nested": {
			"test.top.nested.version": {
				API:         api.MustParse("0.2"),
				Order:       testOrder,
				LayerDiffID: "sha256:test.top.nested.sha256",
				Homepage:    "http://geocities.com/top-bp",
			},
		},
		"test.bp.two": {
			"test.bp.two.version": {
				API:         api.MustParse("0.2"),
				Stacks:      []dist.Stack{{ID: "test.stack.id"}},
				LayerDiffID: "sha256:test.bp.two.sha256",
				Homepage:    "http://geocities.com/cool-bp",
			},
		},
	}
	expectedTestLifecycle = builder.LifecycleDescriptor{
		Info: testLifecycleInfo,
		API: builder.LifecycleAPI{
			BuildpackVersion: api.MustParse("0.1"),
			PlatformVersion:  api.MustParse("2.3"),
		},
		APIs: builder.LifecycleAPIs{
			Buildpack: testBuildpackVersions,
			Platform:  testPlatformVersions,
		},
	}
	expectedDetectionTestOrder = pubbldr.DetectionOrder{
		{
			ModuleRef: dist.ModuleRef{
				ModuleInfo: testBuildpack,
			},
		},
		{
			ModuleRef: dist.ModuleRef{
				ModuleInfo: testTopNestedBuildpack,
			},
			GroupDetectionOrder: pubbldr.DetectionOrder{
				{
					ModuleRef: dist.ModuleRef{
						ModuleInfo: testNestedBuildpack,
					},
				},
			},
		},
	}
)

func TestInspect(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "testInspect", testInspect, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testInspect(t *testing.T, when spec.G, it spec.S) {
	when("Inspect", func() {
		var assert = h.NewAssertionManager(t)

		it("calls Fetch on inspectableFetcher with expected arguments", func() {
			fetcher := newDefaultInspectableFetcher()

			inspector := builder.NewInspector(fetcher, newDefaultLabelManagerFactory(), newDefaultDetectionCalculator())
			_, err := inspector.Inspect(testBuilderName, true, pubbldr.OrderDetectionNone)
			assert.Nil(err)

			assert.Equal(fetcher.CallCount, 1)
			assert.Equal(fetcher.ReceivedName, testBuilderName)
			assert.Equal(fetcher.ReceivedDaemon, true)
			assert.Equal(fetcher.ReceivedPullPolicy, image.PullNever)
		})

		it("instantiates a builder label manager with the correct inspectable", func() {
			inspectable := newNoOpInspectable()

			fetcher := &fakes.FakeInspectableFetcher{
				InspectableToReturn: inspectable,
			}

			labelManagerFactory := newDefaultLabelManagerFactory()

			inspector := builder.NewInspector(fetcher, labelManagerFactory, newDefaultDetectionCalculator())
			_, err := inspector.Inspect(testBuilderName, true, pubbldr.OrderDetectionNone)
			assert.Nil(err)

			assert.Equal(labelManagerFactory.ReceivedInspectable, inspectable)
		})

		it("calls `Order` on detectionCalculator with expected arguments", func() {
			detectionOrderCalculator := newDefaultDetectionCalculator()

			inspector := builder.NewInspector(
				newDefaultInspectableFetcher(),
				newDefaultLabelManagerFactory(),
				detectionOrderCalculator,
			)
			_, err := inspector.Inspect(testBuilderName, true, 3)
			assert.Nil(err)

			assert.Equal(detectionOrderCalculator.ReceivedTopOrder, testOrder)
			assert.Equal(detectionOrderCalculator.ReceivedLayers, testLayers)
			assert.Equal(detectionOrderCalculator.ReceivedDepth, 3)
		})

		it("returns Info object with expected fields", func() {
			fetcher := newDefaultInspectableFetcher()

			inspector := builder.NewInspector(fetcher, newDefaultLabelManagerFactory(), newDefaultDetectionCalculator())
			info, err := inspector.Inspect(testBuilderName, true, pubbldr.OrderDetectionNone)
			assert.Nil(err)

			assert.Equal(info.Description, testBuilderDescription)
			assert.Equal(info.StackID, testStackID)
			assert.Equal(info.Mixins, expectedTestMixins)
			assert.Equal(len(info.RunImages), 2)
			assert.Equal(info.RunImages[0].Image, testRunImage)
			assert.Equal(info.RunImages[1].Image, testStackRunImage)
			assert.Equal(info.RunImages[0].Mirrors, testRunImageMirrors)
			assert.Equal(info.RunImages[1].Mirrors, testRunImageMirrors)
			assert.Equal(info.Buildpacks, testBuildpacks)
			assert.Equal(info.Order, expectedDetectionTestOrder)
			assert.Equal(info.BuildpackLayers, testLayers)
			assert.Equal(info.Lifecycle, expectedTestLifecycle)
			assert.Equal(info.CreatedBy, testCreatorData)
		})

		it("sorts buildPacks by ID then Version", func() {
			metadata := builder.Metadata{
				Description: testBuilderDescription,
				Buildpacks: []dist.ModuleInfo{
					testNestedBuildpack,
					testBuildpackVersionTwo,
					testBuildpack,
				},
			}

			labelManager := newLabelManager(returnForMetadata(metadata))
			inspector := builder.NewInspector(
				newDefaultInspectableFetcher(),
				newLabelManagerFactory(labelManager),
				newDefaultDetectionCalculator(),
			)

			info, err := inspector.Inspect(testBuilderName, true, pubbldr.OrderDetectionNone)

			assert.Nil(err)
			assert.Equal(info.Buildpacks, []dist.ModuleInfo{testBuildpack, testBuildpackVersionTwo, testNestedBuildpack})
		})

		when("there are duplicated buildpacks in metadata", func() {
			it("returns deduplicated buildpacks", func() {
				metadata := builder.Metadata{
					Description: testBuilderDescription,
					Buildpacks: []dist.ModuleInfo{
						testTopNestedBuildpack,
						testNestedBuildpack,
						testTopNestedBuildpack,
					},
				}
				labelManager := newLabelManager(returnForMetadata(metadata))

				inspector := builder.NewInspector(
					newDefaultInspectableFetcher(),
					newLabelManagerFactory(labelManager),
					newDefaultDetectionCalculator(),
				)
				info, err := inspector.Inspect(testBuilderName, true, pubbldr.OrderDetectionNone)

				assert.Nil(err)
				assert.Equal(info.Buildpacks, []dist.ModuleInfo{testNestedBuildpack, testTopNestedBuildpack})
			})
		})

		when("label manager returns an error for `Metadata`", func() {
			it("returns the wrapped error", func() {
				expectedBaseError := errors.New("failed to parse")

				labelManager := newLabelManager(errorForMetadata(expectedBaseError))

				inspector := builder.NewInspector(
					newDefaultInspectableFetcher(),
					newLabelManagerFactory(labelManager),
					newDefaultDetectionCalculator(),
				)
				_, err := inspector.Inspect(testBuilderName, true, pubbldr.OrderDetectionNone)

				assert.ErrorWithMessage(err, "reading image metadata: failed to parse")
			})
		})

		when("label manager does not return an error for `StackID`", func() {
			it("returns the wrapped error", func() {
				expectedBaseError := errors.New("label not found")

				labelManager := newLabelManager(errorForStackID(expectedBaseError))

				inspector := builder.NewInspector(
					newDefaultInspectableFetcher(),
					newLabelManagerFactory(labelManager),
					newDefaultDetectionCalculator(),
				)
				_, err := inspector.Inspect(testBuilderName, true, pubbldr.OrderDetectionNone)

				assert.Nil(err)
			})
		})

		when("label manager returns an error for `Mixins`", func() {
			it("returns the wrapped error", func() {
				expectedBaseError := errors.New("label not found")

				labelManager := newLabelManager(errorForMixins(expectedBaseError))

				inspector := builder.NewInspector(
					newDefaultInspectableFetcher(),
					newLabelManagerFactory(labelManager),
					newDefaultDetectionCalculator(),
				)
				_, err := inspector.Inspect(testBuilderName, true, pubbldr.OrderDetectionNone)

				assert.ErrorWithMessage(err, "reading image mixins: label not found")
			})
		})

		when("label manager returns an error for `Order`", func() {
			it("returns the wrapped error", func() {
				expectedBaseError := errors.New("label not found")

				labelManager := newLabelManager(errorForOrder(expectedBaseError))

				inspector := builder.NewInspector(
					newDefaultInspectableFetcher(),
					newLabelManagerFactory(labelManager),
					newDefaultDetectionCalculator(),
				)
				_, err := inspector.Inspect(testBuilderName, true, pubbldr.OrderDetectionNone)

				assert.ErrorWithMessage(err, "reading image order: label not found")
			})
		})

		when("label manager returns an error for `OrderExtensions`", func() {
			it("returns the wrapped error", func() {
				expectedBaseError := errors.New("label not found")

				labelManager := newLabelManager(errorForOrderExtensions(expectedBaseError))

				inspector := builder.NewInspector(
					newDefaultInspectableFetcher(),
					newLabelManagerFactory(labelManager),
					newDefaultDetectionCalculator(),
				)
				_, err := inspector.Inspect(testBuilderName, true, pubbldr.OrderDetectionNone)

				assert.ErrorWithMessage(err, "reading image order extensions: label not found")
			})
		})

		when("label manager returns an error for `ModuleLayers`", func() {
			it("returns the wrapped error", func() {
				expectedBaseError := errors.New("label not found")

				labelManager := newLabelManager(errorForBuildpackLayers(expectedBaseError))

				inspector := builder.NewInspector(
					newDefaultInspectableFetcher(),
					newLabelManagerFactory(labelManager),
					newDefaultDetectionCalculator(),
				)
				_, err := inspector.Inspect(testBuilderName, true, pubbldr.OrderDetectionNone)

				assert.ErrorWithMessage(err, "reading image buildpack layers: label not found")
			})
		})

		when("detection calculator returns an error for `Order`", func() {
			it("returns the wrapped error", func() {
				expectedBaseError := errors.New("couldn't read label")

				inspector := builder.NewInspector(
					newDefaultInspectableFetcher(),
					newDefaultLabelManagerFactory(),
					newDetectionCalculator(errorForDetectionOrder(expectedBaseError)),
				)
				_, err := inspector.Inspect(testBuilderName, true, pubbldr.OrderDetectionMaxDepth)

				assert.ErrorWithMessage(err, "calculating detection order: couldn't read label")
			})
		})
	})
}

func newDefaultInspectableFetcher() *fakes.FakeInspectableFetcher {
	return &fakes.FakeInspectableFetcher{
		InspectableToReturn: newNoOpInspectable(),
	}
}

func newNoOpInspectable() *fakes.FakeInspectable {
	return &fakes.FakeInspectable{}
}

func newDefaultLabelManagerFactory() *fakes.FakeLabelManagerFactory {
	return newLabelManagerFactory(newDefaultLabelManager())
}

func newLabelManagerFactory(manager builder.LabelInspector) *fakes.FakeLabelManagerFactory {
	return fakes.NewFakeLabelManagerFactory(manager)
}

func newDefaultLabelManager() *fakes.FakeLabelManager {
	return &fakes.FakeLabelManager{
		ReturnForMetadata:        testMetadata,
		ReturnForStackID:         testStackID,
		ReturnForMixins:          testMixins,
		ReturnForOrder:           testOrder,
		ReturnForBuildpackLayers: testLayers,
		ReturnForOrderExtensions: testOrderExtensions,
	}
}

type labelManagerModifier func(manager *fakes.FakeLabelManager)

func returnForMetadata(metadata builder.Metadata) labelManagerModifier {
	return func(manager *fakes.FakeLabelManager) {
		manager.ReturnForMetadata = metadata
	}
}

func errorForMetadata(err error) labelManagerModifier {
	return func(manager *fakes.FakeLabelManager) {
		manager.ErrorForMetadata = err
	}
}

func errorForStackID(err error) labelManagerModifier {
	return func(manager *fakes.FakeLabelManager) {
		manager.ErrorForStackID = err
	}
}

func errorForMixins(err error) labelManagerModifier {
	return func(manager *fakes.FakeLabelManager) {
		manager.ErrorForMixins = err
	}
}

func errorForOrder(err error) labelManagerModifier {
	return func(manager *fakes.FakeLabelManager) {
		manager.ErrorForOrder = err
	}
}

func errorForOrderExtensions(err error) labelManagerModifier {
	return func(manager *fakes.FakeLabelManager) {
		manager.ErrorForOrderExtensions = err
	}
}

func errorForBuildpackLayers(err error) labelManagerModifier {
	return func(manager *fakes.FakeLabelManager) {
		manager.ErrorForBuildpackLayers = err
	}
}

func newLabelManager(modifiers ...labelManagerModifier) *fakes.FakeLabelManager {
	manager := newDefaultLabelManager()

	for _, mod := range modifiers {
		mod(manager)
	}

	return manager
}

func newDefaultDetectionCalculator() *fakes.FakeDetectionCalculator {
	return &fakes.FakeDetectionCalculator{
		ReturnForOrder: expectedDetectionTestOrder,
	}
}

type detectionCalculatorModifier func(calculator *fakes.FakeDetectionCalculator)

func errorForDetectionOrder(err error) detectionCalculatorModifier {
	return func(calculator *fakes.FakeDetectionCalculator) {
		calculator.ErrorForOrder = err
	}
}

func newDetectionCalculator(modifiers ...detectionCalculatorModifier) *fakes.FakeDetectionCalculator {
	calculator := newDefaultDetectionCalculator()

	for _, mod := range modifiers {
		mod(calculator)
	}

	return calculator
}
