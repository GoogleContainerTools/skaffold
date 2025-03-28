package build_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/apex/log"
	ifakes "github.com/buildpacks/imgutil/fakes"
	"github.com/buildpacks/lifecycle/api"
	"github.com/buildpacks/lifecycle/platform/files"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/build"
	"github.com/buildpacks/pack/internal/build/fakes"
	"github.com/buildpacks/pack/internal/paths"
	"github.com/buildpacks/pack/pkg/cache"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

// TestLifecycleExecution are unit tests that test each possible phase to ensure they are executed with the proper parameters
func TestLifecycleExecution(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)

	spec.Run(t, "phases", testLifecycleExecution, spec.Report(report.Terminal{}), spec.Sequential())
}

func testLifecycleExecution(t *testing.T, when spec.G, it spec.S) {
	var (
		dockerConfigDir string
		tmpDir          string

		// lifecycle options
		providedClearCache     bool
		providedPublish        bool
		providedUseCreator     bool
		providedLayout         bool
		providedDockerHost     string
		providedNetworkMode    = "some-network-mode"
		providedRunImage       = "some-run-image"
		providedTargetImage    = "some-target-image"
		providedAdditionalTags = []string{"some-additional-tag1", "some-additional-tag2"}
		providedVolumes        = []string{"some-mount-source:/some-mount-target"}

		// builder options
		providedBuilderImage = "some-registry.com/some-namespace/some-builder-name"
		withOS               = "linux"
		platformAPI          = build.SupportedPlatformAPIVersions[0] // TODO: update the tests to target the latest api by default and make earlier apis special cases
		providedUID          = 2222
		providedGID          = 3333
		providedOrderExt     dist.Order

		lifecycle        *build.LifecycleExecution
		fakeBuildCache   = newFakeVolumeCache()
		fakeLaunchCache  *fakes.FakeCache
		fakeKanikoCache  *fakes.FakeCache
		fakePhase        *fakes.FakePhase
		fakePhaseFactory *fakes.FakePhaseFactory
		fakeFetcher      fakeImageFetcher
		configProvider   *build.PhaseConfigProvider

		extensionsForBuild, extensionsForRun bool
		extensionsRunImageName               string
		extensionsRunImageIdentifier         string
		useCreatorWithExtensions             bool
	)

	var configureDefaultTestLifecycle = func(opts *build.LifecycleOptions) {
		opts.AdditionalTags = providedAdditionalTags
		opts.BuilderImage = providedBuilderImage
		opts.ClearCache = providedClearCache
		opts.DockerHost = providedDockerHost
		opts.Network = providedNetworkMode
		opts.Publish = providedPublish
		opts.RunImage = providedRunImage
		opts.UseCreator = providedUseCreator
		opts.Volumes = providedVolumes
		opts.Layout = providedLayout
		opts.Keychain = authn.DefaultKeychain
		opts.UseCreatorWithExtensions = useCreatorWithExtensions

		targetImageRef, err := name.ParseReference(providedTargetImage)
		h.AssertNil(t, err)
		opts.Image = targetImageRef
	}

	var lifecycleOps = []func(*build.LifecycleOptions){configureDefaultTestLifecycle}

	it.Before(func() {
		// Avoid contaminating tests with existing docker configuration.
		// GGCR resolves the default keychain by inspecting DOCKER_CONFIG - this is used by the Analyze step
		// when constructing the auth config (see `auth.BuildEnvVar` in phases.go).
		var err error
		dockerConfigDir, err = os.MkdirTemp("", "empty-docker-config-dir")
		h.AssertNil(t, err)
		h.AssertNil(t, os.Setenv("DOCKER_CONFIG", dockerConfigDir))

		image := ifakes.NewImage("some-image", "", nil)
		h.AssertNil(t, image.SetOS(withOS))

		fakeBuilder, err := fakes.NewFakeBuilder(
			fakes.WithSupportedPlatformAPIs([]*api.Version{platformAPI}),
			fakes.WithUID(providedUID),
			fakes.WithGID(providedGID),
			fakes.WithOrderExtensions(providedOrderExt),
			fakes.WithImage(image),
		)
		h.AssertNil(t, err)
		fakeFetcher = fakeImageFetcher{
			callCount:           0,
			calledWithArgAtCall: make(map[int]string),
		}
		withFakeFetchRunImageFunc := func(opts *build.LifecycleOptions) {
			opts.FetchRunImageWithLifecycleLayer = newFakeFetchRunImageFunc(&fakeFetcher)
		}
		lifecycleOps = append(lifecycleOps, fakes.WithBuilder(fakeBuilder), withFakeFetchRunImageFunc)

		tmpDir, err = os.MkdirTemp("", "pack.unit")
		h.AssertNil(t, err)
		lifecycle = newTestLifecycleExec(t, true, tmpDir, lifecycleOps...)

		// construct fixtures for extensions
		if extensionsForBuild {
			if platformAPI.LessThan("0.13") {
				err = os.MkdirAll(filepath.Join(tmpDir, "generated", "build", "some-buildpack-id"), 0755)
				h.AssertNil(t, err)
			} else {
				err = os.MkdirAll(filepath.Join(tmpDir, "generated", "some-buildpack-id"), 0755)
				h.AssertNil(t, err)
				_, err = os.Create(filepath.Join(tmpDir, "generated", "some-buildpack-id", "build.Dockerfile"))
				h.AssertNil(t, err)
			}
		}
		amd := files.Analyzed{RunImage: &files.RunImage{
			Extend: false,
			Image:  "",
		}}
		if extensionsForRun {
			amd.RunImage.Extend = true
		}
		if extensionsRunImageName != "" {
			amd.RunImage.Image = extensionsRunImageName
		}
		if extensionsRunImageIdentifier != "" {
			amd.RunImage.Reference = extensionsRunImageIdentifier
		}
		f, err := os.Create(filepath.Join(tmpDir, "analyzed.toml"))
		h.AssertNil(t, err)
		toml.NewEncoder(f).Encode(amd)
		h.AssertNil(t, f.Close())

		fakeLaunchCache = fakes.NewFakeCache()
		fakeLaunchCache.ReturnForType = cache.Volume
		fakeLaunchCache.ReturnForName = "some-launch-cache"

		fakeKanikoCache = fakes.NewFakeCache()
		fakeKanikoCache.ReturnForType = cache.Volume
		fakeKanikoCache.ReturnForName = "some-kaniko-cache"

		fakePhase = &fakes.FakePhase{}
		fakePhaseFactory = fakes.NewFakePhaseFactory(fakes.WhichReturnsForNew(fakePhase))
	})

	it.After(func() {
		h.AssertNil(t, os.Unsetenv("DOCKER_CONFIG"))
		h.AssertNil(t, os.RemoveAll(dockerConfigDir))
		_ = os.RemoveAll(tmpDir)
	})

	when("#NewLifecycleExecution", func() {
		when("lifecycle supports multiple platform APIs", func() {
			it("selects the latest supported version", func() {
				fakeBuilder, err := fakes.NewFakeBuilder(fakes.WithSupportedPlatformAPIs([]*api.Version{
					api.MustParse("0.2"),
					api.MustParse("0.3"),
					api.MustParse("0.4"),
					api.MustParse("0.5"),
					api.MustParse("0.6"),
					api.MustParse("0.7"),
					api.MustParse("0.8"),
				}))
				h.AssertNil(t, err)

				lifecycleExec := newTestLifecycleExec(t, false, "some-temp-dir", fakes.WithBuilder(fakeBuilder))
				h.AssertEq(t, lifecycleExec.PlatformAPI().String(), "0.8")
			})
		})

		when("supported platform API is deprecated", func() {
			it("selects the deprecated version", func() {
				fakeBuilder, err := fakes.NewFakeBuilder(
					fakes.WithDeprecatedPlatformAPIs([]*api.Version{api.MustParse("0.4")}),
					fakes.WithSupportedPlatformAPIs([]*api.Version{api.MustParse("1.2")}),
				)
				h.AssertNil(t, err)

				lifecycleExec := newTestLifecycleExec(t, false, "some-temp-dir", fakes.WithBuilder(fakeBuilder))
				h.AssertEq(t, lifecycleExec.PlatformAPI().String(), "0.4")
			})
		})

		when("pack doesn't support any lifecycle supported platform API", func() {
			it("errors", func() {
				fakeBuilder, err := fakes.NewFakeBuilder(
					fakes.WithSupportedPlatformAPIs([]*api.Version{api.MustParse("1.2")}),
				)
				h.AssertNil(t, err)

				_, err = newTestLifecycleExecErr(t, false, "some-temp-dir", fakes.WithBuilder(fakeBuilder))
				h.AssertError(t, err, "unable to find a supported Platform API version")
			})
		})
	})

	when("FindLatestSupported", func() {
		it("chooses a shared version", func() {
			version, err := build.FindLatestSupported([]*api.Version{api.MustParse("0.6"), api.MustParse("0.7"), api.MustParse("0.8")}, []string{"0.7"})
			h.AssertNil(t, err)
			h.AssertEq(t, version, api.MustParse("0.7"))
		})

		it("chooses a shared version, highest builder supported version", func() {
			version, err := build.FindLatestSupported([]*api.Version{api.MustParse("0.4"), api.MustParse("0.5"), api.MustParse("0.7")}, []string{"0.7", "0.8"})
			h.AssertNil(t, err)
			h.AssertEq(t, version, api.MustParse("0.7"))
		})

		it("chooses a shared version, lowest builder supported version", func() {
			version, err := build.FindLatestSupported([]*api.Version{api.MustParse("0.4"), api.MustParse("0.5"), api.MustParse("0.7")}, []string{"0.1", "0.2", "0.4"})
			h.AssertNil(t, err)
			h.AssertEq(t, version, api.MustParse("0.4"))
		})

		it("Interprets empty lifecycle versions list as lack of constraints", func() {
			version, err := build.FindLatestSupported([]*api.Version{api.MustParse("0.6"), api.MustParse("0.7")}, []string{})
			h.AssertNil(t, err)
			h.AssertEq(t, version, api.MustParse("0.7"))
		})

		it("errors with no shared version, builder has no versions supported for some reason", func() {
			_, err := build.FindLatestSupported([]*api.Version{}, []string{"0.7"})
			h.AssertNotNil(t, err)
		})

		it("errors with no shared version, builder less than lifecycle", func() {
			_, err := build.FindLatestSupported([]*api.Version{api.MustParse("0.4"), api.MustParse("0.5")}, []string{"0.7", "0.8"})
			h.AssertNotNil(t, err)
		})

		it("errors with no shared version, builder greater than lifecycle", func() {
			_, err := build.FindLatestSupported([]*api.Version{api.MustParse("0.8"), api.MustParse("0.9")}, []string{"0.6", "0.7"})
			h.AssertNotNil(t, err)
		})
	})

	when("Run", func() {
		var (
			imageName   name.Tag
			fakeBuilder *fakes.FakeBuilder
			outBuf      bytes.Buffer
			logger      *logging.LogWithWriters
			docker      *fakeDockerClient
			fakeTermui  *fakes.FakeTermui
		)

		it.Before(func() {
			var err error
			imageName, err = name.NewTag("/some/image", name.WeakValidation)
			h.AssertNil(t, err)

			fakeTermui = &fakes.FakeTermui{}

			fakeBuilder, err = fakes.NewFakeBuilder(fakes.WithSupportedPlatformAPIs([]*api.Version{api.MustParse("0.3")}))
			h.AssertNil(t, err)
			logger = logging.NewLogWithWriters(&outBuf, &outBuf)
			docker = &fakeDockerClient{}
			h.AssertNil(t, err)
			fakePhaseFactory = fakes.NewFakePhaseFactory()
		})

		when("Run using creator", func() {
			it("succeeds", func() {
				opts := build.LifecycleOptions{
					Publish:      false,
					ClearCache:   false,
					RunImage:     "test",
					Image:        imageName,
					Builder:      fakeBuilder,
					TrustBuilder: false,
					UseCreator:   true,
					Termui:       fakeTermui,
				}

				lifecycle, err := build.NewLifecycleExecution(logger, docker, "some-temp-dir", opts)
				h.AssertNil(t, err)
				h.AssertEq(t, filepath.Base(lifecycle.AppDir()), "workspace")

				err = lifecycle.Run(context.Background(), func(execution *build.LifecycleExecution) build.PhaseFactory {
					return fakePhaseFactory
				})
				h.AssertNil(t, err)

				h.AssertEq(t, len(fakePhaseFactory.NewCalledWithProvider), 1)

				for _, entry := range fakePhaseFactory.NewCalledWithProvider {
					if entry.Name() == "creator" {
						h.AssertSliceContains(t, entry.ContainerConfig().Cmd, "/some/image")
					}
				}
			})

			when("Run with workspace dir", func() {
				it("succeeds", func() {
					opts := build.LifecycleOptions{
						Publish:      false,
						ClearCache:   false,
						RunImage:     "test",
						Image:        imageName,
						Builder:      fakeBuilder,
						TrustBuilder: true,
						Workspace:    "app",
						UseCreator:   true,
						Termui:       fakeTermui,
					}

					lifecycle, err := build.NewLifecycleExecution(logger, docker, "some-temp-dir", opts)
					h.AssertNil(t, err)

					err = lifecycle.Run(context.Background(), func(execution *build.LifecycleExecution) build.PhaseFactory {
						return fakePhaseFactory
					})
					h.AssertNil(t, err)

					h.AssertEq(t, len(fakePhaseFactory.NewCalledWithProvider), 1)

					for _, entry := range fakePhaseFactory.NewCalledWithProvider {
						if entry.Name() == "creator" {
							h.AssertSliceContainsInOrder(t, entry.ContainerConfig().Cmd, "-app", "/app")
							h.AssertSliceContains(t, entry.ContainerConfig().Cmd, "/some/image")
						}
					}
				})
			})

			when("there are extensions", func() {
				providedUseCreator = true
				providedOrderExt = dist.Order{dist.OrderEntry{Group: []dist.ModuleRef{ /* don't care */ }}}

				when("platform < 0.10", func() {
					platformAPI = api.MustParse("0.9")

					it("succeeds", func() {
						err := lifecycle.Run(context.Background(), func(execution *build.LifecycleExecution) build.PhaseFactory {
							return fakePhaseFactory
						})
						h.AssertNil(t, err)

						h.AssertEq(t, len(fakePhaseFactory.NewCalledWithProvider), 1)

						for _, entry := range fakePhaseFactory.NewCalledWithProvider {
							if entry.Name() == "creator" {
								h.AssertSliceContains(t, entry.ContainerConfig().Cmd, providedTargetImage)
							}
						}
					})
				})

				when("platform >= 0.10", func() {
					platformAPI = api.MustParse("0.10")

					it("errors", func() {
						err := lifecycle.Run(context.Background(), func(execution *build.LifecycleExecution) build.PhaseFactory {
							return fakePhaseFactory
						})
						h.AssertNotNil(t, err)
					})

					when("use creator with extensions supported by the lifecycle", func() {
						useCreatorWithExtensions = true

						it("allows the build to proceed (but the creator will error if extensions are detected)", func() {
							err := lifecycle.Run(context.Background(), func(execution *build.LifecycleExecution) build.PhaseFactory {
								return fakePhaseFactory
							})
							h.AssertNil(t, err)

							h.AssertEq(t, len(fakePhaseFactory.NewCalledWithProvider), 1)

							for _, entry := range fakePhaseFactory.NewCalledWithProvider {
								if entry.Name() == "creator" {
									h.AssertSliceContains(t, entry.ContainerConfig().Cmd, providedTargetImage)
								}
							}
						})
					})
				})
			})
		})

		when("Run without using creator", func() {
			when("platform < 0.7", func() {
				it("calls the phases with the right order", func() {
					opts := build.LifecycleOptions{
						Publish:      false,
						ClearCache:   false,
						RunImage:     "test",
						Image:        imageName,
						Builder:      fakeBuilder,
						TrustBuilder: false,
						UseCreator:   false,
						Termui:       fakeTermui,
					}

					lifecycle, err := build.NewLifecycleExecution(logger, docker, "some-temp-dir", opts)
					h.AssertNil(t, err)

					err = lifecycle.Run(context.Background(), func(execution *build.LifecycleExecution) build.PhaseFactory {
						return fakePhaseFactory
					})
					h.AssertNil(t, err)

					h.AssertEq(t, len(fakePhaseFactory.NewCalledWithProvider), 5)
					expectedPhases := []string{
						"detector", "analyzer", "restorer", "builder", "exporter",
					}
					for i, entry := range fakePhaseFactory.NewCalledWithProvider {
						h.AssertEq(t, entry.Name(), expectedPhases[i])
					}
				})
			})

			when("platform >= 0.7", func() {
				it("calls the phases with the right order", func() {
					fakeBuilder, err := fakes.NewFakeBuilder(fakes.WithSupportedPlatformAPIs([]*api.Version{api.MustParse("0.7")}))
					h.AssertNil(t, err)

					opts := build.LifecycleOptions{
						Publish:      false,
						ClearCache:   false,
						RunImage:     "test",
						Image:        imageName,
						Builder:      fakeBuilder,
						TrustBuilder: false,
						UseCreator:   false,
						Termui:       fakeTermui,
					}

					lifecycle, err := build.NewLifecycleExecution(logger, docker, "some-temp-dir", opts)
					h.AssertNil(t, err)

					err = lifecycle.Run(context.Background(), func(execution *build.LifecycleExecution) build.PhaseFactory {
						return fakePhaseFactory
					})
					h.AssertNil(t, err)

					h.AssertEq(t, len(fakePhaseFactory.NewCalledWithProvider), 5)
					expectedPhases := []string{
						"analyzer", "detector", "restorer", "builder", "exporter",
					}
					for i, entry := range fakePhaseFactory.NewCalledWithProvider {
						h.AssertEq(t, entry.Name(), expectedPhases[i])
					}
				})
			})

			it("succeeds", func() {
				opts := build.LifecycleOptions{
					Publish:      false,
					ClearCache:   false,
					RunImage:     "test",
					Image:        imageName,
					Builder:      fakeBuilder,
					TrustBuilder: false,
					UseCreator:   false,
					Termui:       fakeTermui,
				}

				lifecycle, err := build.NewLifecycleExecution(logger, docker, "some-temp-dir", opts)
				h.AssertNil(t, err)

				err = lifecycle.Run(context.Background(), func(execution *build.LifecycleExecution) build.PhaseFactory {
					return fakePhaseFactory
				})
				h.AssertNil(t, err)

				h.AssertEq(t, len(fakePhaseFactory.NewCalledWithProvider), 5)

				for _, entry := range fakePhaseFactory.NewCalledWithProvider {
					switch entry.Name() {
					case "exporter":
						h.AssertSliceContains(t, entry.ContainerConfig().Cmd, "/some/image")
					case "analyzer":
						h.AssertSliceContains(t, entry.ContainerConfig().Cmd, "/some/image")
					}
				}
			})

			when("Run with workspace dir", func() {
				it("succeeds", func() {
					opts := build.LifecycleOptions{
						Publish:      false,
						ClearCache:   false,
						RunImage:     "test",
						Image:        imageName,
						Builder:      fakeBuilder,
						TrustBuilder: false,
						Workspace:    "app",
						UseCreator:   false,
						Termui:       fakeTermui,
					}

					lifecycle, err := build.NewLifecycleExecution(logger, docker, "some-temp-dir", opts)
					h.AssertNil(t, err)
					h.AssertEq(t, filepath.Base(lifecycle.AppDir()), "app")

					err = lifecycle.Run(context.Background(), func(execution *build.LifecycleExecution) build.PhaseFactory {
						return fakePhaseFactory
					})
					h.AssertNil(t, err)

					h.AssertEq(t, len(fakePhaseFactory.NewCalledWithProvider), 5)

					appCount := 0
					for _, entry := range fakePhaseFactory.NewCalledWithProvider {
						switch entry.Name() {
						case "detector", "builder", "exporter":
							h.AssertSliceContainsInOrder(t, entry.ContainerConfig().Cmd, "-app", "/app")
							appCount++
						}
					}
					h.AssertEq(t, appCount, 3)
				})
			})

			when("--clear-cache", func() {
				providedUseCreator = false
				providedClearCache = true
				lifecycleOps = append(lifecycleOps, func(options *build.LifecycleOptions) { // allow buildCache.Clear to succeed without requiring the docker daemon to be running
					options.Cache.Build.Format = cache.CacheBind
				})

				when("platform < 0.10", func() {
					platformAPI = api.MustParse("0.9")

					it("does not run restore", func() {
						err := lifecycle.Run(context.Background(), func(execution *build.LifecycleExecution) build.PhaseFactory {
							return fakePhaseFactory
						})
						h.AssertNil(t, err)

						h.AssertEq(t, len(fakePhaseFactory.NewCalledWithProvider), 4)
					})
				})

				when("platform >= 0.10", func() {
					platformAPI = api.MustParse("0.10")

					it("runs restore", func() {
						err := lifecycle.Run(context.Background(), func(execution *build.LifecycleExecution) build.PhaseFactory {
							return fakePhaseFactory
						})
						h.AssertNil(t, err)

						h.AssertEq(t, len(fakePhaseFactory.NewCalledWithProvider), 5)
					})
				})
			})

			when("extensions", func() {
				providedUseCreator = false
				providedOrderExt = dist.Order{dist.OrderEntry{Group: []dist.ModuleRef{ /* don't care */ }}}

				when("for build", func() {
					when("present in <layers>/generated/<buildpack-id>", func() {
						extensionsForBuild = true

						when("platform >= 0.13", func() {
							platformAPI = api.MustParse("0.13")

							it("runs the extender (build)", func() {
								err := lifecycle.Run(context.Background(), func(execution *build.LifecycleExecution) build.PhaseFactory {
									return fakePhaseFactory
								})
								h.AssertNil(t, err)

								h.AssertEq(t, len(fakePhaseFactory.NewCalledWithProvider), 5)

								var found bool
								for _, entry := range fakePhaseFactory.NewCalledWithProvider {
									if entry.Name() == "extender" {
										found = true
									}
								}
								h.AssertEq(t, found, true)
							})
						})
					})

					when("present in <layers>/generated/build", func() {
						extensionsForBuild = true

						when("platform < 0.10", func() {
							platformAPI = api.MustParse("0.9")

							it("runs the builder", func() {
								err := lifecycle.Run(context.Background(), func(execution *build.LifecycleExecution) build.PhaseFactory {
									return fakePhaseFactory
								})
								h.AssertNil(t, err)

								h.AssertEq(t, len(fakePhaseFactory.NewCalledWithProvider), 5)

								var found bool
								for _, entry := range fakePhaseFactory.NewCalledWithProvider {
									if entry.Name() == "builder" {
										found = true
									}
								}
								h.AssertEq(t, found, true)
							})
						})

						when("platform 0.10 to 0.12", func() {
							platformAPI = api.MustParse("0.10")

							it("runs the extender (build)", func() {
								err := lifecycle.Run(context.Background(), func(execution *build.LifecycleExecution) build.PhaseFactory {
									return fakePhaseFactory
								})
								h.AssertNil(t, err)

								h.AssertEq(t, len(fakePhaseFactory.NewCalledWithProvider), 5)

								var found bool
								for _, entry := range fakePhaseFactory.NewCalledWithProvider {
									if entry.Name() == "extender" {
										found = true
									}
								}
								h.AssertEq(t, found, true)
							})
						})
					})

					when("not present in <layers>/generated/build", func() {
						platformAPI = api.MustParse("0.10")

						it("runs the builder", func() {
							err := lifecycle.Run(context.Background(), func(execution *build.LifecycleExecution) build.PhaseFactory {
								return fakePhaseFactory
							})
							h.AssertNil(t, err)

							h.AssertEq(t, len(fakePhaseFactory.NewCalledWithProvider), 5)

							var found bool
							for _, entry := range fakePhaseFactory.NewCalledWithProvider {
								if entry.Name() == "builder" {
									found = true
								}
							}
							h.AssertEq(t, found, true)
						})
					})
				})

				when("for run", func() {
					when("analyzed.toml run image", func() {
						when("matches provided run image", func() {
							it("does nothing", func() {
								err := lifecycle.Run(context.Background(), func(execution *build.LifecycleExecution) build.PhaseFactory {
									return fakePhaseFactory
								})
								h.AssertNil(t, err)
								h.AssertEq(t, fakeFetcher.callCount, 0)
							})
						})

						when("does not match provided run image", func() {
							extensionsRunImageName = "some-new-run-image"
							extensionsRunImageIdentifier = "some-new-run-image-identifier"

							it("pulls the new run image", func() {
								err := lifecycle.Run(context.Background(), func(execution *build.LifecycleExecution) build.PhaseFactory {
									return fakePhaseFactory
								})
								h.AssertNil(t, err)
								h.AssertEq(t, fakeFetcher.callCount, 2)
								h.AssertEq(t, fakeFetcher.calledWithArgAtCall[0], "some-new-run-image")
								h.AssertEq(t, fakeFetcher.calledWithArgAtCall[1], "some-new-run-image-identifier")
							})
						})
					})

					when("analyzed.toml run image extend", func() {
						when("true", func() {
							extensionsForRun = true

							when("platform >= 0.12", func() {
								platformAPI = api.MustParse("0.12")

								it("runs the extender (run)", func() {
									err := lifecycle.Run(context.Background(), func(execution *build.LifecycleExecution) build.PhaseFactory {
										return fakePhaseFactory
									})
									h.AssertNil(t, err)

									h.AssertEq(t, len(fakePhaseFactory.NewCalledWithProvider), 6)

									var found bool
									for _, entry := range fakePhaseFactory.NewCalledWithProvider {
										if entry.Name() == "extender" {
											found = true
										}
									}
									h.AssertEq(t, found, true)
								})
							})

							when("platform < 0.12", func() {
								platformAPI = api.MustParse("0.11")

								it("doesn't run the extender", func() {
									err := lifecycle.Run(context.Background(), func(execution *build.LifecycleExecution) build.PhaseFactory {
										return fakePhaseFactory
									})
									h.AssertNil(t, err)

									h.AssertEq(t, len(fakePhaseFactory.NewCalledWithProvider), 5)

									var found bool
									for _, entry := range fakePhaseFactory.NewCalledWithProvider {
										if entry.Name() == "extender" {
											found = true
										}
									}
									h.AssertEq(t, found, false)
								})
							})
						})

						when("false", func() {
							platformAPI = api.MustParse("0.12")

							it("doesn't run the extender", func() {
								err := lifecycle.Run(context.Background(), func(execution *build.LifecycleExecution) build.PhaseFactory {
									return fakePhaseFactory
								})
								h.AssertNil(t, err)

								h.AssertEq(t, len(fakePhaseFactory.NewCalledWithProvider), 5)

								var found bool
								for _, entry := range fakePhaseFactory.NewCalledWithProvider {
									if entry.Name() == "extender" {
										found = true
									}
								}
								h.AssertEq(t, found, false)
							})
						})
					})
				})
			})
		})

		when("network is not provided", func() {
			it("creates an ephemeral bridge network", func() {
				beforeNetworks := func() int {
					networks, err := docker.NetworkList(context.Background(), network.CreateOptions{})
					h.AssertNil(t, err)
					return len(networks)
				}()

				opts := build.LifecycleOptions{
					Image:   imageName,
					Builder: fakeBuilder,
					Termui:  fakeTermui,
				}

				lifecycle, err := build.NewLifecycleExecution(logger, docker, "some-temp-dir", opts)
				h.AssertNil(t, err)

				err = lifecycle.Run(context.Background(), func(execution *build.LifecycleExecution) build.PhaseFactory {
					return fakePhaseFactory
				})
				h.AssertNil(t, err)

				for _, entry := range fakePhaseFactory.NewCalledWithProvider {
					h.AssertContains(t, string(entry.HostConfig().NetworkMode), "pack.local-network-")
					h.AssertEq(t, entry.HostConfig().NetworkMode.IsDefault(), false)
					h.AssertEq(t, entry.HostConfig().NetworkMode.IsHost(), false)
					h.AssertEq(t, entry.HostConfig().NetworkMode.IsNone(), false)
					h.AssertEq(t, entry.HostConfig().NetworkMode.IsPrivate(), true)
					h.AssertEq(t, entry.HostConfig().NetworkMode.IsUserDefined(), true)
				}

				afterNetworks := func() int {
					networks, err := docker.NetworkList(context.Background(), network.CreateOptions{})
					h.AssertNil(t, err)
					return len(networks)
				}()
				h.AssertEq(t, beforeNetworks, afterNetworks)
			})
		})

		when("Error cases", func() {
			when("passed invalid", func() {
				it("fails for cache-image", func() {
					opts := build.LifecycleOptions{
						Publish:      false,
						ClearCache:   false,
						RunImage:     "test",
						Image:        imageName,
						Builder:      fakeBuilder,
						TrustBuilder: false,
						UseCreator:   false,
						CacheImage:   "%%%",
						Termui:       fakeTermui,
					}

					lifecycle, err := build.NewLifecycleExecution(logger, docker, "some-temp-dir", opts)
					h.AssertNil(t, err)

					err = lifecycle.Run(context.Background(), func(execution *build.LifecycleExecution) build.PhaseFactory {
						return fakePhaseFactory
					})

					h.AssertError(t, err, fmt.Sprintf("invalid cache image name: %s", "could not parse reference: %%"))
				})

				it("fails for cache flags", func() {
					opts := build.LifecycleOptions{
						Publish:      false,
						ClearCache:   false,
						RunImage:     "test",
						Image:        imageName,
						Builder:      fakeBuilder,
						TrustBuilder: false,
						UseCreator:   false,
						Cache: cache.CacheOpts{
							Build: cache.CacheInfo{
								Format: cache.CacheImage,
								Source: "%%%",
							},
						},
						Termui: fakeTermui,
					}

					lifecycle, err := build.NewLifecycleExecution(logger, docker, "some-temp-dir", opts)
					h.AssertNil(t, err)

					err = lifecycle.Run(context.Background(), func(execution *build.LifecycleExecution) build.PhaseFactory {
						return fakePhaseFactory
					})

					h.AssertError(t, err, fmt.Sprintf("invalid cache image name: %s", "could not parse reference: %%"))
				})
			})
		})
	})

	when("#Create", func() {
		it.Before(func() {
			err := lifecycle.Create(context.Background(), fakeBuildCache, fakeLaunchCache, fakePhaseFactory)
			h.AssertNil(t, err)

			lastCallIndex := len(fakePhaseFactory.NewCalledWithProvider) - 1
			h.AssertNotEq(t, lastCallIndex, -1)

			configProvider = fakePhaseFactory.NewCalledWithProvider[lastCallIndex]
			h.AssertEq(t, configProvider.Name(), "creator")
		})

		it("creates a phase and then runs it", func() {
			h.AssertEq(t, fakePhase.CleanupCallCount, 1)
			h.AssertEq(t, fakePhase.RunCallCount, 1)
		})

		it("configures the phase with the expected arguments", func() {
			h.AssertIncludeAllExpectedPatterns(t,
				configProvider.ContainerConfig().Cmd,
				[]string{"-log-level", "debug"},
				[]string{"-run-image", providedRunImage},
				[]string{providedTargetImage},
			)
		})

		it("configures the phase with the expected network mode", func() {
			h.AssertEq(t, configProvider.HostConfig().NetworkMode, container.NetworkMode(providedNetworkMode))
		})

		when("clear cache", func() {
			providedClearCache = true

			it("configures the phase with the expected arguments", func() {
				h.AssertIncludeAllExpectedPatterns(t,
					configProvider.ContainerConfig().Cmd,
					[]string{"-skip-restore"},
				)
			})
		})

		when("clear cache is false", func() {
			it("configures the phase with the expected arguments", func() {
				h.AssertIncludeAllExpectedPatterns(t,
					configProvider.ContainerConfig().Cmd,
					[]string{"-cache-dir", "/cache"},
				)
			})
		})

		when("using a cache image", func() {
			providedClearCache = true
			fakeBuildCache = newFakeImageCache()

			it("configures the phase with the expected arguments", func() {
				h.AssertIncludeAllExpectedPatterns(t,
					configProvider.ContainerConfig().Cmd,
					[]string{"-skip-restore"},
					[]string{"-cache-image", "some-cache-image"},
				)
				h.AssertSliceNotContains(t, configProvider.HostConfig().Binds, ":/cache")
			})
		})

		when("additional tags are specified", func() {
			it("configures phases with additional tags", func() {
				h.AssertIncludeAllExpectedPatterns(t,
					configProvider.ContainerConfig().Cmd,
					[]string{"-tag", providedAdditionalTags[0], "-tag", providedAdditionalTags[1]},
				)
			})
		})

		when("publish", func() {
			providedPublish = true

			it("configures the phase with binds", func() {
				expectedBinds := providedVolumes
				expectedBinds = append(expectedBinds, "some-cache:/cache")

				h.AssertSliceContains(t, configProvider.HostConfig().Binds, expectedBinds...)
			})

			it("configures the phase with root", func() {
				h.AssertEq(t, configProvider.ContainerConfig().User, "root")
			})

			it("configures the phase with registry access", func() {
				h.AssertSliceContains(t, configProvider.ContainerConfig().Env, "CNB_REGISTRY_AUTH={}")
			})

			when("using a cache image", func() {
				fakeBuildCache = newFakeImageCache()

				it("configures the phase with the expected arguments", func() {
					h.AssertIncludeAllExpectedPatterns(t,
						configProvider.ContainerConfig().Cmd,
						[]string{"-cache-image", "some-cache-image"},
					)
					h.AssertSliceNotContains(t, configProvider.HostConfig().Binds, ":/cache")
				})
			})

			when("platform 0.3", func() {
				platformAPI = api.MustParse("0.3")

				it("doesn't hint at default process type", func() {
					h.AssertSliceNotContains(t, configProvider.ContainerConfig().Cmd, "-process-type")
				})
			})

			when("platform 0.4", func() {
				platformAPI = api.MustParse("0.4")

				it("hints at default process type", func() {
					h.AssertIncludeAllExpectedPatterns(t, configProvider.ContainerConfig().Cmd, []string{"-process-type", "web"})
				})
			})

			when("platform >= 0.6", func() {
				platformAPI = api.MustParse("0.6")

				when("no user provided process type is present", func() {
					it("doesn't provide 'web' as default process type", func() {
						h.AssertSliceNotContains(t, configProvider.ContainerConfig().Cmd, "-process-type")
					})
				})
			})
		})

		when("publish is false", func() {
			it("configures the phase with the expected arguments", func() {
				h.AssertIncludeAllExpectedPatterns(t,
					configProvider.ContainerConfig().Cmd,
					[]string{"-daemon"},
					[]string{"-launch-cache", "/launch-cache"},
				)
			})

			when("no docker-host", func() {
				it("configures the phase with daemon access", func() {
					h.AssertEq(t, configProvider.ContainerConfig().User, "root")
					h.AssertSliceContains(t, configProvider.HostConfig().Binds, "/var/run/docker.sock:/var/run/docker.sock")
				})
			})

			when("tcp docker-host", func() {
				providedDockerHost = `tcp://localhost:1234`

				it("configures the phase with daemon access with tcp docker-host", func() {
					h.AssertSliceNotContains(t, configProvider.HostConfig().Binds, "/var/run/docker.sock:/var/run/docker.sock")
					h.AssertSliceContains(t, configProvider.ContainerConfig().Env, "DOCKER_HOST=tcp://localhost:1234")
				})
			})

			when("alternative unix socket docker-host", func() {
				providedDockerHost = `unix:///home/user/docker.sock`

				it("configures the phase with daemon access", func() {
					h.AssertSliceContains(t, configProvider.HostConfig().Binds, "/home/user/docker.sock:/var/run/docker.sock")
				})
			})

			when("alternative windows pipe docker-host", func() {
				providedDockerHost = `npipe:\\\\.\pipe\docker_engine_alt`

				it("configures the phase with daemon access", func() {
					h.AssertSliceNotContains(t, configProvider.HostConfig().Binds, "/home/user/docker.sock:/var/run/docker.sock")
					h.AssertSliceContains(t, configProvider.HostConfig().Binds, `\\.\pipe\docker_engine_alt:\\.\pipe\docker_engine`)
				})
			})

			when("environment variable DOCKER_HOST is set", func() {
				providedDockerHost = `inherit`

				var (
					oldDH       string
					oldDHExists bool
				)

				it.Before(func() {
					oldDH, oldDHExists = os.LookupEnv("DOCKER_HOST")
					os.Setenv("DOCKER_HOST", "tcp://example.com:1234")
				})

				it.After(func() {
					if oldDHExists {
						os.Setenv("DOCKER_HOST", oldDH)
					} else {
						os.Unsetenv("DOCKER_HOST")
					}
				})

				it("configures the phase with daemon access with inherited docker-host", func() {
					lifecycle := newTestLifecycleExec(t, true, "some-temp-dir", lifecycleOps...)
					fakePhase := &fakes.FakePhase{}
					fakePhaseFactory := fakes.NewFakePhaseFactory(fakes.WhichReturnsForNew(fakePhase))
					err := lifecycle.Create(context.Background(), fakeBuildCache, fakeLaunchCache, fakePhaseFactory)
					h.AssertNil(t, err)

					lastCallIndex := len(fakePhaseFactory.NewCalledWithProvider) - 1
					h.AssertNotEq(t, lastCallIndex, -1)

					configProvider := fakePhaseFactory.NewCalledWithProvider[lastCallIndex]
					h.AssertSliceContains(t, configProvider.ContainerConfig().Env, "DOCKER_HOST=tcp://example.com:1234")
				})
			})

			when("docker-host with unknown protocol", func() {
				providedDockerHost = `withoutprotocol`

				it("configures the phase with daemon access", func() {
					h.AssertSliceContains(t, configProvider.ContainerConfig().Env, "DOCKER_HOST=withoutprotocol")
				})
			})

			it("configures the phase with binds", func() {
				expectedBinds := providedVolumes
				expectedBinds = append(expectedBinds, "some-cache:/cache", "some-launch-cache:/launch-cache")

				h.AssertSliceContains(t, configProvider.HostConfig().Binds, expectedBinds...)
			})

			when("platform 0.3", func() {
				platformAPI = api.MustParse("0.3")

				it("doesn't hint at default process type", func() {
					h.AssertSliceNotContains(t, configProvider.ContainerConfig().Cmd, "-process-type")
				})
			})

			when("platform 0.4", func() {
				platformAPI = api.MustParse("0.4")

				it("hints at default process type", func() {
					h.AssertIncludeAllExpectedPatterns(t, configProvider.ContainerConfig().Cmd, []string{"-process-type", "web"})
				})
			})

			when("platform >= 0.6", func() {
				platformAPI = api.MustParse("0.6")

				when("no user provided process type is present", func() {
					it("doesn't provide 'web' as default process type", func() {
						h.AssertSliceNotContains(t, configProvider.ContainerConfig().Cmd, "-process-type")
					})
				})
			})
		})

		when("override GID", func() {
			when("override GID is provided", func() {
				lifecycleOps = append(lifecycleOps, func(options *build.LifecycleOptions) {
					options.GID = 2
				})

				it("configures the phase with the expected arguments", func() {
					h.AssertIncludeAllExpectedPatterns(t,
						configProvider.ContainerConfig().Cmd,
						[]string{"-gid", "2"},
					)
				})
			})

			when("override GID is not provided", func() {
				lifecycleOps = append(lifecycleOps, func(options *build.LifecycleOptions) {
					options.GID = -1
				})

				it("gid is not added to the expected arguments", func() {
					h.AssertSliceNotContains(t, configProvider.ContainerConfig().Cmd, "-gid")
				})
			})
		})

		when("override UID", func() {
			when("override UID is provided", func() {
				lifecycleOps = append(lifecycleOps, func(options *build.LifecycleOptions) {
					options.UID = 1001
				})

				it("configures the phase with the expected arguments", func() {
					h.AssertIncludeAllExpectedPatterns(t,
						configProvider.ContainerConfig().Cmd,
						[]string{"-uid", "1001"},
					)
				})
			})

			when("override UID is not provided", func() {
				lifecycleOps = append(lifecycleOps, func(options *build.LifecycleOptions) {
					options.UID = -1
				})

				it("uid is not added to the expected arguments", func() {
					h.AssertSliceNotContains(t, configProvider.ContainerConfig().Cmd, "-uid")
				})
			})
		})

		when("-previous-image is used and builder is trusted", func() {
			when("image is invalid", func() {
				it("errors", func() {
					imageName, err := name.NewTag("/x/y/?!z", name.WeakValidation)
					h.AssertError(t, err, "repository can only contain the characters `abcdefghijklmnopqrstuvwxyz0123456789_-./`")

					lifecycleOps := append(lifecycleOps, func(options *build.LifecycleOptions) {
						options.Image = imageName
						options.PreviousImage = "some-previous-image"
					})
					lifecycle := newTestLifecycleExec(t, true, "some-temp-dir", lifecycleOps...)

					err = lifecycle.Create(context.Background(), fakeBuildCache, fakeLaunchCache, fakePhaseFactory)
					h.AssertError(t, err, "invalid image name")
				})
			})

			when("previous-image is invalid", func() {
				it("errors", func() {
					imageName, err := name.NewTag("/some/image", name.WeakValidation)
					h.AssertNil(t, err)

					lifecycleOps := append(lifecycleOps, func(options *build.LifecycleOptions) {
						options.PreviousImage = "%%%"
						options.Image = imageName
					})
					lifecycle := newTestLifecycleExec(t, true, "some-temp-dir", lifecycleOps...)

					err = lifecycle.Create(context.Background(), fakeBuildCache, fakeLaunchCache, fakePhaseFactory)
					h.AssertError(t, err, "invalid previous image name")
				})
			})

			when("--publish is false", func() {
				imageName, _ := name.NewTag("/some/image", name.WeakValidation)

				lifecycleOps = append(lifecycleOps, func(options *build.LifecycleOptions) {
					options.PreviousImage = "some-previous-image"
					options.Image = imageName
				})

				it("successfully passes previous-image to creator", func() {
					h.AssertIncludeAllExpectedPatterns(t, configProvider.ContainerConfig().Cmd, []string{"-previous-image", "some-previous-image"})
				})
			})

			when("--publish is true", func() {
				providedPublish = true

				when("previous-image and image are in the same registry", func() {
					imageName, _ := name.NewTag("/some/image", name.WeakValidation)

					lifecycleOps = append(lifecycleOps, func(options *build.LifecycleOptions) {
						options.PreviousImage = "index.docker.io/some/previous:latest"
						options.Image = imageName
					})

					it("successfully passes previous-image to creator", func() {
						h.AssertIncludeAllExpectedPatterns(t, configProvider.ContainerConfig().Cmd, []string{"-previous-image", "index.docker.io/some/previous:latest"})
					})
				})

				when("previous-image and image are not in the same registry", func() {
					it("errors", func() {
						imageName, err := name.NewTag("/some/image", name.WeakValidation)
						h.AssertNil(t, err)

						lifecycleOps := append(lifecycleOps, func(options *build.LifecycleOptions) {
							options.PreviousImage = "example.io/some/previous:latest"
							options.Image = imageName
						})
						lifecycle := newTestLifecycleExec(t, true, "some-temp-dir", lifecycleOps...)

						err = lifecycle.Create(context.Background(), fakeBuildCache, fakeLaunchCache, fakePhaseFactory)
						h.AssertError(t, err, fmt.Sprintf("%s", err))
					})
				})
			})
		})

		when("interactive mode", func() {
			lifecycleOps = append(lifecycleOps, func(opts *build.LifecycleOptions) {
				opts.Interactive = true
				opts.Termui = &fakes.FakeTermui{ReadLayersFunc: func(_ io.ReadCloser) {
					// no-op
				}}
			})

			it("provides the termui readLayersFunc as a post container operation", func() {
				h.AssertEq(t, fakePhase.CleanupCallCount, 1)
				h.AssertEq(t, fakePhase.RunCallCount, 1)

				h.AssertEq(t, len(configProvider.PostContainerRunOps()), 2)
				h.AssertFunctionName(t, configProvider.PostContainerRunOps()[0], "EnsureVolumeAccess")
				h.AssertFunctionName(t, configProvider.PostContainerRunOps()[1], "CopyOut")
			})
		})

		when("sbom destination directory is provided", func() {
			lifecycleOps = append(lifecycleOps, func(opts *build.LifecycleOptions) {
				opts.SBOMDestinationDir = "some-destination-dir"
			})

			it("provides copy-sbom-func as a post container operation", func() {
				h.AssertEq(t, fakePhase.CleanupCallCount, 1)
				h.AssertEq(t, fakePhase.RunCallCount, 1)

				h.AssertEq(t, len(configProvider.PostContainerRunOps()), 2)
				h.AssertFunctionName(t, configProvider.PostContainerRunOps()[0], "EnsureVolumeAccess")
				h.AssertFunctionName(t, configProvider.PostContainerRunOps()[1], "CopyOut")
			})
		})

		when("report destination directory is provided", func() {
			lifecycleOps = append(lifecycleOps, func(opts *build.LifecycleOptions) {
				opts.ReportDestinationDir = "a-destination-dir"
			})

			it("provides copy-sbom-func as a post container operation", func() {
				h.AssertEq(t, fakePhase.CleanupCallCount, 1)
				h.AssertEq(t, fakePhase.RunCallCount, 1)

				h.AssertEq(t, len(configProvider.PostContainerRunOps()), 2)
				h.AssertFunctionName(t, configProvider.PostContainerRunOps()[0], "EnsureVolumeAccess")
				h.AssertFunctionName(t, configProvider.PostContainerRunOps()[1], "CopyOut")
			})
		})

		when("--creation-time", func() {
			when("platform < 0.9", func() {
				platformAPI = api.MustParse("0.8")

				intTime, _ := strconv.ParseInt("1234567890", 10, 64)
				providedTime := time.Unix(intTime, 0).UTC()

				lifecycleOps = append(lifecycleOps, func(baseOpts *build.LifecycleOptions) {
					baseOpts.CreationTime = &providedTime
				})

				it("is ignored", func() {
					h.AssertSliceNotContains(t, configProvider.ContainerConfig().Env, "SOURCE_DATE_EPOCH=1234567890")
				})
			})

			when("platform >= 0.9", func() {
				platformAPI = api.MustParse("0.9")

				when("provided", func() {
					intTime, _ := strconv.ParseInt("1234567890", 10, 64)
					providedTime := time.Unix(intTime, 0).UTC()

					lifecycleOps = append(lifecycleOps, func(baseOpts *build.LifecycleOptions) {
						baseOpts.CreationTime = &providedTime
					})

					it("configures the phase with env SOURCE_DATE_EPOCH", func() {
						h.AssertSliceContains(t, configProvider.ContainerConfig().Env, "SOURCE_DATE_EPOCH=1234567890")
					})
				})

				when("not provided", func() {
					lifecycleOps = append(lifecycleOps, func(baseOpts *build.LifecycleOptions) {
						baseOpts.CreationTime = nil
					})

					it("does not panic", func() {
						// no-op
					})
				})
			})
		})

		when("layout", func() {
			providedLayout = true
			layoutRepo := filepath.Join(paths.RootDir, "layout-repo")
			platformAPI = api.MustParse("0.12")

			it("configures the phase with oci layout environment variables", func() {
				h.AssertSliceContains(t, configProvider.ContainerConfig().Env, "CNB_USE_LAYOUT=true")
				h.AssertSliceContains(t, configProvider.ContainerConfig().Env, fmt.Sprintf("CNB_LAYOUT_DIR=%s", layoutRepo))
				h.AssertSliceContains(t, configProvider.ContainerConfig().Env, "CNB_EXPERIMENTAL_MODE=warn")
			})
		})
	})

	when("#Detect", func() {
		it.Before(func() {
			err := lifecycle.Detect(context.Background(), fakePhaseFactory)
			h.AssertNil(t, err)

			lastCallIndex := len(fakePhaseFactory.NewCalledWithProvider) - 1
			h.AssertNotEq(t, lastCallIndex, -1)

			configProvider = fakePhaseFactory.NewCalledWithProvider[lastCallIndex]
			h.AssertEq(t, configProvider.Name(), "detector")
		})

		it("creates a phase and then runs it", func() {
			h.AssertEq(t, fakePhase.CleanupCallCount, 1)
			h.AssertEq(t, fakePhase.RunCallCount, 1)
		})

		it("configures the phase with the expected arguments", func() {
			h.AssertIncludeAllExpectedPatterns(t,
				configProvider.ContainerConfig().Cmd,
				[]string{"-log-level", "debug"},
			)
		})

		it("configures the phase with the expected network mode", func() {
			h.AssertEq(t, configProvider.HostConfig().NetworkMode, container.NetworkMode(providedNetworkMode))
		})

		it("configures the phase to copy app dir", func() {
			h.AssertSliceContains(t, configProvider.HostConfig().Binds, providedVolumes...)
			h.AssertEq(t, len(configProvider.ContainerOps()), 2)
			h.AssertFunctionName(t, configProvider.ContainerOps()[0], "EnsureVolumeAccess")
			h.AssertFunctionName(t, configProvider.ContainerOps()[1], "CopyDir")
		})

		when("extensions", func() {
			platformAPI = api.MustParse("0.10")

			when("present in the order", func() {
				providedOrderExt = dist.Order{dist.OrderEntry{Group: []dist.ModuleRef{ /* don't care */ }}}

				it("sets CNB_EXPERIMENTAL_MODE=warn in the environment", func() {
					h.AssertSliceContains(t, configProvider.ContainerConfig().Env, "CNB_EXPERIMENTAL_MODE=warn")
				})
			})

			when("not present in the order", func() {
				it("sets CNB_EXPERIMENTAL_MODE=warn in the environment", func() {
					h.AssertSliceNotContains(t, configProvider.ContainerConfig().Env, "CNB_EXPERIMENTAL_MODE=warn")
				})
			})
		})
	})

	when("#Analyze", func() {
		it.Before(func() {
			err := lifecycle.Analyze(context.Background(), fakeBuildCache, fakeLaunchCache, fakePhaseFactory)
			h.AssertNil(t, err)

			lastCallIndex := len(fakePhaseFactory.NewCalledWithProvider) - 1
			h.AssertNotEq(t, lastCallIndex, -1)

			configProvider = fakePhaseFactory.NewCalledWithProvider[lastCallIndex]
			h.AssertEq(t, configProvider.Name(), "analyzer")
		})

		it("creates a phase and then runs it", func() {
			h.AssertEq(t, fakePhase.CleanupCallCount, 1)
			h.AssertEq(t, fakePhase.RunCallCount, 1)
		})

		when("platform < 0.7", func() {
			when("clear cache", func() {
				providedClearCache = true

				it("configures the phase with the expected arguments", func() {
					h.AssertSliceContains(t, configProvider.ContainerConfig().Cmd, "-skip-layers")
				})
			})

			when("clear cache is false", func() {
				it("configures the phase with the expected arguments", func() {
					h.AssertIncludeAllExpectedPatterns(t,
						configProvider.ContainerConfig().Cmd,
						[]string{"-cache-dir", "/cache"},
					)
				})
			})

			when("using a cache image", func() {
				fakeBuildCache = newFakeImageCache()

				it("configures the phase with a build cache image", func() {
					h.AssertSliceNotContains(t, configProvider.HostConfig().Binds, ":/cache")
					h.AssertIncludeAllExpectedPatterns(t,
						configProvider.ContainerConfig().Cmd,
						[]string{"-cache-image", "some-cache-image"},
					)
					h.AssertSliceNotContains(t,
						configProvider.ContainerConfig().Cmd,
						"-cache-dir",
					)
				})

				when("clear-cache", func() {
					providedClearCache = true

					it("cache is omitted from Analyze", func() {
						h.AssertSliceNotContains(t, configProvider.ContainerConfig().Cmd, "-cache-image")
					})
				})
			})
		})

		when("platform >= 0.7", func() {
			platformAPI = api.MustParse("0.7")

			it("doesn't set cache dir", func() {
				h.AssertSliceNotContains(t, configProvider.HostConfig().Binds, ":/cache")
			})

			it("passes additional tags", func() {
				h.AssertIncludeAllExpectedPatterns(t,
					configProvider.ContainerConfig().Cmd,
					[]string{"-tag", "some-additional-tag2", "-tag", "some-additional-tag1"},
				)
			})

			it("passes run image", func() {
				h.AssertIncludeAllExpectedPatterns(t,
					configProvider.ContainerConfig().Cmd,
					[]string{"-run-image", "some-run-image"},
				)
			})

			it("passes stack", func() {
				h.AssertIncludeAllExpectedPatterns(t,
					configProvider.ContainerConfig().Cmd,
					[]string{"-stack", "/layers/stack.toml"},
				)
			})

			when("previous image", func() {
				lifecycleOps = append(lifecycleOps, func(options *build.LifecycleOptions) {
					options.PreviousImage = "some-previous-image"
				})

				it("passes previous image", func() {
					h.AssertIncludeAllExpectedPatterns(t,
						configProvider.ContainerConfig().Cmd,
						[]string{"-previous-image", "some-previous-image"},
					)
				})
			})
		})

		when("platform >= 0.12", func() {
			platformAPI = api.MustParse("0.12")

			it("passes run", func() {
				h.AssertIncludeAllExpectedPatterns(t,
					configProvider.ContainerConfig().Cmd,
					[]string{"-run", "/layers/run.toml"},
				)
				h.AssertSliceNotContains(t,
					configProvider.ContainerConfig().Cmd,
					"-stack",
				)
			})

			when("layout is true", func() {
				providedLayout = true

				it("configures the phase with the expected environment variables", func() {
					layoutDir := filepath.Join(paths.RootDir, "layout-repo")
					h.AssertSliceContains(t,
						configProvider.ContainerConfig().Env, "CNB_USE_LAYOUT=true", fmt.Sprintf("CNB_LAYOUT_DIR=%s", layoutDir),
					)
				})
			})
		})

		when("publish", func() {
			providedPublish = true

			when("lifecycle image", func() {
				lifecycleOps = append(lifecycleOps, func(options *build.LifecycleOptions) {
					options.LifecycleImage = "some-lifecycle-image"
				})

				it("runs the phase with the lifecycle image", func() {
					h.AssertEq(t, configProvider.ContainerConfig().Image, "some-lifecycle-image")
				})
			})

			it("sets the CNB_USER_ID and CNB_GROUP_ID in the environment", func() {
				h.AssertSliceContains(t, configProvider.ContainerConfig().Env, fmt.Sprintf("CNB_USER_ID=%d", providedUID))
				h.AssertSliceContains(t, configProvider.ContainerConfig().Env, fmt.Sprintf("CNB_GROUP_ID=%d", providedGID))
			})

			it("configures the phase with registry access", func() {
				h.AssertSliceContains(t, configProvider.ContainerConfig().Env, "CNB_REGISTRY_AUTH={}")
				h.AssertEq(t, configProvider.HostConfig().NetworkMode, container.NetworkMode(providedNetworkMode))
			})

			it("configures the phase with root", func() {
				h.AssertEq(t, configProvider.ContainerConfig().User, "root")
			})

			it("configures the phase with the expected arguments", func() {
				h.AssertIncludeAllExpectedPatterns(t,
					configProvider.ContainerConfig().Cmd,
					[]string{"-log-level", "debug"},
					[]string{providedTargetImage},
				)
			})

			it("configures the phase with binds", func() {
				expectedBind := "some-cache:/cache"

				h.AssertSliceContains(t, configProvider.HostConfig().Binds, expectedBind)
			})

			when("using a cache image", func() {
				fakeBuildCache = newFakeImageCache()

				it("configures the phase with a build cache images", func() {
					h.AssertSliceNotContains(t, configProvider.HostConfig().Binds, ":/cache")
					h.AssertIncludeAllExpectedPatterns(t,
						configProvider.ContainerConfig().Cmd,
						[]string{"-cache-image", "some-cache-image"},
					)
					h.AssertSliceNotContains(t,
						configProvider.ContainerConfig().Cmd,
						"-cache-dir",
					)
				})
			})
		})

		when("publish is false", func() {
			when("lifecycle image", func() {
				lifecycleOps = append(lifecycleOps, func(options *build.LifecycleOptions) {
					options.LifecycleImage = "some-lifecycle-image"
				})

				it("runs the phase with the lifecycle image", func() {
					h.AssertEq(t, configProvider.ContainerConfig().Image, "some-lifecycle-image")
				})
			})

			it("sets the CNB_USER_ID and CNB_GROUP_ID in the environment", func() {
				h.AssertSliceContains(t, configProvider.ContainerConfig().Env, fmt.Sprintf("CNB_USER_ID=%d", providedUID))
				h.AssertSliceContains(t, configProvider.ContainerConfig().Env, fmt.Sprintf("CNB_GROUP_ID=%d", providedGID))
			})

			it("configures the phase with daemon access", func() {
				h.AssertEq(t, configProvider.ContainerConfig().User, "root")
				h.AssertSliceContains(t, configProvider.HostConfig().Binds, "/var/run/docker.sock:/var/run/docker.sock")
			})

			when("tcp docker-host", func() {
				providedDockerHost = `tcp://localhost:1234`

				it("configures the phase with daemon access with TCP docker-host", func() {
					h.AssertSliceNotContains(t, configProvider.HostConfig().Binds, "/var/run/docker.sock:/var/run/docker.sock")
					h.AssertSliceContains(t, configProvider.ContainerConfig().Env, "DOCKER_HOST=tcp://localhost:1234")
				})
			})

			it("configures the phase with the expected arguments", func() {
				h.AssertIncludeAllExpectedPatterns(t,
					configProvider.ContainerConfig().Cmd,
					[]string{"-log-level", "debug"},
					[]string{"-daemon"},
					[]string{providedTargetImage},
				)
			})

			it("configures the phase with the expected network mode", func() {
				h.AssertEq(t, configProvider.HostConfig().NetworkMode, container.NetworkMode(providedNetworkMode))
			})

			it("configures the phase with binds", func() {
				expectedBind := "some-cache:/cache"

				h.AssertSliceContains(t, configProvider.HostConfig().Binds, expectedBind)
			})

			when("platform >= 0.9", func() {
				platformAPI = api.MustParse("0.9")

				providedClearCache = true

				it("configures the phase with launch cache and skip layers", func() {
					expectedBinds := []string{"some-launch-cache:/launch-cache"}

					h.AssertIncludeAllExpectedPatterns(t,
						configProvider.ContainerConfig().Cmd,
						[]string{"-skip-layers"},
						[]string{"-launch-cache", "/launch-cache"},
					)
					h.AssertSliceContains(t, configProvider.HostConfig().Binds, expectedBinds...)
				})

				when("override GID", func() {
					when("override GID is provided", func() {
						lifecycleOps = append(lifecycleOps, func(options *build.LifecycleOptions) {
							options.GID = 2
						})

						it("configures the phase with the expected arguments", func() {
							h.AssertIncludeAllExpectedPatterns(t,
								configProvider.ContainerConfig().Cmd,
								[]string{"-gid", "2"},
							)
						})
					})

					when("override GID is not provided", func() {
						lifecycleOps = append(lifecycleOps, func(options *build.LifecycleOptions) {
							options.GID = -1
						})

						it("gid is not added to the expected arguments", func() {
							h.AssertSliceNotContains(t, configProvider.ContainerConfig().Cmd, "-gid")
						})
					})
				})

				when("override UID", func() {
					when("override UID is provided", func() {
						lifecycleOps = append(lifecycleOps, func(options *build.LifecycleOptions) {
							options.UID = 1001
						})

						it("configures the phase with the expected arguments", func() {
							h.AssertIncludeAllExpectedPatterns(t,
								configProvider.ContainerConfig().Cmd,
								[]string{"-uid", "1001"},
							)
						})
					})

					when("override UID is not provided", func() {
						lifecycleOps = append(lifecycleOps, func(options *build.LifecycleOptions) {
							options.UID = -1
						})

						it("uid is not added to the expected arguments", func() {
							h.AssertSliceNotContains(t, configProvider.ContainerConfig().Cmd, "-uid")
						})
					})
				})
			})
		})

		when("previous-image is used and builder is untrusted", func() {
			when("image is invalid", func() {
				it("errors", func() {
					var imageName name.Tag
					imageName, err := name.NewTag("/x/y/?!z", name.WeakValidation)
					h.AssertError(t, err, "repository can only contain the characters `abcdefghijklmnopqrstuvwxyz0123456789_-./`")

					lifecycleOps := append(lifecycleOps, func(options *build.LifecycleOptions) {
						options.Image = imageName
						options.PreviousImage = "some-previous-image"
					})
					lifecycle := newTestLifecycleExec(t, true, "some-temp-dir", lifecycleOps...)

					err = lifecycle.Analyze(context.Background(), fakeBuildCache, fakeLaunchCache, fakePhaseFactory)
					h.AssertError(t, err, "invalid image name")
				})
			})

			when("previous-image is invalid", func() {
				it("errors", func() {
					var imageName name.Tag
					imageName, err := name.NewTag("/some/image", name.WeakValidation)
					h.AssertNil(t, err)

					lifecycleOps := append(lifecycleOps, func(options *build.LifecycleOptions) {
						options.PreviousImage = "%%%"
						options.Image = imageName
					})
					lifecycle := newTestLifecycleExec(t, true, "some-temp-dir", lifecycleOps...)

					err = lifecycle.Analyze(context.Background(), fakeBuildCache, fakeLaunchCache, fakePhaseFactory)
					h.AssertError(t, err, "invalid previous image name")
				})
			})

			when("--publish is false", func() {
				when("previous image", func() {
					imageName, _ := name.NewTag("/some/image", name.WeakValidation)

					lifecycleOps = append(lifecycleOps, func(options *build.LifecycleOptions) {
						options.PreviousImage = "previous-image"
						options.Image = imageName
					})

					it("successfully passes previous-image to analyzer", func() {
						prevImage, err := name.ParseReference(lifecycle.PrevImageName(), name.WeakValidation)
						h.AssertNil(t, err)
						h.AssertEq(t, lifecycle.ImageName().Name(), prevImage.Name())
					})
				})
			})

			when("--publish is true", func() {
				providedPublish = true

				when("previous-image and image are in the same registry", func() {
					imageName, _ := name.NewTag("/some/image", name.WeakValidation)

					lifecycleOps = append(lifecycleOps, func(options *build.LifecycleOptions) {
						options.PreviousImage = "index.docker.io/some/previous:latest"
						options.Image = imageName
					})

					it("successfully passes previous-image to analyzer", func() {
						prevImage, err := name.ParseReference(lifecycle.PrevImageName(), name.WeakValidation)
						h.AssertNil(t, err)
						h.AssertEq(t, lifecycle.ImageName().Name(), prevImage.Name())
					})
				})

				when("previous-image and image are not in the same registry", func() {
					it("errors", func() {
						imageName, err := name.NewTag("/some/image", name.WeakValidation)
						h.AssertNil(t, err)

						lifecycleOps := append(lifecycleOps, func(options *build.LifecycleOptions) {
							options.PreviousImage = "example.io/some/previous:latest"
							options.Image = imageName
						})
						lifecycle := newTestLifecycleExec(t, true, "some-temp-dir", lifecycleOps...)

						err = lifecycle.Analyze(context.Background(), fakeBuildCache, fakeLaunchCache, fakePhaseFactory)
						h.AssertNotNil(t, err)
					})
				})
			})
		})
	})

	when("#Restore", func() {
		it.Before(func() {
			err := lifecycle.Restore(context.Background(), fakeBuildCache, fakeKanikoCache, fakePhaseFactory)
			h.AssertNil(t, err)

			lastCallIndex := len(fakePhaseFactory.NewCalledWithProvider) - 1
			h.AssertNotEq(t, lastCallIndex, -1)

			configProvider = fakePhaseFactory.NewCalledWithProvider[lastCallIndex]
			h.AssertEq(t, configProvider.Name(), "restorer")
		})

		when("lifecycle image", func() {
			lifecycleOps = append(lifecycleOps, func(options *build.LifecycleOptions) {
				options.LifecycleImage = "some-lifecycle-image"
			})

			it("runs the phase with the lifecycle image", func() {
				h.AssertEq(t, configProvider.ContainerConfig().Image, "some-lifecycle-image")
			})
		})

		it("sets the CNB_USER_ID and CNB_GROUP_ID in the environment", func() {
			h.AssertSliceContains(t, configProvider.ContainerConfig().Env, fmt.Sprintf("CNB_USER_ID=%d", providedUID))
			h.AssertSliceContains(t, configProvider.ContainerConfig().Env, fmt.Sprintf("CNB_GROUP_ID=%d", providedGID))
		})

		it("creates a phase and then runs it", func() {
			h.AssertEq(t, fakePhase.CleanupCallCount, 1)
			h.AssertEq(t, fakePhase.RunCallCount, 1)
		})

		it("configures the phase with root access", func() {
			h.AssertEq(t, configProvider.ContainerConfig().User, "root")
		})

		it("configures the phase with the expected arguments", func() {
			h.AssertIncludeAllExpectedPatterns(t,
				configProvider.ContainerConfig().Cmd,
				[]string{"-log-level", "debug"},
				[]string{"-cache-dir", "/cache"},
			)
		})

		it("configures the phase with the expected network mode", func() {
			h.AssertEq(t, configProvider.HostConfig().NetworkMode, container.NetworkMode(providedNetworkMode))
		})

		it("configures the phase with binds", func() {
			expectedBind := "some-cache:/cache"

			h.AssertSliceContains(t, configProvider.HostConfig().Binds, expectedBind)
		})

		when("there are extensions", func() {
			platformAPI = api.MustParse("0.12")
			providedOrderExt = dist.Order{dist.OrderEntry{Group: []dist.ModuleRef{ /* don't care */ }}}

			when("for build", func() {
				extensionsForBuild = true

				it("configures the phase with registry access", func() {
					h.AssertSliceContains(t, configProvider.ContainerConfig().Env, "CNB_REGISTRY_AUTH={}")
				})
			})

			when("for run", func() {
				extensionsForRun = true

				it("configures the phase with registry access", func() {
					h.AssertSliceContains(t, configProvider.ContainerConfig().Env, "CNB_REGISTRY_AUTH={}")
				})
			})
		})

		when("using cache image", func() {
			fakeBuildCache = newFakeImageCache()

			it("configures the phase with a cache image", func() {
				h.AssertSliceNotContains(t, configProvider.HostConfig().Binds, ":/cache")
				h.AssertIncludeAllExpectedPatterns(t,
					configProvider.ContainerConfig().Cmd,
					[]string{"-cache-image", "some-cache-image"},
				)
				h.AssertSliceNotContains(t, configProvider.ContainerConfig().Cmd, "-cache-dir")
			})

			it("configures the phase with registry access", func() {
				h.AssertSliceContains(t, configProvider.ContainerConfig().Env, "CNB_REGISTRY_AUTH={}")
			})
		})

		when("override GID", func() {
			when("override GID is provided", func() {
				lifecycleOps = append(lifecycleOps, func(options *build.LifecycleOptions) {
					options.GID = 2
				})

				it("configures the phase with the expected arguments", func() {
					h.AssertIncludeAllExpectedPatterns(t,
						configProvider.ContainerConfig().Cmd,
						[]string{"-gid", "2"},
					)
				})
			})

			when("override GID is not provided", func() {
				lifecycleOps = append(lifecycleOps, func(options *build.LifecycleOptions) {
					options.GID = -1
				})

				it("gid is not added to the expected arguments", func() {
					h.AssertSliceNotContains(t, configProvider.ContainerConfig().Cmd, "-gid")
				})
			})
		})

		when("override UID", func() {
			when("override UID is provided", func() {
				lifecycleOps = append(lifecycleOps, func(options *build.LifecycleOptions) {
					options.UID = 1001
				})

				it("configures the phase with the expected arguments", func() {
					h.AssertIncludeAllExpectedPatterns(t,
						configProvider.ContainerConfig().Cmd,
						[]string{"-uid", "1001"},
					)
				})
			})

			when("override UID is not provided", func() {
				lifecycleOps = append(lifecycleOps, func(options *build.LifecycleOptions) {
					options.UID = -1
				})

				it("uid is not added to the expected arguments", func() {
					h.AssertSliceNotContains(t, configProvider.ContainerConfig().Cmd, "-uid")
				})
			})
		})

		when("--clear-cache", func() {
			providedClearCache = true

			it("provides -skip-layers", func() {
				h.AssertSliceContains(t, configProvider.ContainerConfig().Cmd, "-skip-layers")
			})
		})

		when("extensions", func() {
			providedOrderExt = dist.Order{dist.OrderEntry{Group: []dist.ModuleRef{ /* don't care */ }}}

			when("for build", func() {
				when("present in <layers>/generated/build", func() {
					extensionsForBuild = true

					when("platform < 0.10", func() {
						platformAPI = api.MustParse("0.9")

						it("does not provide -build-image or /kaniko bind", func() {
							h.AssertSliceNotContains(t, configProvider.ContainerConfig().Cmd, "-build-image")
							h.AssertSliceNotContains(t, configProvider.HostConfig().Binds, "some-kaniko-cache:/kaniko")
						})
					})

					when("platform >= 0.10", func() {
						platformAPI = api.MustParse("0.10")

						it("provides -build-image and /kaniko bind", func() {
							h.AssertSliceContainsInOrder(t, configProvider.ContainerConfig().Cmd, "-build-image", providedBuilderImage)
							h.AssertSliceContains(t, configProvider.ContainerConfig().Env, "CNB_REGISTRY_AUTH={}")
							h.AssertSliceContains(t, configProvider.HostConfig().Binds, "some-kaniko-cache:/kaniko")
						})
					})
				})

				when("not present in <layers>/generated/build", func() {
					platformAPI = api.MustParse("0.10")

					it("does not provide -build-image or /kaniko bind", func() {
						h.AssertSliceNotContains(t, configProvider.ContainerConfig().Cmd, "-build-image")
						h.AssertSliceNotContains(t, configProvider.HostConfig().Binds, "some-kaniko-cache:/kaniko")
					})
				})
			})

			when("for run", func() {
				when("analyzed.toml extend", func() {
					when("true", func() {
						extensionsForRun = true

						when("platform >= 0.12", func() {
							platformAPI = api.MustParse("0.12")

							it("provides /kaniko bind", func() {
								h.AssertSliceContains(t, configProvider.HostConfig().Binds, "some-kaniko-cache:/kaniko")
							})
						})

						when("platform < 0.12", func() {
							platformAPI = api.MustParse("0.11")

							it("does not provide /kaniko bind", func() {
								h.AssertSliceNotContains(t, configProvider.HostConfig().Binds, "some-kaniko-cache:/kaniko")
							})
						})
					})

					when("false", func() {
						platformAPI = api.MustParse("0.12")

						it("does not provide /kaniko bind", func() {
							h.AssertSliceNotContains(t, configProvider.HostConfig().Binds, "some-kaniko-cache:/kaniko")
						})
					})
				})
			})
		})

		when("publish is false", func() {
			when("platform >= 0.12", func() {
				platformAPI = api.MustParse("0.12")

				it("configures the phase with daemon access", func() {
					h.AssertEq(t, configProvider.ContainerConfig().User, "root")
					h.AssertSliceContains(t, configProvider.HostConfig().Binds, "/var/run/docker.sock:/var/run/docker.sock")
				})

				it("configures the phase with the expected arguments", func() {
					h.AssertIncludeAllExpectedPatterns(t,
						configProvider.ContainerConfig().Cmd,
						[]string{"-daemon"},
					)
				})
			})
		})

		when("layout is true", func() {
			when("platform >= 0.12", func() {
				platformAPI = api.MustParse("0.12")
				providedLayout = true

				it("it configures the phase with access to provided volumes", func() {
					// this is required to read the /layout-repo
					h.AssertSliceContains(t, configProvider.HostConfig().Binds, providedVolumes...)
				})

				it("configures the phase with the expected environment variables", func() {
					layoutDir := filepath.Join(paths.RootDir, "layout-repo")
					h.AssertSliceContains(t,
						configProvider.ContainerConfig().Env, "CNB_USE_LAYOUT=true", fmt.Sprintf("CNB_LAYOUT_DIR=%s", layoutDir),
					)
				})
			})
		})
	})

	when("#Build", func() {
		it.Before(func() {
			err := lifecycle.Build(context.Background(), fakePhaseFactory)
			h.AssertNil(t, err)

			lastCallIndex := len(fakePhaseFactory.NewCalledWithProvider) - 1
			h.AssertNotEq(t, lastCallIndex, -1)

			configProvider = fakePhaseFactory.NewCalledWithProvider[lastCallIndex]
			h.AssertEq(t, configProvider.Name(), "builder")
		})

		it("creates a phase and then runs it", func() {
			h.AssertEq(t, fakePhase.CleanupCallCount, 1)
			h.AssertEq(t, fakePhase.RunCallCount, 1)
		})

		it("configures the phase with the expected arguments", func() {
			h.AssertIncludeAllExpectedPatterns(t,
				configProvider.ContainerConfig().Cmd,
				[]string{"-log-level", "debug"},
			)
		})

		it("configures the phase with the expected network mode", func() {
			h.AssertEq(t, configProvider.HostConfig().NetworkMode, container.NetworkMode(providedNetworkMode))
		})

		it("configures the phase with binds", func() {
			h.AssertSliceContains(t, configProvider.HostConfig().Binds, providedVolumes...)
		})
	})

	when("#ExtendBuild", func() {
		var experimental bool
		it.Before(func() {
			experimental = true
			err := lifecycle.ExtendBuild(context.Background(), fakeKanikoCache, fakePhaseFactory, experimental)
			h.AssertNil(t, err)

			lastCallIndex := len(fakePhaseFactory.NewCalledWithProvider) - 1
			h.AssertNotEq(t, lastCallIndex, -1)

			configProvider = fakePhaseFactory.NewCalledWithProvider[lastCallIndex]
			h.AssertEq(t, configProvider.Name(), "extender")
		})

		it("creates a phase and then runs it", func() {
			h.AssertEq(t, fakePhase.CleanupCallCount, 1)
			h.AssertEq(t, fakePhase.RunCallCount, 1)
		})

		it("configures the phase with the expected arguments", func() {
			h.AssertSliceContainsInOrder(t, configProvider.ContainerConfig().Cmd, "-log-level", "debug")
			h.AssertSliceContainsInOrder(t, configProvider.ContainerConfig().Cmd, "-app", "/workspace")
		})

		it("configures the phase with binds", func() {
			expectedBinds := providedVolumes
			expectedBinds = append(expectedBinds, "some-kaniko-cache:/kaniko")

			h.AssertSliceContains(t, configProvider.HostConfig().Binds, expectedBinds...)
		})

		it("sets CNB_EXPERIMENTAL_MODE=warn in the environment", func() {
			h.AssertSliceContains(t, configProvider.ContainerConfig().Env, "CNB_EXPERIMENTAL_MODE=warn")
		})

		it("configures the phase with the expected network mode", func() {
			h.AssertEq(t, configProvider.HostConfig().NetworkMode, container.NetworkMode(providedNetworkMode))
		})

		it("configures the phase with root", func() {
			h.AssertEq(t, configProvider.ContainerConfig().User, "root")
		})

		when("experimental is false", func() {
			it.Before(func() {
				experimental = false
				err := lifecycle.ExtendBuild(context.Background(), fakeKanikoCache, fakePhaseFactory, experimental)
				h.AssertNil(t, err)

				lastCallIndex := len(fakePhaseFactory.NewCalledWithProvider) - 1
				h.AssertNotEq(t, lastCallIndex, -1)

				configProvider = fakePhaseFactory.NewCalledWithProvider[lastCallIndex]
				h.AssertEq(t, configProvider.Name(), "extender")
			})

			it("CNB_EXPERIMENTAL_MODE=warn is not enable in the environment", func() {
				h.AssertSliceNotContains(t, configProvider.ContainerConfig().Env, "CNB_EXPERIMENTAL_MODE=warn")
			})
		})
	})

	when("#ExtendRun", func() {
		var experimental bool
		it.Before(func() {
			experimental = true
			err := lifecycle.ExtendRun(context.Background(), fakeKanikoCache, fakePhaseFactory, "some-run-image", experimental)
			h.AssertNil(t, err)

			lastCallIndex := len(fakePhaseFactory.NewCalledWithProvider) - 1
			h.AssertNotEq(t, lastCallIndex, -1)

			configProvider = fakePhaseFactory.NewCalledWithProvider[lastCallIndex]
			h.AssertEq(t, configProvider.Name(), "extender")
		})

		it("creates a phase and then runs it", func() {
			h.AssertEq(t, fakePhase.CleanupCallCount, 1)
			h.AssertEq(t, fakePhase.RunCallCount, 1)
		})

		it("runs the phase with the run image", func() {
			h.AssertEq(t, configProvider.ContainerConfig().Image, "some-run-image")
		})

		it("configures the phase with the expected arguments", func() {
			h.AssertSliceContainsInOrder(t, configProvider.ContainerConfig().Entrypoint, "") // the run image may have an entrypoint configured, override it
			h.AssertSliceContainsInOrder(t, configProvider.ContainerConfig().Cmd, "-log-level", "debug")
			h.AssertSliceContainsInOrder(t, configProvider.ContainerConfig().Cmd, "-app", "/workspace")
			h.AssertSliceContainsInOrder(t, configProvider.ContainerConfig().Cmd, "-kind", "run")
		})

		it("configures the phase with binds", func() {
			expectedBinds := providedVolumes
			expectedBinds = append(expectedBinds, "some-kaniko-cache:/kaniko")

			h.AssertSliceContains(t, configProvider.HostConfig().Binds, expectedBinds...)
		})

		it("sets CNB_EXPERIMENTAL_MODE=warn in the environment", func() {
			h.AssertSliceContains(t, configProvider.ContainerConfig().Env, "CNB_EXPERIMENTAL_MODE=warn")
		})

		it("configures the phase with the expected network mode", func() {
			h.AssertEq(t, configProvider.HostConfig().NetworkMode, container.NetworkMode(providedNetworkMode))
		})

		it("configures the phase with root", func() {
			h.AssertEq(t, configProvider.ContainerConfig().User, "root")
		})

		when("experimental is false", func() {
			it.Before(func() {
				experimental = false
				err := lifecycle.ExtendRun(context.Background(), fakeKanikoCache, fakePhaseFactory, "some-run-image", experimental)
				h.AssertNil(t, err)

				lastCallIndex := len(fakePhaseFactory.NewCalledWithProvider) - 1
				h.AssertNotEq(t, lastCallIndex, -1)

				configProvider = fakePhaseFactory.NewCalledWithProvider[lastCallIndex]
				h.AssertEq(t, configProvider.Name(), "extender")
			})

			it("CNB_EXPERIMENTAL_MODE=warn is not enable in the environment", func() {
				h.AssertSliceNotContains(t, configProvider.ContainerConfig().Env, "CNB_EXPERIMENTAL_MODE=warn")
			})
		})
	})

	when("#Export", func() {
		it.Before(func() {
			err := lifecycle.Export(context.Background(), fakeBuildCache, fakeLaunchCache, fakeKanikoCache, fakePhaseFactory)
			h.AssertNil(t, err)

			lastCallIndex := len(fakePhaseFactory.NewCalledWithProvider) - 1
			h.AssertNotEq(t, lastCallIndex, -1)

			configProvider = fakePhaseFactory.NewCalledWithProvider[lastCallIndex]
			h.AssertEq(t, configProvider.Name(), "exporter")
		})

		it("creates a phase and then runs it", func() {
			h.AssertEq(t, fakePhase.CleanupCallCount, 1)
			h.AssertEq(t, fakePhase.RunCallCount, 1)
		})

		it("configures the phase with the expected arguments", func() {
			h.AssertIncludeAllExpectedPatterns(t,
				configProvider.ContainerConfig().Cmd,
				[]string{"-log-level", "debug"},
				[]string{"-cache-dir", "/cache"},
				[]string{"-run-image", providedRunImage},
				[]string{"-stack", "/layers/stack.toml"},
				[]string{providedTargetImage},
			)
			h.AssertSliceNotContains(t, configProvider.ContainerConfig().Cmd, "-run")
		})

		when("platform >= 0.12", func() {
			platformAPI = api.MustParse("0.12")

			it("provides -run instead of -stack", func() {
				h.AssertIncludeAllExpectedPatterns(t,
					configProvider.ContainerConfig().Cmd,
					[]string{"-run", "/layers/run.toml"},
				)
				h.AssertSliceNotContains(t, configProvider.ContainerConfig().Cmd, "-stack")
			})

			when("there are extensions", func() {
				providedOrderExt = dist.Order{dist.OrderEntry{Group: []dist.ModuleRef{ /* don't care */ }}}

				when("for run", func() {
					extensionsForRun = true

					it("sets CNB_EXPERIMENTAL_MODE=warn in the environment", func() {
						h.AssertSliceContains(t, configProvider.ContainerConfig().Env, "CNB_EXPERIMENTAL_MODE=warn")
					})

					it("configures the phase with binds", func() {
						expectedBinds := []string{"some-cache:/cache", "some-launch-cache:/launch-cache", "some-kaniko-cache:/kaniko"}

						h.AssertSliceContains(t, configProvider.HostConfig().Binds, expectedBinds...)
					})
				})
			})

			when("layout is true", func() {
				providedLayout = true

				it("it configures the phase with access to provided volumes", func() {
					// this is required to read the /layout-repo
					h.AssertSliceContains(t, configProvider.HostConfig().Binds, providedVolumes...)
				})

				it("configures the phase with the expected environment variables", func() {
					layoutDir := filepath.Join(paths.RootDir, "layout-repo")
					h.AssertSliceContains(t,
						configProvider.ContainerConfig().Env, "CNB_USE_LAYOUT=true", fmt.Sprintf("CNB_LAYOUT_DIR=%s", layoutDir),
					)
				})
			})
		})

		when("additional tags are specified", func() {
			it("passes tag arguments to the exporter", func() {
				h.AssertIncludeAllExpectedPatterns(t,
					configProvider.ContainerConfig().Cmd,
					[]string{"-log-level", "debug"},
					[]string{"-cache-dir", "/cache"},
					[]string{"-run-image", providedRunImage},
					[]string{providedTargetImage, providedAdditionalTags[0], providedAdditionalTags[1]},
				)
			})
		})

		when("platform >= 0.7", func() {
			platformAPI = api.MustParse("0.7")

			it("doesn't hint at default process type", func() {
				h.AssertIncludeAllExpectedPatterns(t,
					configProvider.ContainerConfig().Cmd,
					[]string{"-log-level", "debug"},
					[]string{"-cache-dir", "/cache"},
					[]string{providedTargetImage, providedAdditionalTags[0], providedAdditionalTags[1]},
				)
				h.AssertSliceNotContains(t, configProvider.ContainerConfig().Cmd, "-run-image")
			})
		})

		when("using cache image", func() {
			fakeBuildCache = newFakeImageCache()

			it("configures phase with cache image", func() {
				h.AssertSliceNotContains(t, configProvider.HostConfig().Binds, ":/cache")
				h.AssertIncludeAllExpectedPatterns(t,
					configProvider.ContainerConfig().Cmd,
					[]string{"-cache-image", "some-cache-image"},
				)
			})
		})

		when("publish", func() {
			providedPublish = true

			when("lifecycle image", func() {
				lifecycleOps = append(lifecycleOps, func(options *build.LifecycleOptions) {
					options.LifecycleImage = "some-lifecycle-image"
				})

				it("runs the phase with the lifecycle image", func() {
					h.AssertEq(t, configProvider.ContainerConfig().Image, "some-lifecycle-image")
				})
			})

			it("sets the CNB_USER_ID and CNB_GROUP_ID in the environment", func() {
				h.AssertSliceContains(t, configProvider.ContainerConfig().Env, fmt.Sprintf("CNB_USER_ID=%d", providedUID))
				h.AssertSliceContains(t, configProvider.ContainerConfig().Env, fmt.Sprintf("CNB_GROUP_ID=%d", providedGID))
			})

			it("configures the phase with registry access", func() {
				h.AssertSliceContains(t, configProvider.ContainerConfig().Env, "CNB_REGISTRY_AUTH={}")
			})

			it("configures the phase with root", func() {
				h.AssertEq(t, configProvider.ContainerConfig().User, "root")
			})

			it("configures the phase with the expected network mode", func() {
				h.AssertEq(t, configProvider.HostConfig().NetworkMode, container.NetworkMode(providedNetworkMode))
			})

			it("configures the phase with binds", func() {
				expectedBind := "some-cache:/cache"

				h.AssertSliceContains(t, configProvider.HostConfig().Binds, expectedBind)
			})

			it("configures the phase to write stack toml", func() {
				expectedBinds := []string{"some-cache:/cache"}
				h.AssertSliceContains(t, configProvider.HostConfig().Binds, expectedBinds...)

				h.AssertEq(t, len(configProvider.ContainerOps()), 3)
				h.AssertFunctionName(t, configProvider.ContainerOps()[0], "WriteStackToml")
				h.AssertFunctionName(t, configProvider.ContainerOps()[1], "WriteRunToml")
				h.AssertFunctionName(t, configProvider.ContainerOps()[2], "WriteProjectMetadata")
			})

			when("default process type", func() {
				lifecycleOps = append(lifecycleOps, func(options *build.LifecycleOptions) {
					options.DefaultProcessType = "test-process"
				})

				it("configures the phase with default process type", func() {
					expectedDefaultProc := []string{"-process-type", "test-process"}
					h.AssertIncludeAllExpectedPatterns(t, configProvider.ContainerConfig().Cmd, expectedDefaultProc)
				})
			})

			when("using cache image and publishing", func() {
				fakeBuildCache = newFakeImageCache()

				it("configures phase with cache image", func() {
					h.AssertSliceNotContains(t, configProvider.HostConfig().Binds, ":/cache")
					h.AssertIncludeAllExpectedPatterns(t,
						configProvider.ContainerConfig().Cmd,
						[]string{"-cache-image", "some-cache-image"},
					)
				})
			})

			when("platform 0.3", func() {
				platformAPI = api.MustParse("0.3")

				it("doesn't hint at default process type", func() {
					h.AssertSliceNotContains(t, configProvider.ContainerConfig().Cmd, "-process-type")
				})
			})

			when("platform 0.4", func() {
				platformAPI = api.MustParse("0.4")

				it("hints at default process type", func() {
					h.AssertIncludeAllExpectedPatterns(t, configProvider.ContainerConfig().Cmd, []string{"-process-type", "web"})
				})
			})

			when("platform >= 0.6", func() {
				platformAPI = api.MustParse("0.6")

				when("no user provided process type is present", func() {
					it("doesn't provide 'web' as default process type", func() {
						h.AssertSliceNotContains(t, configProvider.ContainerConfig().Cmd, "-process-type")
					})
				})
			})
		})

		when("publish is false", func() {
			when("lifecycle image", func() {
				lifecycleOps = append(lifecycleOps, func(options *build.LifecycleOptions) {
					options.LifecycleImage = "some-lifecycle-image"
				})

				it("runs the phase with the lifecycle image", func() {
					h.AssertEq(t, configProvider.ContainerConfig().Image, "some-lifecycle-image")
				})
			})

			it("sets the CNB_USER_ID and CNB_GROUP_ID in the environment", func() {
				h.AssertSliceContains(t, configProvider.ContainerConfig().Env, fmt.Sprintf("CNB_USER_ID=%d", providedUID))
				h.AssertSliceContains(t, configProvider.ContainerConfig().Env, fmt.Sprintf("CNB_GROUP_ID=%d", providedGID))
			})

			it("configures the phase with daemon access", func() {
				h.AssertEq(t, configProvider.ContainerConfig().User, "root")
				h.AssertSliceContains(t, configProvider.HostConfig().Binds, "/var/run/docker.sock:/var/run/docker.sock")
			})

			when("tcp docker-host", func() {
				providedDockerHost = `tcp://localhost:1234`

				it("configures the phase with daemon access with tcp docker-host", func() {
					h.AssertSliceNotContains(t, configProvider.HostConfig().Binds, "/var/run/docker.sock:/var/run/docker.sock")
					h.AssertSliceContains(t, configProvider.ContainerConfig().Env, "DOCKER_HOST=tcp://localhost:1234")
				})
			})

			it("configures the phase with the expected arguments", func() {
				h.AssertIncludeAllExpectedPatterns(t,
					configProvider.ContainerConfig().Cmd,
					[]string{"-daemon"},
					[]string{"-launch-cache", "/launch-cache"},
				)
			})

			it("configures the phase with the expected network mode", func() {
				h.AssertEq(t, configProvider.HostConfig().NetworkMode, container.NetworkMode(providedNetworkMode))
			})

			it("configures the phase with binds", func() {
				expectedBinds := []string{"some-cache:/cache", "some-launch-cache:/launch-cache"}

				h.AssertSliceContains(t, configProvider.HostConfig().Binds, expectedBinds...)
			})

			it("configures the phase to write stack toml", func() {
				expectedBinds := []string{"some-cache:/cache", "some-launch-cache:/launch-cache"}
				h.AssertSliceContains(t, configProvider.HostConfig().Binds, expectedBinds...)

				h.AssertEq(t, len(configProvider.ContainerOps()), 3)
				h.AssertFunctionName(t, configProvider.ContainerOps()[0], "WriteStackToml")
				h.AssertFunctionName(t, configProvider.ContainerOps()[1], "WriteRunToml")
				h.AssertFunctionName(t, configProvider.ContainerOps()[2], "WriteProjectMetadata")
			})

			when("default process type", func() {
				when("provided", func() {
					lifecycleOps = append(lifecycleOps, func(options *build.LifecycleOptions) {
						options.DefaultProcessType = "test-process"
					})

					it("configures the phase with default process type", func() {
						expectedDefaultProc := []string{"-process-type", "test-process"}
						h.AssertIncludeAllExpectedPatterns(t, configProvider.ContainerConfig().Cmd, expectedDefaultProc)
					})
				})

				when("platform 0.3", func() {
					platformAPI = api.MustParse("0.3")

					it("doesn't hint at default process type", func() {
						h.AssertSliceNotContains(t, configProvider.ContainerConfig().Cmd, "-process-type")
					})
				})

				when("platform 0.4", func() {
					platformAPI = api.MustParse("0.4")

					it("hints at default process type", func() {
						h.AssertIncludeAllExpectedPatterns(t, configProvider.ContainerConfig().Cmd, []string{"-process-type", "web"})
					})
				})

				when("platform >= 0.6", func() {
					platformAPI = api.MustParse("0.6")

					when("no user provided process type is present", func() {
						it("doesn't provide 'web' as default process type", func() {
							h.AssertSliceNotContains(t, configProvider.ContainerConfig().Cmd, "-process-type")
						})
					})
				})
			})
		})

		when("override GID", func() {
			when("override GID is provided", func() {
				lifecycleOps = append(lifecycleOps, func(options *build.LifecycleOptions) {
					options.GID = 2
				})

				it("configures the phase with the expected arguments", func() {
					h.AssertIncludeAllExpectedPatterns(t,
						configProvider.ContainerConfig().Cmd,
						[]string{"-gid", "2"},
					)
				})
			})

			when("override GID is not provided", func() {
				lifecycleOps = append(lifecycleOps, func(options *build.LifecycleOptions) {
					options.GID = -1
				})

				it("gid is not added to the expected arguments", func() {
					h.AssertSliceNotContains(t, configProvider.ContainerConfig().Cmd, "-gid")
				})
			})
		})

		when("override UID", func() {
			when("override UID is provided", func() {
				lifecycleOps = append(lifecycleOps, func(options *build.LifecycleOptions) {
					options.UID = 1001
				})

				it("configures the phase with the expected arguments", func() {
					h.AssertIncludeAllExpectedPatterns(t,
						configProvider.ContainerConfig().Cmd,
						[]string{"-uid", "1001"},
					)
				})
			})

			when("override UID is not provided", func() {
				lifecycleOps = append(lifecycleOps, func(options *build.LifecycleOptions) {
					options.UID = -1
				})

				it("uid is not added to the expected arguments", func() {
					h.AssertSliceNotContains(t, configProvider.ContainerConfig().Cmd, "-uid")
				})
			})
		})

		when("interactive mode", func() {
			lifecycleOps = append(lifecycleOps, func(opts *build.LifecycleOptions) {
				opts.Interactive = true
				opts.Termui = &fakes.FakeTermui{ReadLayersFunc: func(_ io.ReadCloser) {
					// no-op
				}}
			})

			it("provides the termui readLayersFunc as a post container operation", func() {
				h.AssertEq(t, len(configProvider.PostContainerRunOps()), 2)
				h.AssertFunctionName(t, configProvider.PostContainerRunOps()[0], "EnsureVolumeAccess")
				h.AssertFunctionName(t, configProvider.PostContainerRunOps()[1], "CopyOut")
			})
		})

		when("sbom destination directory is provided", func() {
			lifecycleOps = append(lifecycleOps, func(opts *build.LifecycleOptions) {
				opts.SBOMDestinationDir = "some-destination-dir"
			})

			it("provides copy-sbom-func as a post container operation", func() {
				h.AssertEq(t, len(configProvider.PostContainerRunOps()), 2)
				h.AssertFunctionName(t, configProvider.PostContainerRunOps()[0], "EnsureVolumeAccess")
				h.AssertFunctionName(t, configProvider.PostContainerRunOps()[1], "CopyOut")
			})
		})

		when("report destination directory is provided", func() {
			lifecycleOps = append(lifecycleOps, func(opts *build.LifecycleOptions) {
				opts.ReportDestinationDir = "a-destination-dir"
			})

			it("provides copy-sbom-func as a post container operation", func() {
				h.AssertEq(t, len(configProvider.PostContainerRunOps()), 2)
				h.AssertFunctionName(t, configProvider.PostContainerRunOps()[0], "EnsureVolumeAccess")
				h.AssertFunctionName(t, configProvider.PostContainerRunOps()[1], "CopyOut")
			})
		})

		when("--creation-time", func() {
			when("platform < 0.9", func() {
				platformAPI = api.MustParse("0.8")

				intTime, _ := strconv.ParseInt("1234567890", 10, 64)
				providedTime := time.Unix(intTime, 0).UTC()

				lifecycleOps = append(lifecycleOps, func(baseOpts *build.LifecycleOptions) {
					baseOpts.CreationTime = &providedTime
				})

				it("is ignored", func() {
					h.AssertSliceNotContains(t, configProvider.ContainerConfig().Env, "SOURCE_DATE_EPOCH=1234567890")
				})
			})

			when("platform >= 0.9", func() {
				platformAPI = api.MustParse("0.9")

				when("provided", func() {
					intTime, _ := strconv.ParseInt("1234567890", 10, 64)
					providedTime := time.Unix(intTime, 0).UTC()

					lifecycleOps = append(lifecycleOps, func(baseOpts *build.LifecycleOptions) {
						baseOpts.CreationTime = &providedTime
					})

					it("configures the phase with env SOURCE_DATE_EPOCH", func() {
						h.AssertSliceContains(t, configProvider.ContainerConfig().Env, "SOURCE_DATE_EPOCH=1234567890")
					})
				})

				when("not provided", func() {
					it("does not panic", func() {
						// no-op
					})
				})
			})
		})
	})
}

