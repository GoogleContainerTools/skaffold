package commands_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/buildpacks/lifecycle/api"
	"github.com/golang/mock/gomock"
	"github.com/heroku/color"
	"github.com/pkg/errors"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/commands"
	"github.com/buildpacks/pack/internal/commands/testmocks"
	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/pkg/buildpack"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/image"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

const complexOutputSection = `Stacks:
  ID: io.buildpacks.stacks.first-stack
    Mixins:
      (omitted)
  ID: io.buildpacks.stacks.second-stack
    Mixins:
      (omitted)

Buildpacks:
  ID                                 NAME        VERSION        HOMEPAGE
  some/first-inner-buildpack         -           1.0.0          first-inner-buildpack-homepage
  some/second-inner-buildpack        -           2.0.0          second-inner-buildpack-homepage
  some/third-inner-buildpack         -           3.0.0          third-inner-buildpack-homepage
  some/top-buildpack                 top         0.0.1          top-buildpack-homepage

Detection Order:
 └ Group #1:
    └ some/top-buildpack@0.0.1
       ├ Group #1:
       │  ├ some/first-inner-buildpack@1.0.0
       │  │  ├ Group #1:
       │  │  │  ├ some/first-inner-buildpack@1.0.0    [cyclic]
       │  │  │  └ some/third-inner-buildpack@3.0.0
       │  │  └ Group #2:
       │  │     └ some/third-inner-buildpack@3.0.0
       │  └ some/second-inner-buildpack@2.0.0
       └ Group #2:
          └ some/first-inner-buildpack@1.0.0
             ├ Group #1:
             │  ├ some/first-inner-buildpack@1.0.0    [cyclic]
             │  └ some/third-inner-buildpack@3.0.0
             └ Group #2:
                └ some/third-inner-buildpack@3.0.0`

const simpleOutputSection = `Stacks:
  ID: io.buildpacks.stacks.first-stack
    Mixins:
      (omitted)
  ID: io.buildpacks.stacks.second-stack
    Mixins:
      (omitted)

Buildpacks:
  ID                                NAME        VERSION        HOMEPAGE
  some/single-buildpack             some        0.0.1          single-buildpack-homepage
  some/buildpack-no-homepage        -           0.0.2          -

Detection Order:
 └ Group #1:
    └ some/single-buildpack@0.0.1`

const inspectOutputTemplate = `Inspecting buildpack: '%s'

%s

%s
`

const depthOutputSection = `
Detection Order:
 └ Group #1:
    └ some/top-buildpack@0.0.1
       ├ Group #1:
       │  ├ some/first-inner-buildpack@1.0.0
       │  └ some/second-inner-buildpack@2.0.0
       └ Group #2:
          └ some/first-inner-buildpack@1.0.0`

const simpleMixinOutputSection = `
  ID: io.buildpacks.stacks.first-stack
    Mixins:
      mixin1
      mixin2
      build:mixin3
      build:mixin4
  ID: io.buildpacks.stacks.second-stack
    Mixins:
      mixin1
      mixin2`

func TestBuildpackInspectCommand(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "BuildpackInspectCommand", testBuildpackInspectCommand, spec.Sequential(), spec.Report(report.Terminal{}))
}

