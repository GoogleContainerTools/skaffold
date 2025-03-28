package commands_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/heroku/color"
	"github.com/pkg/errors"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/spf13/cobra"

	pubbldpkg "github.com/buildpacks/pack/buildpackage"
	"github.com/buildpacks/pack/internal/commands"
	"github.com/buildpacks/pack/internal/commands/fakes"
	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/image"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestPackageBuildpackCommand(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "PackageBuildpackCommand", testPackageBuildpackCommand, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testPackageBuildpackCommand(t *testing.T, when spec.G, it spec.S) {
	when("PackageBuildpack#Execute", func() {
		when("valid package config", func() {
			it("prints deprecation warning", func() {
				var outBuf bytes.Buffer
				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				cmd := packageBuildpackCommand(withLogger(logger))
				h.AssertNil(t, cmd.Execute())
				h.AssertContains(t, outBuf.String(), "Warning: Command 'pack package-buildpack' has been deprecated, please use 'pack buildpack package' instead")
			})

			it("reads package config from the configured path", func() {
				fakePackageConfigReader := fakes.NewFakePackageConfigReader()
				expectedPackageConfigPath := "/path/to/some/file"

				packageBuildpackCommand := packageBuildpackCommand(
					withPackageConfigReader(fakePackageConfigReader),
					withPackageConfigPath(expectedPackageConfigPath),
				)

				err := packageBuildpackCommand.Execute()
				h.AssertNil(t, err)

				h.AssertEq(t, fakePackageConfigReader.ReadCalledWithArg, expectedPackageConfigPath)
			})

			it("creates package with correct image name", func() {
				fakeBuildpackPackager := &fakes.FakeBuildpackPackager{}

				packageBuildpackCommand := packageBuildpackCommand(
					withImageName("my-specific-image"),
					withBuildpackPackager(fakeBuildpackPackager),
				)

				err := packageBuildpackCommand.Execute()
				h.AssertNil(t, err)

				receivedOptions := fakeBuildpackPackager.CreateCalledWithOptions

				h.AssertEq(t, receivedOptions.Name, "my-specific-image")
			})

			it("creates package with config returned by the reader", func() {
				fakeBuildpackPackager := &fakes.FakeBuildpackPackager{}

				myConfig := pubbldpkg.Config{
					Buildpack: dist.BuildpackURI{URI: "test"},
				}

				packageBuildpackCommand := packageBuildpackCommand(
					withBuildpackPackager(fakeBuildpackPackager),
					withPackageConfigReader(fakes.NewFakePackageConfigReader(whereReadReturns(myConfig, nil))),
				)

				err := packageBuildpackCommand.Execute()
				h.AssertNil(t, err)

				receivedOptions := fakeBuildpackPackager.CreateCalledWithOptions

				h.AssertEq(t, receivedOptions.Config, myConfig)
			})

			when("pull-policy", func() {
				var (
					outBuf                bytes.Buffer
					cmd                   *cobra.Command
					args                  []string
					fakeBuildpackPackager *fakes.FakeBuildpackPackager
				)

				it.Before(func() {
					logger := logging.NewLogWithWriters(&outBuf, &outBuf)
					fakeBuildpackPackager = &fakes.FakeBuildpackPackager{}

					cmd = packageBuildpackCommand(withLogger(logger), withBuildpackPackager(fakeBuildpackPackager))
					args = []string{
						"some-image-name",
						"--config", "/path/to/some/file",
					}
				})

				it("pull-policy=never sets policy", func() {
					args = append(args, "--pull-policy", "never")
					cmd.SetArgs(args)

					err := cmd.Execute()
					h.AssertNil(t, err)

					receivedOptions := fakeBuildpackPackager.CreateCalledWithOptions
					h.AssertEq(t, receivedOptions.PullPolicy, image.PullNever)
				})

				it("pull-policy=always sets policy", func() {
					args = append(args, "--pull-policy", "always")
					cmd.SetArgs(args)

					err := cmd.Execute()
					h.AssertNil(t, err)

					receivedOptions := fakeBuildpackPackager.CreateCalledWithOptions
					h.AssertEq(t, receivedOptions.PullPolicy, image.PullAlways)
				})
				it("takes precedence over a configured pull policy", func() {
					logger := logging.NewLogWithWriters(&bytes.Buffer{}, &bytes.Buffer{})
					configReader := fakes.NewFakePackageConfigReader()
					buildpackPackager := &fakes.FakeBuildpackPackager{}
					clientConfig := config.Config{PullPolicy: "if-not-present"}

					command := commands.PackageBuildpack(logger, clientConfig, buildpackPackager, configReader)
					command.SetArgs([]string{
						"some-image-name",
						"--config", "/path/to/some/file",
						"--pull-policy",
						"never",
					})

					err := command.Execute()
					h.AssertNil(t, err)

					receivedOptions := buildpackPackager.CreateCalledWithOptions
					h.AssertEq(t, receivedOptions.PullPolicy, image.PullNever)
				})
			})
			when("configured pull policy", func() {
				it("uses the configured pull policy", func() {
					logger := logging.NewLogWithWriters(&bytes.Buffer{}, &bytes.Buffer{})
					configReader := fakes.NewFakePackageConfigReader()
					buildpackPackager := &fakes.FakeBuildpackPackager{}
					clientConfig := config.Config{PullPolicy: "never"}

					command := commands.PackageBuildpack(logger, clientConfig, buildpackPackager, configReader)
					command.SetArgs([]string{
						"some-image-name",
						"--config", "/path/to/some/file",
					})

					err := command.Execute()
					h.AssertNil(t, err)

					receivedOptions := buildpackPackager.CreateCalledWithOptions
					h.AssertEq(t, receivedOptions.PullPolicy, image.PullNever)
				})
			})
		})
	})

	when("invalid flags", func() {
		when("both --publish and --pull-policy never flags are specified", func() {
			it("errors with a descriptive message", func() {
				logger := logging.NewLogWithWriters(&bytes.Buffer{}, &bytes.Buffer{})
				configReader := fakes.NewFakePackageConfigReader()
				buildpackPackager := &fakes.FakeBuildpackPackager{}
				clientConfig := config.Config{}

				command := commands.PackageBuildpack(logger, clientConfig, buildpackPackager, configReader)
				command.SetArgs([]string{
					"some-image-name",
					"--config", "/path/to/some/file",
					"--publish",
					"--pull-policy",
					"never",
				})

				err := command.Execute()
				h.AssertNotNil(t, err)
				h.AssertError(t, err, "--publish and --pull-policy never cannot be used together. The --publish flag requires the use of remote images.")
			})
		})

		it("logs an error and exits when package toml is invalid", func() {
			outBuf := &bytes.Buffer{}
			expectedErr := errors.New("it went wrong")

			packageBuildpackCommand := packageBuildpackCommand(
				withLogger(logging.NewLogWithWriters(outBuf, outBuf)),
				withPackageConfigReader(
					fakes.NewFakePackageConfigReader(whereReadReturns(pubbldpkg.Config{}, expectedErr)),
				),
			)

			err := packageBuildpackCommand.Execute()
			h.AssertNotNil(t, err)

			h.AssertContains(t, outBuf.String(), fmt.Sprintf("ERROR: reading config: %s", expectedErr))
		})

		when("package-config is specified", func() {
			it("errors with a descriptive message", func() {
				outBuf := &bytes.Buffer{}

				config := &packageCommandConfig{
					logger:              logging.NewLogWithWriters(outBuf, outBuf),
					packageConfigReader: fakes.NewFakePackageConfigReader(),
					buildpackPackager:   &fakes.FakeBuildpackPackager{},

					imageName:  "some-image-name",
					configPath: "/path/to/some/file",
				}

				cmd := commands.PackageBuildpack(config.logger, config.clientConfig, config.buildpackPackager, config.packageConfigReader)
				cmd.SetArgs([]string{config.imageName, "--package-config", config.configPath})

				err := cmd.Execute()
				h.AssertError(t, err, "unknown flag: --package-config")
			})
		})

		when("no config path is specified", func() {
			it("creates a default config", func() {
				config := &packageCommandConfig{
					logger:              logging.NewLogWithWriters(&bytes.Buffer{}, &bytes.Buffer{}),
					packageConfigReader: fakes.NewFakePackageConfigReader(),
					buildpackPackager:   &fakes.FakeBuildpackPackager{},

					imageName: "some-image-name",
				}

				cmd := commands.PackageBuildpack(config.logger, config.clientConfig, config.buildpackPackager, config.packageConfigReader)
				cmd.SetArgs([]string{config.imageName})

				err := cmd.Execute()
				h.AssertNil(t, err)

				receivedOptions := config.buildpackPackager.CreateCalledWithOptions
				h.AssertEq(t, receivedOptions.Config.Buildpack.URI, ".")
			})
		})

		when("--pull-policy unknown-policy", func() {
			it("fails to run", func() {
				logger := logging.NewLogWithWriters(&bytes.Buffer{}, &bytes.Buffer{})
				configReader := fakes.NewFakePackageConfigReader()
				buildpackPackager := &fakes.FakeBuildpackPackager{}
				clientConfig := config.Config{}

				command := commands.PackageBuildpack(logger, clientConfig, buildpackPackager, configReader)
				command.SetArgs([]string{
					"some-image-name",
					"--config", "/path/to/some/file",
					"--pull-policy",
					"unknown-policy",
				})

				h.AssertError(t, command.Execute(), "parsing pull policy")
			})
		})
	})
}

func packageBuildpackCommand(ops ...packageCommandOption) *cobra.Command {
	config := &packageCommandConfig{
		logger:              logging.NewLogWithWriters(&bytes.Buffer{}, &bytes.Buffer{}),
		packageConfigReader: fakes.NewFakePackageConfigReader(),
		buildpackPackager:   &fakes.FakeBuildpackPackager{},
		clientConfig:        config.Config{},

		imageName:  "some-image-name",
		configPath: "/path/to/some/file",
	}

	for _, op := range ops {
		op(config)
	}

	cmd := commands.PackageBuildpack(config.logger, config.clientConfig, config.buildpackPackager, config.packageConfigReader)
	cmd.SetArgs([]string{config.imageName, "--config", config.configPath})

	return cmd
}