func newFakeVolumeCache() *fakes.FakeCache {
	c := fakes.NewFakeCache()
	c.ReturnForType = cache.Volume
	c.ReturnForName = "some-cache"
	return c
}

func newFakeImageCache() *fakes.FakeCache {
	c := fakes.NewFakeCache()
	c.ReturnForType = cache.Image
	c.ReturnForName = "some-cache-image"
	return c
}

func newFakeFetchRunImageFunc(f *fakeImageFetcher) func(name string) (string, error) {
	return func(name string) (string, error) {
		return fmt.Sprintf("ephemeral-%s", name), f.fetchRunImage(name)
	}
}

type fakeImageFetcher struct {
	callCount           int
	calledWithArgAtCall map[int]string
}

func (f *fakeImageFetcher) fetchRunImage(name string) error {
	f.calledWithArgAtCall[f.callCount] = name
	f.callCount++
	return nil
}

type fakeDockerClient struct {
	nNetworks int
	build.DockerClient
}

func (f *fakeDockerClient) NetworkList(ctx context.Context, opts network.CreateOptions) ([]network.Inspect, error) {
	ret := make([]network.Inspect, f.nNetworks)
	return ret, nil
}

func (f *fakeDockerClient) NetworkCreate(ctx context.Context, name string, options network.CreateOptions) (network.CreateResponse, error) {
	f.nNetworks++
	return network.CreateResponse{}, nil
}