func testBuildpackInspectCommand(t *testing.T, when spec.G, it spec.S) {
	apiVersion, err := api.NewVersion("0.2")
	if err != nil {
		t.Fail()
	}

	var (
		command        *cobra.Command
		logger         logging.Logger
		outBuf         bytes.Buffer
		mockController *gomock.Controller
		mockClient     *testmocks.MockPackClient
		cfg            config.Config
		complexInfo    *client.BuildpackInfo
		simpleInfo     *client.BuildpackInfo
		assert         = h.NewAssertionManager(t)
	)

	it.Before(func() {
		mockController = gomock.NewController(t)
		mockClient = testmocks.NewMockPackClient(mockController)
		logger = logging.NewLogWithWriters(&outBuf, &outBuf)

		cfg = config.Config{
			DefaultRegistryName: "default-registry",
		}

		complexInfo = &client.BuildpackInfo{
			BuildpackMetadata: buildpack.Metadata{
				ModuleInfo: dist.ModuleInfo{
					ID:       "some/top-buildpack",
					Version:  "0.0.1",
					Homepage: "top-buildpack-homepage",
					Name:     "top",
				},
				Stacks: []dist.Stack{
					{ID: "io.buildpacks.stacks.first-stack", Mixins: []string{"mixin1", "mixin2", "build:mixin3", "build:mixin4"}},
					{ID: "io.buildpacks.stacks.second-stack", Mixins: []string{"mixin1", "mixin2"}},
				},
			},
			Buildpacks: []dist.ModuleInfo{
				{
					ID:       "some/first-inner-buildpack",
					Version:  "1.0.0",
					Homepage: "first-inner-buildpack-homepage",
				},
				{
					ID:       "some/second-inner-buildpack",
					Version:  "2.0.0",
					Homepage: "second-inner-buildpack-homepage",
				},
				{
					ID:       "some/third-inner-buildpack",
					Version:  "3.0.0",
					Homepage: "third-inner-buildpack-homepage",
				},
				{
					ID:       "some/top-buildpack",
					Version:  "0.0.1",
					Homepage: "top-buildpack-homepage",
					Name:     "top",
				},
			},
			Order: dist.Order{
				{
					Group: []dist.ModuleRef{
						{
							ModuleInfo: dist.ModuleInfo{
								ID:       "some/top-buildpack",
								Version:  "0.0.1",
								Homepage: "top-buildpack-homepage",
								Name:     "top",
							},
							Optional: false,
						},
					},
				},
			},
			BuildpackLayers: dist.ModuleLayers{
				"some/first-inner-buildpack": {
					"1.0.0": {
						API: apiVersion,
						Stacks: []dist.Stack{
							{ID: "io.buildpacks.stacks.first-stack", Mixins: []string{"mixin1", "mixin2", "build:mixin3", "build:mixin4"}},
							{ID: "io.buildpacks.stacks.second-stack", Mixins: []string{"mixin1", "mixin2"}},
						},
						Order: dist.Order{
							{
								Group: []dist.ModuleRef{
									{
										ModuleInfo: dist.ModuleInfo{
											ID:      "some/first-inner-buildpack",
											Version: "1.0.0",
										},
										Optional: false,
									},
									{
										ModuleInfo: dist.ModuleInfo{
											ID:      "some/third-inner-buildpack",
											Version: "3.0.0",
										},
										Optional: false,
									},
								},
							},
							{
								Group: []dist.ModuleRef{
									{
										ModuleInfo: dist.ModuleInfo{
											ID:      "some/third-inner-buildpack",
											Version: "3.0.0",
										},
										Optional: false,
									},
								},
							},
						},
						LayerDiffID: "sha256:first-inner-buildpack-diff-id",
						Homepage:    "first-inner-buildpack-homepage",
					},
				},
				"some/second-inner-buildpack": {
					"2.0.0": {
						API: apiVersion,
						Stacks: []dist.Stack{
							{ID: "io.buildpacks.stacks.first-stack", Mixins: []string{"mixin1", "mixin2", "build:mixin3", "build:mixin4"}},
							{ID: "io.buildpacks.stacks.second-stack", Mixins: []string{"mixin1", "mixin2"}},
						},
						LayerDiffID: "sha256:second-inner-buildpack-diff-id",
						Homepage:    "second-inner-buildpack-homepage",
					},
				},
				"some/third-inner-buildpack": {
					"3.0.0": {
						API: apiVersion,
						Stacks: []dist.Stack{
							{ID: "io.buildpacks.stacks.first-stack", Mixins: []string{"mixin1", "mixin2", "build:mixin3", "build:mixin4"}},
							{ID: "io.buildpacks.stacks.second-stack", Mixins: []string{"mixin1", "mixin2"}},
						},
						LayerDiffID: "sha256:third-inner-buildpack-diff-id",
						Homepage:    "third-inner-buildpack-homepage",
					},
				},
				"some/top-buildpack": {
					"0.0.1": {
						API: apiVersion,
						Order: dist.Order{
							{
								Group: []dist.ModuleRef{
									{
										ModuleInfo: dist.ModuleInfo{
											ID:      "some/first-inner-buildpack",
											Version: "1.0.0",
										},
										Optional: false,
									},
									{
										ModuleInfo: dist.ModuleInfo{
											ID:      "some/second-inner-buildpack",
											Version: "2.0.0",
										},
										Optional: false,
									},
								},
							},
							{
								Group: []dist.ModuleRef{
									{
										ModuleInfo: dist.ModuleInfo{
											ID:      "some/first-inner-buildpack",
											Version: "1.0.0",
										},
										Optional: false,
									},
								},
							},
						},
						LayerDiffID: "sha256:top-buildpack-diff-id",
						Homepage:    "top-buildpack-homepage",
						Name:        "top",
					},
				},
			},
		}

		simpleInfo = &client.BuildpackInfo{
			BuildpackMetadata: buildpack.Metadata{
				ModuleInfo: dist.ModuleInfo{
					ID:       "some/single-buildpack",
					Version:  "0.0.1",
					Homepage: "single-homepage-homepace",
					Name:     "some",
				},
				Stacks: []dist.Stack{
					{ID: "io.buildpacks.stacks.first-stack", Mixins: []string{"mixin1", "mixin2", "build:mixin3", "build:mixin4"}},
					{ID: "io.buildpacks.stacks.second-stack", Mixins: []string{"mixin1", "mixin2"}},
				},
			},
			Buildpacks: []dist.ModuleInfo{
				{
					ID:       "some/single-buildpack",
					Version:  "0.0.1",
					Name:     "some",
					Homepage: "single-buildpack-homepage",
				},
				{
					ID:      "some/buildpack-no-homepage",
					Version: "0.0.2",
				},
			},
			Order: dist.Order{
				{
					Group: []dist.ModuleRef{
						{
							ModuleInfo: dist.ModuleInfo{
								ID:       "some/single-buildpack",
								Version:  "0.0.1",
								Homepage: "single-buildpack-homepage",
							},
							Optional: false,
						},
					},
				},
			},
			BuildpackLayers: dist.ModuleLayers{
				"some/single-buildpack": {
					"0.0.1": {
						API: apiVersion,
						Stacks: []dist.Stack{
							{ID: "io.buildpacks.stacks.first-stack", Mixins: []string{"mixin1", "mixin2", "build:mixin3", "build:mixin4"}},
							{ID: "io.buildpacks.stacks.second-stack", Mixins: []string{"mixin1", "mixin2"}},
						},
						LayerDiffID: "sha256:single-buildpack-diff-id",
						Homepage:    "single-buildpack-homepage",
						Name:        "some",
					},
				},
			},
		}

		command = commands.BuildpackInspect(logger, cfg, mockClient)
	})

	when("BuildpackInspect", func() {
		when("inspecting an image", func() {
			when("both remote and local image are present", func() {
				it.Before(func() {
					complexInfo.Location = buildpack.PackageLocator
					simpleInfo.Location = buildpack.PackageLocator

					mockClient.EXPECT().InspectBuildpack(client.InspectBuildpackOptions{
						BuildpackName: "test/buildpack",
						Daemon:        true,
						Registry:      "default-registry",
					}).Return(complexInfo, nil)

					mockClient.EXPECT().InspectBuildpack(client.InspectBuildpackOptions{
						BuildpackName: "test/buildpack",
						Daemon:        false,
						Registry:      "default-registry",
					}).Return(simpleInfo, nil)
				})

				it("succeeds", func() {
					command.SetArgs([]string{"test/buildpack"})
					assert.Nil(command.Execute())

					localOutputSection := fmt.Sprintf(inspectOutputTemplate,
						"test/buildpack",
						"LOCAL IMAGE:",
						complexOutputSection)

					remoteOutputSection := fmt.Sprintf("%s\n\n%s",
						"REMOTE IMAGE:",
						simpleOutputSection)

					assert.AssertTrimmedContains(outBuf.String(), localOutputSection)
					assert.AssertTrimmedContains(outBuf.String(), remoteOutputSection)
				})
			})

			when("only a local image is present", func() {
				it.Before(func() {
					complexInfo.Location = buildpack.PackageLocator
					simpleInfo.Location = buildpack.PackageLocator

					mockClient.EXPECT().InspectBuildpack(client.InspectBuildpackOptions{
						BuildpackName: "only-local-test/buildpack",
						Daemon:        true,
						Registry:      "default-registry",
					}).Return(complexInfo, nil)

					mockClient.EXPECT().InspectBuildpack(client.InspectBuildpackOptions{
						BuildpackName: "only-local-test/buildpack",
						Daemon:        false,
						Registry:      "default-registry",
					}).Return(nil, errors.Wrap(image.ErrNotFound, "remote image not found!"))
				})

				it("displays output for local image", func() {
					command.SetArgs([]string{"only-local-test/buildpack"})
					assert.Nil(command.Execute())

					expectedOutput := fmt.Sprintf(inspectOutputTemplate,
						"only-local-test/buildpack",
						"LOCAL IMAGE:",
						complexOutputSection)

					assert.AssertTrimmedContains(outBuf.String(), expectedOutput)
				})
			})

			when("only a remote image is present", func() {
				it.Before(func() {
					complexInfo.Location = buildpack.PackageLocator
					simpleInfo.Location = buildpack.PackageLocator

					mockClient.EXPECT().InspectBuildpack(client.InspectBuildpackOptions{
						BuildpackName: "only-remote-test/buildpack",
						Daemon:        false,
						Registry:      "default-registry",
					}).Return(complexInfo, nil)

					mockClient.EXPECT().InspectBuildpack(client.InspectBuildpackOptions{
						BuildpackName: "only-remote-test/buildpack",
						Daemon:        true,
						Registry:      "default-registry",
					}).Return(nil, errors.Wrap(image.ErrNotFound, "remote image not found!"))
				})

				it("displays output for remote image", func() {
					command.SetArgs([]string{"only-remote-test/buildpack"})
					assert.Nil(command.Execute())

					expectedOutput := fmt.Sprintf(inspectOutputTemplate,
						"only-remote-test/buildpack",
						"REMOTE IMAGE:",
						complexOutputSection)

					assert.AssertTrimmedContains(outBuf.String(), expectedOutput)
				})
			})
		})

		when("inspecting a buildpack uri", func() {
			it.Before(func() {
				simpleInfo.Location = buildpack.URILocator
			})

			when("uri is a local path", func() {
				it.Before(func() {
					mockClient.EXPECT().InspectBuildpack(client.InspectBuildpackOptions{
						BuildpackName: "/path/to/test/buildpack",
						Daemon:        true,
						Registry:      "default-registry",
					}).Return(simpleInfo, nil)
				})

				it("succeeds", func() {
					command.SetArgs([]string{"/path/to/test/buildpack"})
					assert.Nil(command.Execute())

					expectedOutput := fmt.Sprintf(inspectOutputTemplate,
						"/path/to/test/buildpack",
						"LOCAL ARCHIVE:",
						simpleOutputSection)

					assert.TrimmedEq(outBuf.String(), expectedOutput)
				})
			})

			when("uri is an http or https location", func() {
				it.Before(func() {
					simpleInfo.Location = buildpack.URILocator
				})
				when("uri is a local path", func() {
					it.Before(func() {
						mockClient.EXPECT().InspectBuildpack(client.InspectBuildpackOptions{
							BuildpackName: "https://path/to/test/buildpack",
							Daemon:        true,
							Registry:      "default-registry",
						}).Return(simpleInfo, nil)
					})

					it("succeeds", func() {
						command.SetArgs([]string{"https://path/to/test/buildpack"})
						assert.Nil(command.Execute())

						expectedOutput := fmt.Sprintf(inspectOutputTemplate,
							"https://path/to/test/buildpack",
							"REMOTE ARCHIVE:",
							simpleOutputSection)

						assert.TrimmedEq(outBuf.String(), expectedOutput)
					})
				})
			})
		})

		when("inspecting a buildpack on the registry", func() {
			it.Before(func() {
				simpleInfo.Location = buildpack.RegistryLocator
			})

			when("using the default registry", func() {
				it.Before(func() {
					mockClient.EXPECT().InspectBuildpack(client.InspectBuildpackOptions{
						BuildpackName: "urn:cnb:registry:test/buildpack",
						Daemon:        true,
						Registry:      "default-registry",
					}).Return(simpleInfo, nil)
				})
				it("succeeds", func() {
					command.SetArgs([]string{"urn:cnb:registry:test/buildpack"})
					assert.Nil(command.Execute())

					expectedOutput := fmt.Sprintf(inspectOutputTemplate,
						"urn:cnb:registry:test/buildpack",
						"REGISTRY IMAGE:",
						simpleOutputSection)

					assert.TrimmedEq(outBuf.String(), expectedOutput)
				})
			})

			when("using a user provided registry", func() {
				it.Before(func() {
					mockClient.EXPECT().InspectBuildpack(client.InspectBuildpackOptions{
						BuildpackName: "urn:cnb:registry:test/buildpack",
						Daemon:        true,
						Registry:      "some-registry",
					}).Return(simpleInfo, nil)
				})

				it("succeeds", func() {
					command.SetArgs([]string{"urn:cnb:registry:test/buildpack", "-r", "some-registry"})
					assert.Nil(command.Execute())

					expectedOutput := fmt.Sprintf(inspectOutputTemplate,
						"urn:cnb:registry:test/buildpack",
						"REGISTRY IMAGE:",
						simpleOutputSection)

					assert.TrimmedEq(outBuf.String(), expectedOutput)
				})
			})
		})

		when("a depth flag is passed", func() {
			it.Before(func() {
				complexInfo.Location = buildpack.URILocator

				mockClient.EXPECT().InspectBuildpack(client.InspectBuildpackOptions{
					BuildpackName: "/other/path/to/test/buildpack",
					Daemon:        true,
					Registry:      "default-registry",
				}).Return(complexInfo, nil)
			})

			it("displays detection order to specified depth", func() {
				command.SetArgs([]string{"/other/path/to/test/buildpack", "-d", "2"})
				assert.Nil(command.Execute())

				assert.AssertTrimmedContains(outBuf.String(), depthOutputSection)
			})
		})
	})

	when("verbose flag is passed", func() {
		it.Before(func() {
			simpleInfo.Location = buildpack.URILocator
			mockClient.EXPECT().InspectBuildpack(client.InspectBuildpackOptions{
				BuildpackName: "/another/path/to/test/buildpack",
				Daemon:        true,
				Registry:      "default-registry",
			}).Return(simpleInfo, nil)
		})

		it("displays all mixins", func() {
			command.SetArgs([]string{"/another/path/to/test/buildpack", "-v"})
			assert.Nil(command.Execute())

			assert.AssertTrimmedContains(outBuf.String(), simpleMixinOutputSection)
		})
	})

	when("failure cases", func() {
		when("unable to inspect buildpack image", func() {
			it.Before(func() {
				mockClient.EXPECT().InspectBuildpack(client.InspectBuildpackOptions{
					BuildpackName: "failure-case/buildpack",
					Daemon:        true,
					Registry:      "default-registry",
				}).Return(&client.BuildpackInfo{}, errors.Wrap(image.ErrNotFound, "unable to inspect local failure-case/buildpack"))

				mockClient.EXPECT().InspectBuildpack(client.InspectBuildpackOptions{
					BuildpackName: "failure-case/buildpack",
					Daemon:        false,
					Registry:      "default-registry",
				}).Return(&client.BuildpackInfo{}, errors.Wrap(image.ErrNotFound, "unable to inspect remote failure-case/buildpack"))
			})

			it("errors", func() {
				command.SetArgs([]string{"failure-case/buildpack"})
				err := command.Execute()
				assert.Error(err)
			})
		})
		when("unable to inspect buildpack archive", func() {
			it.Before(func() {
				mockClient.EXPECT().InspectBuildpack(client.InspectBuildpackOptions{
					BuildpackName: "http://path/to/failure-case/buildpack",
					Daemon:        true,
					Registry:      "default-registry",
				}).Return(&client.BuildpackInfo{}, errors.New("error inspecting local archive"))

				it("errors", func() {
					command.SetArgs([]string{"http://path/to/failure-case/buildpack"})
					err := command.Execute()

					assert.Error(err)
					assert.Contains(err.Error(), "error inspecting local archive")
				})
			})
		})
		when("unable to inspect both remote and local images", func() {
			it.Before(func() {
				mockClient.EXPECT().InspectBuildpack(client.InspectBuildpackOptions{
					BuildpackName: "image-failure-case/buildpack",
					Daemon:        true,
					Registry:      "default-registry",
				}).Return(&client.BuildpackInfo{}, errors.Wrap(image.ErrNotFound, "error inspecting local archive"))

				mockClient.EXPECT().InspectBuildpack(client.InspectBuildpackOptions{
					BuildpackName: "image-failure-case/buildpack",
					Daemon:        false,
					Registry:      "default-registry",
				}).Return(&client.BuildpackInfo{}, errors.Wrap(image.ErrNotFound, "error inspecting remote archive"))
			})

			it("errors", func() {
				command.SetArgs([]string{"image-failure-case/buildpack"})
				err := command.Execute()

				assert.Error(err)
				assert.Contains(err.Error(), "error writing buildpack output: \"error inspecting local archive: not found, error inspecting remote archive: not found\"")
			})
		})

		when("unable to inspect buildpack on registry", func() {
			it.Before(func() {
				mockClient.EXPECT().InspectBuildpack(client.InspectBuildpackOptions{
					BuildpackName: "urn:cnb:registry:registry-failure/buildpack",
					Daemon:        true,
					Registry:      "some-registry",
				}).Return(&client.BuildpackInfo{}, errors.New("error inspecting registry image"))

				mockClient.EXPECT().InspectBuildpack(client.InspectBuildpackOptions{
					BuildpackName: "urn:cnb:registry:registry-failure/buildpack",
					Daemon:        false,
					Registry:      "some-registry",
				}).Return(&client.BuildpackInfo{}, errors.New("error inspecting registry image"))
			})

			it("errors", func() {
				command.SetArgs([]string{"urn:cnb:registry:registry-failure/buildpack", "-r", "some-registry"})

				err := command.Execute()
				assert.Error(err)
				assert.Contains(err.Error(), "error inspecting registry image")
			})
		})
	})
}
