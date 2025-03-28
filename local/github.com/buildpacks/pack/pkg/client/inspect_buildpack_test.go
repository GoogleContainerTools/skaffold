package client_test

import (
	"archive/tar"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/buildpacks/imgutil/fakes"
	"github.com/buildpacks/lifecycle/api"
	"github.com/golang/mock/gomock"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/heroku/color"
	"github.com/pkg/errors"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	cfg "github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/pkg/archive"
	"github.com/buildpacks/pack/pkg/blob"
	"github.com/buildpacks/pack/pkg/buildpack"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/image"
	"github.com/buildpacks/pack/pkg/logging"
	"github.com/buildpacks/pack/pkg/testmocks"
	h "github.com/buildpacks/pack/testhelpers"
)

const buildpackageMetadataTag = `{
  "id": "some/top-buildpack",
  "version": "0.0.1",
  "name": "top",
  "homepage": "top-buildpack-homepage",
  "stacks": [
    {
      "id": "io.buildpacks.stacks.first-stack"
    },
    {
      "id": "io.buildpacks.stacks.second-stack"
    }
  ]
}`

const buildpackLayersTag = `{
   "some/first-inner-buildpack":{
      "1.0.0":{
         "api":"0.2",
         "order":[
            {
               "group":[
                  {
                     "id":"some/first-inner-buildpack",
                     "version":"1.0.0"
                  },
                  {
                     "id":"some/second-inner-buildpack",
                     "version":"3.0.0"
                  }
               ]
            },
            {
               "group":[
                  {
                     "id":"some/second-inner-buildpack",
                     "version":"3.0.0"
                  }
               ]
            }
         ],
         "stacks":[
            {
               "id":"io.buildpacks.stacks.first-stack"
            },
            {
               "id":"io.buildpacks.stacks.second-stack"
            }
         ],
         "layerDiffID":"sha256:first-inner-buildpack-diff-id",
         "homepage":"first-inner-buildpack-homepage"
      }
   },
   "some/second-inner-buildpack":{
      "2.0.0":{
         "api":"0.2",
         "stacks":[
            {
               "id":"io.buildpacks.stacks.first-stack"
            },
            {
               "id":"io.buildpacks.stacks.second-stack"
            }
         ],
         "layerDiffID":"sha256:second-inner-buildpack-diff-id",
         "homepage":"second-inner-buildpack-homepage"
      },
      "3.0.0":{
         "api":"0.2",
         "stacks":[
            {
               "id":"io.buildpacks.stacks.first-stack"
            },
            {
               "id":"io.buildpacks.stacks.second-stack"
            }
         ],
         "layerDiffID":"sha256:third-inner-buildpack-diff-id",
         "homepage":"third-inner-buildpack-homepage"
      }
   },
   "some/top-buildpack":{
      "0.0.1":{
         "api":"0.2",
         "order":[
            {
               "group":[
                  {
                     "id":"some/first-inner-buildpack",
                     "version":"1.0.0"
                  },
                  {
                     "id":"some/second-inner-buildpack",
                     "version":"2.0.0"
                  }
               ]
            },
            {
               "group":[
                  {
                     "id":"some/first-inner-buildpack",
                     "version":"1.0.0"
                  }
               ]
            }
         ],
         "layerDiffID":"sha256:top-buildpack-diff-id",
         "homepage":"top-buildpack-homepage",
		 "name": "top"
      }
   }
}`

func TestInspectBuildpack(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "InspectBuilder", testInspectBuildpack, spec.Sequential(), spec.Report(report.Terminal{}))
}