func (f *fakeDockerClient) NetworkRemove(ctx context.Context, network string) error {
	f.nNetworks--
	return nil
}

func newTestLifecycleExecErr(t *testing.T, logVerbose bool, tmpDir string, ops ...func(*build.LifecycleOptions)) (*build.LifecycleExecution, error) {
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.38"))
	h.AssertNil(t, err)

	var outBuf bytes.Buffer
	logger := logging.NewLogWithWriters(&outBuf, &outBuf)
	if logVerbose {
		logger.Level = log.DebugLevel
	}

	defaultBuilder, err := fakes.NewFakeBuilder()
	h.AssertNil(t, err)

	opts := build.LifecycleOptions{
		AppPath:    "some-app-path",
		Builder:    defaultBuilder,
		HTTPProxy:  "some-http-proxy",
		HTTPSProxy: "some-https-proxy",
		NoProxy:    "some-no-proxy",
		Termui:     &fakes.FakeTermui{},
	}

	for _, op := range ops {
		op(&opts)
	}

	return build.NewLifecycleExecution(logger, docker, tmpDir, opts)
}

func newTestLifecycleExec(t *testing.T, logVerbose bool, tmpDir string, ops ...func(*build.LifecycleOptions)) *build.LifecycleExecution {
	t.Helper()

	lifecycleExec, err := newTestLifecycleExecErr(t, logVerbose, tmpDir, ops...)
	h.AssertNil(t, err)
	return lifecycleExec
}
