package build_test

import (
	"bytes"
	"io"
	"testing"

	ifakes "github.com/buildpacks/imgutil/fakes"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/heroku/color"
	"github.com/pkg/errors"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/build"
	"github.com/buildpacks/pack/internal/build/fakes"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestPhaseConfigProvider(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)

	spec.Run(t, "phase_config_provider", testPhaseConfigProvider, spec.Report(report.Terminal{}), spec.Sequential())
}

func testPhaseConfigProvider(t *testing.T, when spec.G, it spec.S) {
	when("#NewPhaseConfigProvider", func() {
		it("returns a phase config provider with defaults", func() {
			expectedBuilderImage := ifakes.NewImage("some-builder-name", "", nil)
			fakeBuilder, err := fakes.NewFakeBuilder(fakes.WithImage(expectedBuilderImage))
			h.AssertNil(t, err)
			lifecycle := newTestLifecycleExec(t, false, "some-temp-dir", fakes.WithBuilder(fakeBuilder))
			expectedPhaseName := "some-name"
			expectedCmd := strslice.StrSlice{"/cnb/lifecycle/" + expectedPhaseName}

			phaseConfigProvider := build.NewPhaseConfigProvider(expectedPhaseName, lifecycle)

			h.AssertEq(t, phaseConfigProvider.Name(), expectedPhaseName)
			h.AssertEq(t, phaseConfigProvider.ContainerConfig().Cmd, expectedCmd)
			h.AssertEq(t, phaseConfigProvider.ContainerConfig().Image, expectedBuilderImage.Name())
			h.AssertEq(t, phaseConfigProvider.ContainerConfig().Labels, map[string]string{"author": "pack"})

			// NewFakeBuilder sets the Platform API
			h.AssertSliceContains(t, phaseConfigProvider.ContainerConfig().Env, "CNB_PLATFORM_API=0.4")

			// CreateFakeLifecycleExecution sets the following:
			h.AssertSliceContains(t, phaseConfigProvider.ContainerConfig().Env, "HTTP_PROXY=some-http-proxy")
			h.AssertSliceContains(t, phaseConfigProvider.ContainerConfig().Env, "http_proxy=some-http-proxy")
			h.AssertSliceContains(t, phaseConfigProvider.ContainerConfig().Env, "HTTPS_PROXY=some-https-proxy")
			h.AssertSliceContains(t, phaseConfigProvider.ContainerConfig().Env, "https_proxy=some-https-proxy")
			h.AssertSliceContains(t, phaseConfigProvider.ContainerConfig().Env, "NO_PROXY=some-no-proxy")
			h.AssertSliceContains(t, phaseConfigProvider.ContainerConfig().Env, "no_proxy=some-no-proxy")

			h.AssertSliceContainsMatch(t, phaseConfigProvider.HostConfig().Binds, "pack-layers-.*:/layers")
			h.AssertSliceContainsMatch(t, phaseConfigProvider.HostConfig().Binds, "pack-app-.*:/workspace")

			h.AssertEq(t, phaseConfigProvider.HostConfig().Isolation, container.IsolationEmpty)
			h.AssertEq(t, phaseConfigProvider.HostConfig().UsernsMode, container.UsernsMode("host"))
			h.AssertSliceContains(t, phaseConfigProvider.HostConfig().SecurityOpt, "no-new-privileges=true")
		})

		when("building for Windows", func() {
			it("sets process isolation", func() {
				fakeBuilderImage := ifakes.NewImage("fake-builder", "", nil)
				h.AssertNil(t, fakeBuilderImage.SetOS("windows"))
				fakeBuilder, err := fakes.NewFakeBuilder(fakes.WithImage(fakeBuilderImage))
				h.AssertNil(t, err)
				lifecycle := newTestLifecycleExec(t, false, "some-temp-dir", fakes.WithBuilder(fakeBuilder))

				phaseConfigProvider := build.NewPhaseConfigProvider("some-name", lifecycle)

				h.AssertEq(t, phaseConfigProvider.HostConfig().Isolation, container.IsolationProcess)
				h.AssertSliceNotContains(t, phaseConfigProvider.HostConfig().SecurityOpt, "no-new-privileges=true")
			})
		})

		when("building with interactive mode", func() {
			it("returns a phase config provider with interactive args", func() {
				handler := func(bodyChan <-chan container.WaitResponse, errChan <-chan error, reader io.Reader) error {
					return errors.New("i was called")
				}

				fakeTermui := &fakes.FakeTermui{HandlerFunc: handler}
				lifecycle := newTestLifecycleExec(t, false, "some-temp-dir", fakes.WithTermui(fakeTermui))
				phaseConfigProvider := build.NewPhaseConfigProvider("some-name", lifecycle)

				h.AssertError(t, phaseConfigProvider.Handler()(nil, nil, nil), "i was called")
			})
		})

		when("called with WithArgs", func() {
			it("sets args on the config", func() {
				lifecycle := newTestLifecycleExec(t, false, "some-temp-dir")
				expectedArgs := strslice.StrSlice{"some-arg-1", "some-arg-2"}

				phaseConfigProvider := build.NewPhaseConfigProvider(
					"some-name",
					lifecycle,
					build.WithArgs(expectedArgs...),
				)

				cmd := phaseConfigProvider.ContainerConfig().Cmd
				h.AssertSliceContainsInOrder(t, cmd, "some-arg-1", "some-arg-2")
			})
		})

		when("called with WithFlags", func() {
			it("sets args on the config", func() {
				lifecycle := newTestLifecycleExec(t, false, "some-temp-dir")

				phaseConfigProvider := build.NewPhaseConfigProvider(
					"some-name",
					lifecycle,
					build.WithArgs("arg-1", "arg-2"),
					build.WithFlags("flag-1", "flag-2"),
				)

				cmd := phaseConfigProvider.ContainerConfig().Cmd
				h.AssertSliceContainsInOrder(t, cmd, "flag-1", "flag-2", "arg-1", "arg-2")
			})
		})

		when("called with WithBinds", func() {
			it("sets binds on the config", func() {
				lifecycle := newTestLifecycleExec(t, false, "some-temp-dir")
				expectedBinds := []string{"some-bind-1", "some-bind-2"}

				phaseConfigProvider := build.NewPhaseConfigProvider(
					"some-name",
					lifecycle,
					build.WithBinds(expectedBinds...),
				)

				h.AssertSliceContains(t, phaseConfigProvider.HostConfig().Binds, expectedBinds...)
			})
		})

		when("called with WithDaemonAccess", func() {
			when("building for non-Windows", func() {
				it("sets daemon access on the config", func() {
					lifecycle := newTestLifecycleExec(t, false, "some-temp-dir")

					phaseConfigProvider := build.NewPhaseConfigProvider(
						"some-name",
						lifecycle,
						build.WithDaemonAccess(""),
					)

					h.AssertEq(t, phaseConfigProvider.ContainerConfig().User, "root")
					h.AssertSliceContains(t, phaseConfigProvider.HostConfig().Binds, "/var/run/docker.sock:/var/run/docker.sock")
					h.AssertSliceContains(t, phaseConfigProvider.HostConfig().SecurityOpt, "label=disable")
				})
			})

			when("building for Windows", func() {
				it("sets daemon access on the config", func() {
					fakeBuilderImage := ifakes.NewImage("fake-builder", "", nil)
					h.AssertNil(t, fakeBuilderImage.SetOS("windows"))
					fakeBuilder, err := fakes.NewFakeBuilder(fakes.WithImage(fakeBuilderImage))
					h.AssertNil(t, err)
					lifecycle := newTestLifecycleExec(t, false, "some-temp-dir", fakes.WithBuilder(fakeBuilder))

					phaseConfigProvider := build.NewPhaseConfigProvider(
						"some-name",
						lifecycle,
						build.WithDaemonAccess(""),
					)

					h.AssertEq(t, phaseConfigProvider.ContainerConfig().User, "ContainerAdministrator")
					h.AssertSliceContains(t, phaseConfigProvider.HostConfig().Binds, `\\.\pipe\docker_engine:\\.\pipe\docker_engine`)
				})
			})
		})

		when("called with WithEnv", func() {
			it("sets the environment on the config", func() {
				lifecycle := newTestLifecycleExec(t, false, "some-temp-dir")

				phaseConfigProvider := build.NewPhaseConfigProvider(
					"some-name",
					lifecycle,
					build.WithEnv("SOME_VARIABLE=some-value"),
				)

				h.AssertSliceContains(t, phaseConfigProvider.ContainerConfig().Env, "SOME_VARIABLE=some-value")
			})
		})

		when("called with WithImage", func() {
			it("sets the image on the config", func() {
				lifecycle := newTestLifecycleExec(t, false, "some-temp-dir")

				phaseConfigProvider := build.NewPhaseConfigProvider(
					"some-name",
					lifecycle,
					build.WithImage("some-image-name"),
				)

				h.AssertEq(t, phaseConfigProvider.ContainerConfig().Image, "some-image-name")
			})
		})

		when("called with WithNetwork", func() {
			it("sets the network mode on the config", func() {
				lifecycle := newTestLifecycleExec(t, false, "some-temp-dir")
				expectedNetworkMode := "some-network-mode"

				phaseConfigProvider := build.NewPhaseConfigProvider(
					"some-name",
					lifecycle,
					build.WithNetwork(expectedNetworkMode),
				)

				h.AssertEq(
					t,
					phaseConfigProvider.HostConfig().NetworkMode,
					container.NetworkMode(expectedNetworkMode),
				)
			})
		})

		when("called with WithRegistryAccess", func() {
			it("sets registry access on the config", func() {
				lifecycle := newTestLifecycleExec(t, false, "some-temp-dir")
				authConfig := "some-auth-config"

				phaseConfigProvider := build.NewPhaseConfigProvider(
					"some-name",
					lifecycle,
					build.WithRegistryAccess(authConfig),
				)

				h.AssertSliceContains(
					t,
					phaseConfigProvider.ContainerConfig().Env,
					"CNB_REGISTRY_AUTH="+authConfig,
				)
			})
		})

		when("called with WithRoot", func() {
			when("building for non-Windows", func() {
				it("sets root user on the config", func() {
					lifecycle := newTestLifecycleExec(t, false, "some-temp-dir")

					phaseConfigProvider := build.NewPhaseConfigProvider(
						"some-name",
						lifecycle,
						build.WithRoot(),
					)

					h.AssertEq(t, phaseConfigProvider.ContainerConfig().User, "root")
				})
			})

			when("building for Windows", func() {
				it("sets root user on the config", func() {
					fakeBuilderImage := ifakes.NewImage("fake-builder", "", nil)
					h.AssertNil(t, fakeBuilderImage.SetOS("windows"))
					fakeBuilder, err := fakes.NewFakeBuilder(fakes.WithImage(fakeBuilderImage))
					h.AssertNil(t, err)
					lifecycle := newTestLifecycleExec(t, false, "some-temp-dir", fakes.WithBuilder(fakeBuilder))

					phaseConfigProvider := build.NewPhaseConfigProvider(
						"some-name",
						lifecycle,
						build.WithRoot(),
					)

					h.AssertEq(t, phaseConfigProvider.ContainerConfig().User, "ContainerAdministrator")
				})
			})
		})

		when("called with WithLogPrefix", func() {
			it("sets prefix writers", func() {
				lifecycle := newTestLifecycleExec(t, false, "some-temp-dir")

				phaseConfigProvider := build.NewPhaseConfigProvider(
					"some-name",
					lifecycle,
					build.WithLogPrefix("some-prefix"),
				)

				_, isType := phaseConfigProvider.InfoWriter().(*logging.PrefixWriter)
				h.AssertEq(t, isType, true)

				_, isType = phaseConfigProvider.ErrorWriter().(*logging.PrefixWriter)
				h.AssertEq(t, isType, true)
			})
		})

		when("verbose", func() {
			it("prints debug information about the phase", func() {
				var outBuf bytes.Buffer
				logger := logging.NewLogWithWriters(&outBuf, &outBuf, logging.WithVerbose())

				docker, err := client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.38"))
				h.AssertNil(t, err)

				defaultBuilder, err := fakes.NewFakeBuilder()
				h.AssertNil(t, err)

				opts := build.LifecycleOptions{
					AppPath: "some-app-path",
					Builder: defaultBuilder,
				}

				lifecycleExec, err := build.NewLifecycleExecution(logger, docker, "some-temp-dir", opts)
				h.AssertNil(t, err)

				_ = build.NewPhaseConfigProvider(
					"some-name",
					lifecycleExec,
					build.WithRoot(),
				)

				h.AssertContains(t, outBuf.String(), "Running the 'some-name' on OS")
				h.AssertContains(t, outBuf.String(), "Args: '/cnb/lifecycle/some-name'")
				h.AssertContains(t, outBuf.String(), "System Envs: 'CNB_PLATFORM_API=0.4'")
				h.AssertContains(t, outBuf.String(), "Image: 'some-builder-name'")
				h.AssertContains(t, outBuf.String(), "User:")
				h.AssertContains(t, outBuf.String(), "Labels: 'map[author:pack]'")
				h.AssertContainsMatch(t, outBuf.String(), `Binds: \'\S+:\S+layers \S+:\S+workspace'`)
				h.AssertContains(t, outBuf.String(), "Network Mode: ''")
			})

			when("there is registry auth", func() {
				it("sanitizes the output", func() {
					authConfig := "some-auth-config"

					var outBuf bytes.Buffer
					logger := logging.NewLogWithWriters(&outBuf, &outBuf, logging.WithVerbose())

					docker, err := client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.38"))
					h.AssertNil(t, err)

					defaultBuilder, err := fakes.NewFakeBuilder()
					h.AssertNil(t, err)

					opts := build.LifecycleOptions{
						AppPath: "some-app-path",
						Builder: defaultBuilder,
					}

					lifecycleExec, err := build.NewLifecycleExecution(logger, docker, "some-temp-dir", opts)
					h.AssertNil(t, err)

					_ = build.NewPhaseConfigProvider(
						"some-name",
						lifecycleExec,
						build.WithRegistryAccess(authConfig),
					)

					h.AssertContains(t, outBuf.String(), "System Envs: 'CNB_REGISTRY_AUTH=<redacted> CNB_PLATFORM_API=0.4'")
				})
			})
		})
	})
}