func testInspectBuildpack(t *testing.T, when spec.G, it spec.S) {
	var (
		subject          *client.Client
		mockImageFetcher *testmocks.MockImageFetcher
		mockController   *gomock.Controller
		out              bytes.Buffer
		buildpackImage   *fakes.Image
		apiVersion       *api.Version
		expectedInfo     *client.BuildpackInfo
		mockDownloader   *testmocks.MockBlobDownloader

		tmpDir        string
		buildpackPath string
	)

	it.Before(func() {
		mockController = gomock.NewController(t)
		mockImageFetcher = testmocks.NewMockImageFetcher(mockController)
		mockDownloader = testmocks.NewMockBlobDownloader(mockController)

		subject = &client.Client{}
		client.WithLogger(logging.NewLogWithWriters(&out, &out))(subject)
		client.WithFetcher(mockImageFetcher)(subject)
		client.WithDownloader(mockDownloader)(subject)

		buildpackImage = fakes.NewImage("some/buildpack", "", nil)
		h.AssertNil(t, buildpackImage.SetLabel(buildpack.MetadataLabel, buildpackageMetadataTag))
		h.AssertNil(t, buildpackImage.SetLabel(dist.BuildpackLayersLabel, buildpackLayersTag))

		var err error
		apiVersion, err = api.NewVersion("0.2")
		h.AssertNil(t, err)

		tmpDir, err = os.MkdirTemp("", "inspectBuildpack")
		h.AssertNil(t, err)

		buildpackPath = filepath.Join(tmpDir, "buildpackTarFile.tar")

		expectedInfo = &client.BuildpackInfo{
			BuildpackMetadata: buildpack.Metadata{
				ModuleInfo: dist.ModuleInfo{
					ID:       "some/top-buildpack",
					Version:  "0.0.1",
					Name:     "top",
					Homepage: "top-buildpack-homepage",
				},
				Stacks: []dist.Stack{
					{ID: "io.buildpacks.stacks.first-stack"},
					{ID: "io.buildpacks.stacks.second-stack"},
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
					ID:       "some/second-inner-buildpack",
					Version:  "3.0.0",
					Homepage: "third-inner-buildpack-homepage",
				},
				{
					ID:       "some/top-buildpack",
					Version:  "0.0.1",
					Name:     "top",
					Homepage: "top-buildpack-homepage",
				},
			},
			Order: dist.Order{
				{
					Group: []dist.ModuleRef{
						{
							ModuleInfo: dist.ModuleInfo{
								ID:       "some/top-buildpack",
								Version:  "0.0.1",
								Name:     "top",
								Homepage: "top-buildpack-homepage",
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
							{ID: "io.buildpacks.stacks.first-stack"},
							{ID: "io.buildpacks.stacks.second-stack"},
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
											ID:      "some/second-inner-buildpack",
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
											ID:      "some/second-inner-buildpack",
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
							{ID: "io.buildpacks.stacks.first-stack"},
							{ID: "io.buildpacks.stacks.second-stack"},
						},
						LayerDiffID: "sha256:second-inner-buildpack-diff-id",
						Homepage:    "second-inner-buildpack-homepage",
					},
					"3.0.0": {
						API: apiVersion,
						Stacks: []dist.Stack{
							{ID: "io.buildpacks.stacks.first-stack"},
							{ID: "io.buildpacks.stacks.second-stack"},
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
	})

	it.After(func() {
		mockController.Finish()
		err := os.RemoveAll(tmpDir)
		if runtime.GOOS != "windows" {
			h.AssertNil(t, err)
		}
	})

	when("inspect-buildpack", func() {
		when("inspecting a registry buildpack", func() {
			var registryFixture string
			var configPath string
			it.Before(func() {
				expectedInfo.Location = buildpack.RegistryLocator

				registryFixture = h.CreateRegistryFixture(t, tmpDir, filepath.Join("testdata", "registry"))
				packHome := filepath.Join(tmpDir, "packHome")
				h.AssertNil(t, os.Setenv("PACK_HOME", packHome))

				configPath = filepath.Join(packHome, "config.toml")
				h.AssertNil(t, cfg.Write(cfg.Config{
					Registries: []cfg.Registry{
						{
							Name: "some-registry",
							Type: "github",
							URL:  registryFixture,
						},
					},
				}, configPath))

				mockImageFetcher.EXPECT().Fetch(
					gomock.Any(),
					"example.com/some/package@sha256:8c27fe111c11b722081701dfed3bd55e039b9ce92865473cf4cdfa918071c566",
					image.FetchOptions{Daemon: false, PullPolicy: image.PullNever}).Return(buildpackImage, nil)
			})

			it.After(func() {
				h.AssertNil(t, os.Unsetenv("PACK_HOME"))
			})

			it("succeeds", func() {
				registryBuildpack := "urn:cnb:registry:example/java"
				inspectOptions := client.InspectBuildpackOptions{
					BuildpackName: registryBuildpack,
					Registry:      "some-registry",
				}
				info, err := subject.InspectBuildpack(inspectOptions)
				h.AssertNil(t, err)

				h.AssertEq(t, info, expectedInfo)
			})

			// TODO add test case when buildpack is flattened
		})

		when("inspecting local buildpack archive", func() {
			it.Before(func() {
				expectedInfo.Location = buildpack.URILocator

				assert := h.NewAssertionManager(t)
				writeBuildpackArchive(buildpackPath, tmpDir, assert)
			})

			it("succeeds", func() {
				mockDownloader.EXPECT().Download(gomock.Any(), buildpackPath).Return(blob.NewBlob(buildpackPath), nil)
				inspectOptions := client.InspectBuildpackOptions{
					BuildpackName: buildpackPath,
					Daemon:        false,
				}
				info, err := subject.InspectBuildpack(inspectOptions)
				h.AssertNil(t, err)

				h.AssertEq(t, info, expectedInfo)
			})

			// TODO add test case when buildpack is flattened
		})

		when("inspecting an image", func() {
			for _, useDaemon := range []bool{true, false} {
				useDaemon := useDaemon
				when(fmt.Sprintf("daemon is %t", useDaemon), func() {
					it.Before(func() {
						expectedInfo.Location = buildpack.PackageLocator
						if useDaemon {
							mockImageFetcher.EXPECT().Fetch(gomock.Any(), "some/buildpack", image.FetchOptions{Daemon: true, PullPolicy: image.PullNever}).Return(buildpackImage, nil)
						} else {
							mockImageFetcher.EXPECT().Fetch(gomock.Any(), "some/buildpack", image.FetchOptions{Daemon: false, PullPolicy: image.PullNever}).Return(buildpackImage, nil)
						}
					})

					it("succeeds", func() {
						inspectOptions := client.InspectBuildpackOptions{
							BuildpackName: "docker://some/buildpack",
							Daemon:        useDaemon,
						}
						info, err := subject.InspectBuildpack(inspectOptions)
						h.AssertNil(t, err)

						h.AssertEq(t, info, expectedInfo)
					})
				})
			}
		})
	})
	when("failure cases", func() {
		when("invalid buildpack name", func() {
			it("returns an error", func() {
				invalidBuildpackName := ""
				inspectOptions := client.InspectBuildpackOptions{
					BuildpackName: invalidBuildpackName,
				}
				_, err := subject.InspectBuildpack(inspectOptions)

				h.AssertError(t, err, "unable to handle locator ")
				h.AssertFalse(t, errors.Is(err, image.ErrNotFound))
			})
		})
		when("buildpack image", func() {
			when("unable to fetch buildpack image", func() {
				it.Before(func() {
					mockImageFetcher.EXPECT().Fetch(gomock.Any(), "missing/buildpack", image.FetchOptions{Daemon: true, PullPolicy: image.PullNever}).Return(nil, errors.Wrapf(image.ErrNotFound, "big bad error"))
				})
				it("returns an ErrNotFound error", func() {
					inspectOptions := client.InspectBuildpackOptions{
						BuildpackName: "docker://missing/buildpack",
						Daemon:        true,
					}
					_, err := subject.InspectBuildpack(inspectOptions)
					h.AssertTrue(t, errors.Is(err, image.ErrNotFound))
				})
			})

			when("image does not have buildpackage metadata", func() {
				it.Before(func() {
					fakeImage := fakes.NewImage("empty", "", nil)
					h.AssertNil(t, fakeImage.SetLabel(dist.BuildpackLayersLabel, ":::"))
					mockImageFetcher.EXPECT().Fetch(gomock.Any(), "missing-metadata/buildpack", image.FetchOptions{Daemon: true, PullPolicy: image.PullNever}).Return(fakeImage, nil)
				})
				it("returns an error", func() {
					inspectOptions := client.InspectBuildpackOptions{
						BuildpackName: "docker://missing-metadata/buildpack",
						Daemon:        true,
					}
					_, err := subject.InspectBuildpack(inspectOptions)

					h.AssertError(t, err, fmt.Sprintf("unable to get image label %s", dist.BuildpackLayersLabel))
					h.AssertFalse(t, errors.Is(err, image.ErrNotFound))
				})
			})
		})
		when("buildpack archive", func() {
			when("archive is not a buildpack", func() {
				it.Before(func() {
					invalidBuildpackPath := filepath.Join(tmpDir, "fake-buildpack-path")
					h.AssertNil(t, os.WriteFile(invalidBuildpackPath, []byte("not a buildpack"), os.ModePerm))

					mockDownloader.EXPECT().Download(gomock.Any(), "https://invalid/buildpack").Return(blob.NewBlob(invalidBuildpackPath), nil)
				})
				it("returns an error", func() {
					inspectOptions := client.InspectBuildpackOptions{
						BuildpackName: "https://invalid/buildpack",
						Daemon:        true,
					}

					_, err := subject.InspectBuildpack(inspectOptions)
					h.AssertNotNil(t, err)
					h.AssertFalse(t, errors.Is(err, image.ErrNotFound))
					h.AssertError(t, err, "unable to fetch config from buildpack blob:")
				})
			})
			when("unable to download buildpack archive", func() {
				it.Before(func() {
					mockDownloader.EXPECT().Download(gomock.Any(), "https://missing/buildpack").Return(nil, errors.New("unable to download archive"))
				})
				it("returns a untyped error", func() {
					inspectOptions := client.InspectBuildpackOptions{
						BuildpackName: "https://missing/buildpack",
						Daemon:        true,
					}

					_, err := subject.InspectBuildpack(inspectOptions)
					h.AssertNotNil(t, err)
					h.AssertFalse(t, errors.Is(err, image.ErrNotFound))
					h.AssertError(t, err, "unable to download archive")
				})
			})
		})

		when("buildpack on registry", func() {
			when("unable to get registry", func() {
				it("returns an error", func() {
					registryBuildpack := "urn:cnb:registry:example/foo"
					inspectOptions := client.InspectBuildpackOptions{
						BuildpackName: registryBuildpack,
						Daemon:        true,
						Registry:      ":::",
					}

					_, err := subject.InspectBuildpack(inspectOptions)

					h.AssertError(t, err, "invalid registry :::")
					h.AssertFalse(t, errors.Is(err, image.ErrNotFound))
				})
			})
			when("buildpack is not on registry", func() {
				var registryFixture string
				var configPath string

				it.Before(func() {
					registryFixture = h.CreateRegistryFixture(t, tmpDir, filepath.Join("testdata", "registry"))
					packHome := filepath.Join(tmpDir, "packHome")
					h.AssertNil(t, os.Setenv("PACK_HOME", packHome))
					configPath = filepath.Join(packHome, "config.toml")
					h.AssertNil(t, cfg.Write(cfg.Config{
						Registries: []cfg.Registry{
							{
								Name: "some-registry",
								Type: "github",
								URL:  registryFixture,
							},
						},
					}, configPath))
				})
				it("returns an error", func() {
					registryBuildpack := "urn:cnb:registry:example/not-present"
					inspectOptions := client.InspectBuildpackOptions{
						BuildpackName: registryBuildpack,
						Daemon:        true,
						Registry:      "some-registry",
					}

					_, err := subject.InspectBuildpack(inspectOptions)

					h.AssertError(t, err, "unable to find 'urn:cnb:registry:example/not-present' in registry:")
				})
			})
			when("unable to fetch buildpack from registry", func() {
				var registryFixture string
				var configPath string

				it.Before(func() {
					registryFixture = h.CreateRegistryFixture(t, tmpDir, filepath.Join("testdata", "registry"))
					packHome := filepath.Join(tmpDir, "packHome")
					h.AssertNil(t, os.Setenv("PACK_HOME", packHome))

					configPath = filepath.Join(packHome, "config.toml")
					h.AssertNil(t, cfg.Write(cfg.Config{
						Registries: []cfg.Registry{
							{
								Name: "some-registry",
								Type: "github",
								URL:  registryFixture,
							},
						},
					}, configPath))
					mockImageFetcher.EXPECT().Fetch(
						gomock.Any(),
						"example.com/some/package@sha256:2560f05307e8de9d830f144d09556e19dd1eb7d928aee900ed02208ae9727e7a",
						image.FetchOptions{Daemon: false, PullPolicy: image.PullNever}).Return(nil, image.ErrNotFound)
				})
				it("returns an untyped error", func() {
					registryBuildpack := "urn:cnb:registry:example/foo"
					inspectOptions := client.InspectBuildpackOptions{
						BuildpackName: registryBuildpack,
						Daemon:        true,
						Registry:      "some-registry",
					}

					_, err := subject.InspectBuildpack(inspectOptions)
					h.AssertNotNil(t, err)
					h.AssertFalse(t, errors.Is(err, image.ErrNotFound))
					h.AssertError(t, err, "error pulling registry specified image")
				})
			})
		})
	})
}

// write an OCI image using GGCR lib
func writeBuildpackArchive(buildpackPath, tmpDir string, assert h.AssertionManager) {
	layoutDir := filepath.Join(tmpDir, "layout")
	imgIndex := empty.Index
	img := empty.Image
	c, err := img.ConfigFile()
	assert.Nil(err)

	c.Config.Labels = map[string]string{}
	c.Config.Labels[buildpack.MetadataLabel] = buildpackageMetadataTag
	c.Config.Labels[dist.BuildpackLayersLabel] = buildpackLayersTag
	img, err = mutate.Config(img, c.Config)
	assert.Nil(err)

	p, err := layout.Write(layoutDir, imgIndex)
	assert.Nil(err)

	assert.Nil(p.AppendImage(img))
	assert.Nil(err)

	buildpackWriter, err := os.Create(buildpackPath)
	assert.Nil(err)
	defer buildpackWriter.Close()

	tw := tar.NewWriter(buildpackWriter)
	defer tw.Close()

	assert.Nil(archive.WriteDirToTar(tw, layoutDir, "/", 0, 0, 0755, true, false, nil))
}
