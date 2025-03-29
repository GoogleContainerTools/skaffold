package client

import (
	"bytes"
	"fmt"
	"testing"

	pubbldr "github.com/buildpacks/pack/builder"

	"github.com/buildpacks/imgutil/fakes"
	"github.com/buildpacks/lifecycle/api"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/heroku/color"
	"github.com/pkg/errors"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/image"
	"github.com/buildpacks/pack/pkg/logging"
	"github.com/buildpacks/pack/pkg/testmocks"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestInspectBuilder(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "InspectBuilder", testInspectBuilder, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testInspectBuilder(t *testing.T, when spec.G, it spec.S) {
	var (
		subject          *Client
		mockImageFetcher *testmocks.MockImageFetcher
		mockController   *gomock.Controller
		builderImage     *fakes.Image
		out              bytes.Buffer
		assert           = h.NewAssertionManager(t)
	)

	it.Before(func() {
		mockController = gomock.NewController(t)
		mockImageFetcher = testmocks.NewMockImageFetcher(mockController)

		subject = &Client{
			logger:       logging.NewLogWithWriters(&out, &out),
			imageFetcher: mockImageFetcher,
		}

		builderImage = fakes.NewImage("some/builder", "", nil)
		assert.Succeeds(builderImage.SetLabel("io.buildpacks.stack.id", "test.stack.id"))
		assert.Succeeds(builderImage.SetLabel(
			"io.buildpacks.stack.mixins",
			`["mixinOne", "build:mixinTwo", "mixinThree", "build:mixinFour"]`,
		))
		assert.Succeeds(builderImage.SetEnv("CNB_USER_ID", "1234"))
		assert.Succeeds(builderImage.SetEnv("CNB_GROUP_ID", "4321"))
	})

	it.After(func() {
		mockController.Finish()
	})

	when("the image exists", func() {
		for _, useDaemon := range []bool{true, false} {
			useDaemon := useDaemon
			when(fmt.Sprintf("daemon is %t", useDaemon), func() {
				it.Before(func() {
					if useDaemon {
						mockImageFetcher.EXPECT().Fetch(gomock.Any(), "some/builder", image.FetchOptions{Daemon: true, PullPolicy: image.PullNever}).Return(builderImage, nil)
					} else {
						mockImageFetcher.EXPECT().Fetch(gomock.Any(), "some/builder", image.FetchOptions{Daemon: false, PullPolicy: image.PullNever}).Return(builderImage, nil)
					}
				})

				when("only deprecated lifecycle apis are present", func() {
					it.Before(func() {
						assert.Succeeds(builderImage.SetLabel(
							"io.buildpacks.builder.metadata",
							`{"lifecycle": {"version": "1.2.3", "api": {"buildpack": "1.2","platform": "2.3"}}}`,
						))
					})

					it("returns has both deprecated and new fields", func() {
						builderInfo, err := subject.InspectBuilder("some/builder", useDaemon)
						assert.Nil(err)

						assert.Equal(builderInfo.Lifecycle, builder.LifecycleDescriptor{
							Info: builder.LifecycleInfo{
								Version: builder.VersionMustParse("1.2.3"),
							},
							API: builder.LifecycleAPI{
								BuildpackVersion: api.MustParse("1.2"),
								PlatformVersion:  api.MustParse("2.3"),
							},
							APIs: builder.LifecycleAPIs{
								Buildpack: builder.APIVersions{Supported: builder.APISet{api.MustParse("1.2")}},
								Platform:  builder.APIVersions{Supported: builder.APISet{api.MustParse("2.3")}},
							},
						})
					})
				})

				when("the builder image has appropriate metadata labels", func() {
					it.Before(func() {
						assert.Succeeds(builderImage.SetLabel("io.buildpacks.builder.metadata", `{
  "description": "Some description",
  "stack": {
    "runImage": {
      "image": "some/run-image",
      "mirrors": [
        "gcr.io/some/default"
      ]
    }
  },
  "buildpacks": [
    {
      "id": "test.nested",
	  "version": "test.nested.version",
	  "homepage": "http://geocities.com/top-bp"
	},
	{
      "id": "test.bp.one",
	  "version": "test.bp.one.version",
	  "homepage": "http://geocities.com/cool-bp",
	  "name": "one"
    },
	{
      "id": "test.bp.two",
	  "version": "test.bp.two.version"
    },
	{
      "id": "test.bp.two",
	  "version": "test.bp.two.version"
    }
  ],
  "lifecycle": {"version": "1.2.3", "api": {"buildpack": "0.1","platform": "2.3"}, "apis":  {
	"buildpack": {"deprecated": ["0.1"], "supported": ["1.2", "1.3"]},
	"platform": {"deprecated": [], "supported": ["2.3", "2.4"]}
  }},
  "createdBy": {"name": "pack", "version": "1.2.3"},
  "images": [
    {
      "image": "some/run-image",
      "mirrors": [
        "gcr.io/some/default"
      ]
    }
  ]
}`))

						assert.Succeeds(builderImage.SetLabel(
							"io.buildpacks.buildpack.order",
							`[
	{
	  "group": 
		[
		  {
			"id": "test.nested",
			"version": "test.nested.version",
			"optional": false
		  },
		  {
			"id": "test.bp.two",
			"optional": true
		  }
		]
	}
]`,
						))

						assert.Succeeds(builderImage.SetLabel(
							"io.buildpacks.buildpack.layers",
							`{
  "test.nested": {
    "test.nested.version": {
      "api": "0.2",
      "order": [
        {
          "group": [
            {
              "id": "test.bp.one",
              "version": "test.bp.one.version"
            },
            {
              "id": "test.bp.two",
              "version": "test.bp.two.version"
            }
          ]
        }
      ],
      "layerDiffID": "sha256:test.nested.sha256",
	  "homepage": "http://geocities.com/top-bp"
    }
  },
  "test.bp.one": {
    "test.bp.one.version": {
      "api": "0.2",
      "stacks": [
        {
          "id": "test.stack.id"
        }
      ],
      "layerDiffID": "sha256:test.bp.one.sha256",
	  "homepage": "http://geocities.com/cool-bp",
	  "name": "one"
    }
  },
 "test.bp.two": {
    "test.bp.two.version": {
      "api": "0.2",
      "stacks": [
        {
          "id": "test.stack.id"
        }
      ],
      "layerDiffID": "sha256:test.bp.two.sha256"
    }
  }
}`))
					})

					it("returns the builder with the given name with information from the label", func() {
						builderInfo, err := subject.InspectBuilder("some/builder", useDaemon)
						assert.Nil(err)
						apiVersion, err := api.NewVersion("0.2")
						assert.Nil(err)

						want := BuilderInfo{
							Description: "Some description",
							Stack:       "test.stack.id",
							Mixins:      []string{"mixinOne", "mixinThree", "build:mixinTwo", "build:mixinFour"},
							RunImages:   []pubbldr.RunImageConfig{{Image: "some/run-image", Mirrors: []string{"gcr.io/some/default"}}},
							Buildpacks: []dist.ModuleInfo{
								{
									ID:       "test.bp.one",
									Version:  "test.bp.one.version",
									Name:     "one",
									Homepage: "http://geocities.com/cool-bp",
								},
								{
									ID:      "test.bp.two",
									Version: "test.bp.two.version",
								},
								{
									ID:       "test.nested",
									Version:  "test.nested.version",
									Homepage: "http://geocities.com/top-bp",
								},
							},
							Order: pubbldr.DetectionOrder{
								{
									GroupDetectionOrder: pubbldr.DetectionOrder{
										{
											ModuleRef: dist.ModuleRef{
												ModuleInfo: dist.ModuleInfo{ID: "test.nested", Version: "test.nested.version"},
												Optional:   false,
											},
										},
										{
											ModuleRef: dist.ModuleRef{
												ModuleInfo: dist.ModuleInfo{ID: "test.bp.two"},
												Optional:   true,
											},
										},
									},
								},
							},
							BuildpackLayers: map[string]map[string]dist.ModuleLayerInfo{
								"test.nested": {
									"test.nested.version": {
										API: apiVersion,
										Order: dist.Order{
											{
												Group: []dist.ModuleRef{
													{
														ModuleInfo: dist.ModuleInfo{
															ID:      "test.bp.one",
															Version: "test.bp.one.version",
														},
														Optional: false,
													},
													{
														ModuleInfo: dist.ModuleInfo{
															ID:      "test.bp.two",
															Version: "test.bp.two.version",
														},
														Optional: false,
													},
												},
											},
										},
										LayerDiffID: "sha256:test.nested.sha256",
										Homepage:    "http://geocities.com/top-bp",
									},
								},
								"test.bp.one": {
									"test.bp.one.version": {
										API: apiVersion,
										Stacks: []dist.Stack{
											{
												ID: "test.stack.id",
											},
										},
										LayerDiffID: "sha256:test.bp.one.sha256",
										Homepage:    "http://geocities.com/cool-bp",
										Name:        "one",
									},
								},
								"test.bp.two": {
									"test.bp.two.version": {
										API: apiVersion,
										Stacks: []dist.Stack{
											{
												ID: "test.stack.id",
											},
										},
										LayerDiffID: "sha256:test.bp.two.sha256",
									},
								},
							},
							Lifecycle: builder.LifecycleDescriptor{
								Info: builder.LifecycleInfo{
									Version: builder.VersionMustParse("1.2.3"),
								},
								API: builder.LifecycleAPI{
									BuildpackVersion: api.MustParse("0.1"),
									PlatformVersion:  api.MustParse("2.3"),
								},
								APIs: builder.LifecycleAPIs{
									Buildpack: builder.APIVersions{
										Deprecated: builder.APISet{api.MustParse("0.1")},
										Supported:  builder.APISet{api.MustParse("1.2"), api.MustParse("1.3")},
									},
									Platform: builder.APIVersions{
										Deprecated: builder.APISet{},
										Supported:  builder.APISet{api.MustParse("2.3"), api.MustParse("2.4")},
									},
								},
							},
							CreatedBy: builder.CreatorMetadata{
								Name:    "pack",
								Version: "1.2.3",
							},
						}

						if diff := cmp.Diff(want, *builderInfo); diff != "" {
							t.Errorf("InspectBuilder() mismatch (-want +got):\n%s", diff)
						}
					})

					when("order detection depth is higher than None", func() {
						it("shows subgroup order as part of order", func() {
							builderInfo, err := subject.InspectBuilder(
								"some/builder",
								useDaemon,
								WithDetectionOrderDepth(pubbldr.OrderDetectionMaxDepth),
							)
							h.AssertNil(t, err)

							want := pubbldr.DetectionOrder{
								{
									GroupDetectionOrder: pubbldr.DetectionOrder{
										{
											ModuleRef: dist.ModuleRef{
												ModuleInfo: dist.ModuleInfo{ID: "test.nested", Version: "test.nested.version"},
												Optional:   false,
											},
											GroupDetectionOrder: pubbldr.DetectionOrder{
												{
													ModuleRef: dist.ModuleRef{
														ModuleInfo: dist.ModuleInfo{
															ID:      "test.bp.one",
															Version: "test.bp.one.version",
														},
													},
												},
												{
													ModuleRef: dist.ModuleRef{
														ModuleInfo: dist.ModuleInfo{
															ID:      "test.bp.two",
															Version: "test.bp.two.version",
														},
													},
												},
											},
										},
										{
											ModuleRef: dist.ModuleRef{
												ModuleInfo: dist.ModuleInfo{ID: "test.bp.two"},
												Optional:   true,
											},
										},
									},
								},
							}

							if diff := cmp.Diff(want, builderInfo.Order); diff != "" {
								t.Errorf("\"InspectBuilder() mismatch (-want +got):\b%s", diff)
							}
						})
					})
				})

				// TODO add test case when builder is flattened
			})
		}
	})

	when("the image does not exist", func() {
		it.Before(func() {
			notFoundImage := fakes.NewImage("", "", nil)
			notFoundImage.Delete()
			mockImageFetcher.EXPECT().Fetch(gomock.Any(), "some/builder", image.FetchOptions{Daemon: true, PullPolicy: image.PullNever}).Return(nil, errors.Wrap(image.ErrNotFound, "some-error"))
		})

		it("return nil metadata", func() {
			metadata, err := subject.InspectBuilder("some/builder", true)
			assert.Nil(err)
			assert.Nil(metadata)
		})
	})
}
