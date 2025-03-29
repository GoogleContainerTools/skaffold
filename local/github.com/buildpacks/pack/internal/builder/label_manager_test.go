package builder_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/lifecycle/api"

	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/internal/builder/fakes"
	"github.com/buildpacks/pack/pkg/dist"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestLabelManager(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "testLabelManager", testLabelManager, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testLabelManager(t *testing.T, when spec.G, it spec.S) {
	var assert = h.NewAssertionManager(t)

	when("Metadata", func() {
		const (
			buildpackFormat = `{
      "id": "%s",
      "version": "%s",
      "homepage": "%s"
    }`
			lifecycleFormat = `{
      "version": "%s",
      "api": {
        "buildpack": "%s",
        "platform": "%s"
      },
      "apis": {
        "buildpack": {"deprecated": ["%s"], "supported": ["%s", "%s"]},
        "platform": {"deprecated": ["%s"], "supported": ["%s", "%s"]}
      }
    }`
			metadataFormat = `{
  "description": "%s",
  "stack": {
    "runImage": {
      "image": "%s",
      "mirrors": ["%s"]
    }
  },
  "buildpacks": [
    %s,
    %s
  ],
  "lifecycle": %s,
  "createdBy": {"name": "%s", "version": "%s"}
}`
		)

		var (
			expectedDescription    = "Test image description"
			expectedRunImage       = "some/run-image"
			expectedRunImageMirror = "gcr.io/some/default"
			expectedBuildpacks     = []dist.ModuleInfo{
				{
					ID:       "test.buildpack",
					Version:  "test.buildpack.version",
					Homepage: "http://geocities.com/test-bp",
				},
				{
					ID:       "test.buildpack.two",
					Version:  "test.buildpack.two.version",
					Homepage: "http://geocities.com/test-bp-two",
				},
			}
			expectedLifecycleVersion    = builder.VersionMustParse("1.2.3")
			expectedBuildpackAPI        = api.MustParse("0.1")
			expectedPlatformAPI         = api.MustParse("2.3")
			expectedBuildpackDeprecated = "0.1"
			expectedBuildpackSupported  = []string{"1.2", "1.3"}
			expectedPlatformDeprecated  = "1.2"
			expectedPlatformSupported   = []string{"2.3", "2.4"}
			expectedCreatorName         = "pack"
			expectedVersion             = "2.3.4"

			rawMetadata = fmt.Sprintf(
				metadataFormat,
				expectedDescription,
				expectedRunImage,
				expectedRunImageMirror,
				fmt.Sprintf(
					buildpackFormat,
					expectedBuildpacks[0].ID,
					expectedBuildpacks[0].Version,
					expectedBuildpacks[0].Homepage,
				),
				fmt.Sprintf(
					buildpackFormat,
					expectedBuildpacks[1].ID,
					expectedBuildpacks[1].Version,
					expectedBuildpacks[1].Homepage,
				),
				fmt.Sprintf(
					lifecycleFormat,
					expectedLifecycleVersion,
					expectedBuildpackAPI,
					expectedPlatformAPI,
					expectedBuildpackDeprecated,
					expectedBuildpackSupported[0],
					expectedBuildpackSupported[1],
					expectedPlatformDeprecated,
					expectedPlatformSupported[0],
					expectedPlatformSupported[1],
				),
				expectedCreatorName,
				expectedVersion,
			)
		)

		it("returns full metadata", func() {
			inspectable := newInspectable(returnForLabel(rawMetadata))

			labelManager := builder.NewLabelManager(inspectable)
			metadata, err := labelManager.Metadata()
			assert.Nil(err)
			assert.Equal(metadata.Description, expectedDescription)
			assert.Equal(metadata.Stack.RunImage.Image, expectedRunImage)
			assert.Equal(metadata.Stack.RunImage.Mirrors, []string{expectedRunImageMirror})
			assert.Equal(metadata.Buildpacks, expectedBuildpacks)
			assert.Equal(metadata.Lifecycle.Version, expectedLifecycleVersion)
			assert.Equal(metadata.Lifecycle.API.BuildpackVersion, expectedBuildpackAPI)
			assert.Equal(metadata.Lifecycle.API.PlatformVersion, expectedPlatformAPI)
			assert.Equal(metadata.Lifecycle.APIs.Buildpack.Deprecated.AsStrings(), []string{expectedBuildpackDeprecated})
			assert.Equal(metadata.Lifecycle.APIs.Buildpack.Supported.AsStrings(), expectedBuildpackSupported)
			assert.Equal(metadata.Lifecycle.APIs.Platform.Deprecated.AsStrings(), []string{expectedPlatformDeprecated})
			assert.Equal(metadata.Lifecycle.APIs.Platform.Supported.AsStrings(), expectedPlatformSupported)
			assert.Equal(metadata.CreatedBy.Name, expectedCreatorName)
			assert.Equal(metadata.CreatedBy.Version, expectedVersion)
		})

		it("requests the expected label", func() {
			inspectable := newInspectable(returnForLabel(rawMetadata))

			labelManager := builder.NewLabelManager(inspectable)
			_, err := labelManager.Metadata()
			assert.Nil(err)

			assert.Equal(inspectable.ReceivedName, "io.buildpacks.builder.metadata")
		})

		when("inspectable returns an error for `Label`", func() {
			it("returns a wrapped error", func() {
				expectedError := errors.New("couldn't find label")

				inspectable := newInspectable(errorForLabel(expectedError))

				labelManager := builder.NewLabelManager(inspectable)
				_, err := labelManager.Metadata()

				assert.ErrorWithMessage(
					err,
					"getting label io.buildpacks.builder.metadata: couldn't find label",
				)
			})
		})

		when("inspectable returns invalid json for `Label`", func() {
			it("returns a wrapped error", func() {
				inspectable := newInspectable(returnForLabel("{"))

				labelManager := builder.NewLabelManager(inspectable)
				_, err := labelManager.Metadata()

				assert.ErrorWithMessage(
					err,
					"parsing label content for io.buildpacks.builder.metadata: unexpected end of JSON input",
				)
			})
		})

		when("inspectable returns empty content for `Label`", func() {
			it("returns an error suggesting rebuilding the builder", func() {
				inspectable := newInspectable(returnForLabel(""))

				labelManager := builder.NewLabelManager(inspectable)
				_, err := labelManager.Metadata()

				assert.ErrorWithMessage(
					err,
					"builder missing label io.buildpacks.builder.metadata -- try recreating builder",
				)
			})
		})
	})

	when("StackID", func() {
		it("returns the stack ID", func() {
			inspectable := newInspectable(returnForLabel("some.stack.id"))

			labelManager := builder.NewLabelManager(inspectable)
			stackID, err := labelManager.StackID()
			assert.Nil(err)

			assert.Equal(stackID, "some.stack.id")
		})

		it("requests the expected label", func() {
			inspectable := newInspectable(returnForLabel("some.stack.id"))

			labelManager := builder.NewLabelManager(inspectable)
			_, err := labelManager.StackID()
			assert.Nil(err)

			assert.Equal(inspectable.ReceivedName, "io.buildpacks.stack.id")
		})

		when("inspectable return empty content for `Label`", func() {
			it("returns an error suggesting rebuilding the builder", func() {
				inspectable := newInspectable(returnForLabel(""))

				labelManager := builder.NewLabelManager(inspectable)
				_, err := labelManager.StackID()

				assert.ErrorWithMessage(
					err,
					"builder missing label io.buildpacks.stack.id -- try recreating builder",
				)
			})
		})

		when("inspectable returns an error for `Label`", func() {
			it("returns a wrapped error", func() {
				expectedError := errors.New("couldn't find label")

				inspectable := newInspectable(errorForLabel(expectedError))

				labelManager := builder.NewLabelManager(inspectable)
				_, err := labelManager.StackID()

				assert.ErrorWithMessage(
					err,
					"getting label io.buildpacks.stack.id: couldn't find label",
				)
			})
		})
	})

	when("Mixins", func() {
		it("returns the mixins", func() {
			inspectable := newInspectable(returnForLabel(`["mixinX", "mixinY", "build:mixinA"]`))

			labelManager := builder.NewLabelManager(inspectable)
			mixins, err := labelManager.Mixins()
			assert.Nil(err)

			assert.Equal(mixins, []string{"mixinX", "mixinY", "build:mixinA"})
		})

		it("requests the expected label", func() {
			inspectable := newInspectable(returnForLabel(`["mixinX", "mixinY", "build:mixinA"]`))

			labelManager := builder.NewLabelManager(inspectable)
			_, err := labelManager.Mixins()
			assert.Nil(err)

			assert.Equal(inspectable.ReceivedName, "io.buildpacks.stack.mixins")
		})

		when("inspectable return empty content for `Label`", func() {
			it("returns empty stack mixins", func() {
				inspectable := newInspectable(returnForLabel(""))

				labelManager := builder.NewLabelManager(inspectable)
				mixins, err := labelManager.Mixins()
				assert.Nil(err)

				assert.Equal(mixins, []string{})
			})
		})

		when("inspectable returns an error for `Label`", func() {
			it("returns a wrapped error", func() {
				expectedError := errors.New("couldn't find label")

				inspectable := newInspectable(errorForLabel(expectedError))

				labelManager := builder.NewLabelManager(inspectable)
				_, err := labelManager.Mixins()

				assert.ErrorWithMessage(
					err,
					"getting label io.buildpacks.stack.mixins: couldn't find label",
				)
			})
		})

		when("inspectable returns invalid json for `Label`", func() {
			it("returns a wrapped error", func() {
				inspectable := newInspectable(returnForLabel("{"))

				labelManager := builder.NewLabelManager(inspectable)
				_, err := labelManager.Mixins()

				assert.ErrorWithMessage(
					err,
					"parsing label content for io.buildpacks.stack.mixins: unexpected end of JSON input",
				)
			})
		})
	})

	when("Order", func() {
		var rawOrder = `[{"group": [{"id": "buildpack-1-id", "optional": false}, {"id": "buildpack-2-id", "version": "buildpack-2-version-1", "optional": true}]}]`

		it("returns the order", func() {
			inspectable := newInspectable(returnForLabel(rawOrder))

			labelManager := builder.NewLabelManager(inspectable)
			mixins, err := labelManager.Order()
			assert.Nil(err)

			expectedOrder := dist.Order{
				{
					Group: []dist.ModuleRef{
						{
							ModuleInfo: dist.ModuleInfo{
								ID: "buildpack-1-id",
							},
						},
						{
							ModuleInfo: dist.ModuleInfo{
								ID:      "buildpack-2-id",
								Version: "buildpack-2-version-1",
							},
							Optional: true,
						},
					},
				},
			}

			assert.Equal(mixins, expectedOrder)
		})

		it("requests the expected label", func() {
			inspectable := newInspectable(returnForLabel(rawOrder))

			labelManager := builder.NewLabelManager(inspectable)
			_, err := labelManager.Order()
			assert.Nil(err)

			assert.Equal(inspectable.ReceivedName, "io.buildpacks.buildpack.order")
		})

		when("inspectable return empty content for `Label`", func() {
			it("returns an empty order object", func() {
				inspectable := newInspectable(returnForLabel(""))

				labelManager := builder.NewLabelManager(inspectable)
				order, err := labelManager.Order()
				assert.Nil(err)

				assert.Equal(order, dist.Order{})
			})
		})

		when("inspectable returns an error for `Label`", func() {
			it("returns a wrapped error", func() {
				expectedError := errors.New("couldn't find label")

				inspectable := newInspectable(errorForLabel(expectedError))

				labelManager := builder.NewLabelManager(inspectable)
				_, err := labelManager.Order()

				assert.ErrorWithMessage(
					err,
					"getting label io.buildpacks.buildpack.order: couldn't find label",
				)
			})
		})

		when("inspectable returns invalid json for `Label`", func() {
			it("returns a wrapped error", func() {
				inspectable := newInspectable(returnForLabel("{"))

				labelManager := builder.NewLabelManager(inspectable)
				_, err := labelManager.Order()

				assert.ErrorWithMessage(
					err,
					"parsing label content for io.buildpacks.buildpack.order: unexpected end of JSON input",
				)
			})
		})
	})

	when("OrderExtensions", func() {
		var rawOrder = `[{"group": [{"id": "buildpack-1-id", "optional": false}, {"id": "buildpack-2-id", "version": "buildpack-2-version-1", "optional": true}]}]`

		it("returns the order", func() {
			inspectable := newInspectable(returnForLabel(rawOrder))

			labelManager := builder.NewLabelManager(inspectable)
			mixins, err := labelManager.OrderExtensions()
			assert.Nil(err)

			expectedOrder := dist.Order{
				{
					Group: []dist.ModuleRef{
						{
							ModuleInfo: dist.ModuleInfo{
								ID: "buildpack-1-id",
							},
						},
						{
							ModuleInfo: dist.ModuleInfo{
								ID:      "buildpack-2-id",
								Version: "buildpack-2-version-1",
							},
							Optional: true,
						},
					},
				},
			}

			assert.Equal(mixins, expectedOrder)
		})

		it("requests the expected label", func() {
			inspectable := newInspectable(returnForLabel(rawOrder))

			labelManager := builder.NewLabelManager(inspectable)
			_, err := labelManager.OrderExtensions()
			assert.Nil(err)

			assert.Equal(inspectable.ReceivedName, "io.buildpacks.buildpack.order-extensions")
		})

		when("inspectable return empty content for `Label`", func() {
			it("returns an empty order object", func() {
				inspectable := newInspectable(returnForLabel(""))

				labelManager := builder.NewLabelManager(inspectable)
				order, err := labelManager.OrderExtensions()
				assert.Nil(err)

				assert.Equal(order, dist.Order{})
			})
		})

		when("inspectable returns an error for `Label`", func() {
			it("returns a wrapped error", func() {
				expectedError := errors.New("couldn't find label")

				inspectable := newInspectable(errorForLabel(expectedError))

				labelManager := builder.NewLabelManager(inspectable)
				_, err := labelManager.OrderExtensions()

				assert.ErrorWithMessage(
					err,
					"getting label io.buildpacks.buildpack.order-extensions: couldn't find label",
				)
			})
		})

		when("inspectable returns invalid json for `Label`", func() {
			it("returns a wrapped error", func() {
				inspectable := newInspectable(returnForLabel("{"))

				labelManager := builder.NewLabelManager(inspectable)
				_, err := labelManager.OrderExtensions()

				assert.ErrorWithMessage(
					err,
					"parsing label content for io.buildpacks.buildpack.order-extensions: unexpected end of JSON input",
				)
			})
		})
	})

	when("ModuleLayers", func() {
		var rawLayers = `
{
  "buildpack-1-id": {
    "buildpack-1-version-1": {
      "api": "0.1",
      "layerDiffID": "sha256:buildpack-1-version-1-diff-id"
    },
    "buildpack-1-version-2": {
      "api": "0.2",
      "layerDiffID": "sha256:buildpack-1-version-2-diff-id"
    }
  }
}
`

		it("returns the layers", func() {
			inspectable := newInspectable(returnForLabel(rawLayers))

			labelManager := builder.NewLabelManager(inspectable)
			layers, err := labelManager.BuildpackLayers()
			assert.Nil(err)

			expectedLayers := dist.ModuleLayers{
				"buildpack-1-id": {
					"buildpack-1-version-1": dist.ModuleLayerInfo{
						API:         api.MustParse("0.1"),
						LayerDiffID: "sha256:buildpack-1-version-1-diff-id",
					},
					"buildpack-1-version-2": dist.ModuleLayerInfo{
						API:         api.MustParse("0.2"),
						LayerDiffID: "sha256:buildpack-1-version-2-diff-id",
					},
				},
			}

			assert.Equal(layers, expectedLayers)
		})

		it("requests the expected label", func() {
			inspectable := newInspectable(returnForLabel(rawLayers))

			labelManager := builder.NewLabelManager(inspectable)
			_, err := labelManager.BuildpackLayers()
			assert.Nil(err)

			assert.Equal(inspectable.ReceivedName, "io.buildpacks.buildpack.layers")
		})

		when("inspectable return empty content for `Label`", func() {
			it("returns an empty buildpack layers object", func() {
				inspectable := newInspectable(returnForLabel(""))

				labelManager := builder.NewLabelManager(inspectable)
				layers, err := labelManager.BuildpackLayers()
				assert.Nil(err)

				assert.Equal(layers, dist.ModuleLayers{})
			})
		})

		when("inspectable returns an error for `Label`", func() {
			it("returns a wrapped error", func() {
				expectedError := errors.New("couldn't find label")

				inspectable := newInspectable(errorForLabel(expectedError))

				labelManager := builder.NewLabelManager(inspectable)
				_, err := labelManager.BuildpackLayers()

				assert.ErrorWithMessage(
					err,
					"getting label io.buildpacks.buildpack.layers: couldn't find label",
				)
			})
		})

		when("inspectable returns invalid json for `Label`", func() {
			it("returns a wrapped error", func() {
				inspectable := newInspectable(returnForLabel("{"))

				labelManager := builder.NewLabelManager(inspectable)
				_, err := labelManager.BuildpackLayers()

				assert.ErrorWithMessage(
					err,
					"parsing label content for io.buildpacks.buildpack.layers: unexpected end of JSON input",
				)
			})
		})
	})
}

type inspectableModifier func(i *fakes.FakeInspectable)

func returnForLabel(response string) inspectableModifier {
	return func(i *fakes.FakeInspectable) {
		i.ReturnForLabel = response
	}
}

func errorForLabel(err error) inspectableModifier {
	return func(i *fakes.FakeInspectable) {
		i.ErrorForLabel = err
	}
}

func newInspectable(modifiers ...inspectableModifier) *fakes.FakeInspectable {
	inspectable := &fakes.FakeInspectable{}

	for _, mod := range modifiers {
		mod(inspectable)
	}

	return inspectable
}
