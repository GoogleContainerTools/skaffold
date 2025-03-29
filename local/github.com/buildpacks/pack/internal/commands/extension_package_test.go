package commands_test

import (
	"bytes"
	"fmt"
	"path/filepath"
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

func TestExtensionPackageCommand(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "ExtensionPackageCommand", testExtensionPackageCommand, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testExtensionPackageCommand(t *testing.T, when spec.G, it spec.S) {
	var (
		logger *logging.LogWithWriters
		outBuf bytes.Buffer
	)

	it.Before(func() {
		logger = logging.NewLogWithWriters(&outBuf, &outBuf)
	})

	when("Package#Execute", func() {
		var fakeExtensionPackager *fakes.FakeBuildpackPackager

		it.Before(func() {
			fakeExtensionPackager = &fakes.FakeBuildpackPackager{}
		})

		when("valid package config", func() {
			it("reads package config from the configured path", func() {
				fakePackageConfigReader := fakes.NewFakePackageConfigReader()
				expectedPackageConfigPath := "/path/to/some/file"

				cmd := packageExtensionCommand(
					withExtensionPackageConfigReader(fakePackageConfigReader),
					withExtensionPackageConfigPath(expectedPackageConfigPath),
				)
				err := cmd.Execute()
				h.AssertNil(t, err)

				h.AssertEq(t, fakePackageConfigReader.ReadCalledWithArg, expectedPackageConfigPath)
			})

			it("creates package with correct image name", func() {
				cmd := packageExtensionCommand(
					withExtensionImageName("my-specific-image"),
					withExtensionPackager(fakeExtensionPackager),
				)
				err := cmd.Execute()
				h.AssertNil(t, err)

				receivedOptions := fakeExtensionPackager.CreateCalledWithOptions
				h.AssertEq(t, receivedOptions.Name, "my-specific-image")
			})

			it("creates package with config returned by the reader", func() {
				myConfig := pubbldpkg.Config{
					Extension: dist.BuildpackURI{URI: "test"},
				}

				cmd := packageExtensionCommand(
					withExtensionPackager(fakeExtensionPackager),
					withExtensionPackageConfigReader(fakes.NewFakePackageConfigReader(whereReadReturns(myConfig, nil))),
				)
				err := cmd.Execute()
				h.AssertNil(t, err)

				receivedOptions := fakeExtensionPackager.CreateCalledWithOptions
				h.AssertEq(t, receivedOptions.Config, myConfig)
			})

			when("file format", func() {
				when("extension is .cnb", func() {
					it("does not modify the name", func() {
						cmd := packageExtensionCommand(withExtensionPackager(fakeExtensionPackager))
						cmd.SetArgs([]string{"test.cnb", "-f", "file"})
						h.AssertNil(t, cmd.Execute())

						receivedOptions := fakeExtensionPackager.CreateCalledWithOptions
						h.AssertEq(t, receivedOptions.Name, "test.cnb")
					})
				})
				when("extension is empty", func() {
					it("appends .cnb to the name", func() {
						cmd := packageExtensionCommand(withExtensionPackager(fakeExtensionPackager))
						cmd.SetArgs([]string{"test", "-f", "file"})
						h.AssertNil(t, cmd.Execute())

						receivedOptions := fakeExtensionPackager.CreateCalledWithOptions
						h.AssertEq(t, receivedOptions.Name, "test.cnb")
					})
				})
				when("extension is something other than .cnb", func() {
					it("does not modify the name but shows a warning", func() {
						cmd := packageExtensionCommand(withExtensionPackager(fakeExtensionPackager), withExtensionLogger(logger))
						cmd.SetArgs([]string{"test.tar.gz", "-f", "file"})
						h.AssertNil(t, cmd.Execute())

						receivedOptions := fakeExtensionPackager.CreateCalledWithOptions
						h.AssertEq(t, receivedOptions.Name, "test.tar.gz")
						h.AssertContains(t, outBuf.String(), "'.gz' is not a valid extension for a packaged extension. Packaged extensions must have a '.cnb' extension")
					})
				})
			})

			when("pull-policy", func() {
				var pullPolicyArgs = []string{
					"some-image-name",
					"--config", "/path/to/some/file",
					"--pull-policy",
				}

				it("pull-policy=never sets policy", func() {
					cmd := packageExtensionCommand(withExtensionPackager(fakeExtensionPackager))
					cmd.SetArgs(append(pullPolicyArgs, "never"))
					h.AssertNil(t, cmd.Execute())

					receivedOptions := fakeExtensionPackager.CreateCalledWithOptions
					h.AssertEq(t, receivedOptions.PullPolicy, image.PullNever)
				})

				it("pull-policy=always sets policy", func() {
					cmd := packageExtensionCommand(withExtensionPackager(fakeExtensionPackager))
					cmd.SetArgs(append(pullPolicyArgs, "always"))
					h.AssertNil(t, cmd.Execute())

					receivedOptions := fakeExtensionPackager.CreateCalledWithOptions
					h.AssertEq(t, receivedOptions.PullPolicy, image.PullAlways)
				})
			})
			when("no --pull-policy", func() {
				var pullPolicyArgs = []string{
					"some-image-name",
					"--config", "/path/to/some/file",
				}

				it("uses the default policy when no policy configured", func() {
					cmd := packageExtensionCommand(withExtensionPackager(fakeExtensionPackager))
					cmd.SetArgs(pullPolicyArgs)
					h.AssertNil(t, cmd.Execute())

					receivedOptions := fakeExtensionPackager.CreateCalledWithOptions
					h.AssertEq(t, receivedOptions.PullPolicy, image.PullAlways)
				})
				it("uses the configured pull policy when policy configured", func() {
					cmd := packageExtensionCommand(
						withExtensionPackager(fakeExtensionPackager),
						withExtensionClientConfig(config.Config{PullPolicy: "never"}),
					)

					cmd.SetArgs([]string{
						"some-image-name",
						"--config", "/path/to/some/file",
					})

					err := cmd.Execute()
					h.AssertNil(t, err)

					receivedOptions := fakeExtensionPackager.CreateCalledWithOptions
					h.AssertEq(t, receivedOptions.PullPolicy, image.PullNever)
				})
			})
		})

		when("no config path is specified", func() {
			when("no path is specified", func() {
				it("creates a default config with the uri set to the current working directory", func() {
					cmd := packageExtensionCommand(withExtensionPackager(fakeExtensionPackager))
					cmd.SetArgs([]string{"some-name"})
					h.AssertNil(t, cmd.Execute())

					receivedOptions := fakeExtensionPackager.CreateCalledWithOptions
					h.AssertEq(t, receivedOptions.Config.Extension.URI, ".")
				})
			})
		})

		when("a path is specified", func() {
			when("no multi-platform", func() {
				it("creates a default config with the appropriate path", func() {
					cmd := packageExtensionCommand(withExtensionPackager(fakeExtensionPackager))
					cmd.SetArgs([]string{"some-name", "-p", ".."})
					h.AssertNil(t, cmd.Execute())
					bpPath, _ := filepath.Abs("..")
					receivedOptions := fakeExtensionPackager.CreateCalledWithOptions
					h.AssertEq(t, receivedOptions.Config.Extension.URI, bpPath)
				})
			})

			when("multi-platform", func() {
				var targets []dist.Target

				when("single extension", func() {
					it.Before(func() {
						targets = []dist.Target{
							{OS: "linux", Arch: "amd64"},
							{OS: "windows", Arch: "amd64"},
						}
					})

					it("creates a multi-platform extension package", func() {
						cmd := packageExtensionCommand(withExtensionPackager(fakeExtensionPackager))
						cmd.SetArgs([]string{"some-name", "-p", "some-path", "--target", "linux/amd64", "--target", "windows/amd64", "--format", "image", "--publish"})
						h.AssertNil(t, cmd.Execute())
						h.AssertEq(t, fakeExtensionPackager.CreateCalledWithOptions.Targets, targets)
					})
				})
			})
		})
	})

	when("invalid flags", func() {
		when("both --publish and --pull-policy never flags are specified", func() {
			it("errors with a descriptive message", func() {
				cmd := packageExtensionCommand()
				cmd.SetArgs([]string{
					"some-image-name", "--config", "/path/to/some/file",
					"--publish",
					"--pull-policy", "never",
				})

				err := cmd.Execute()
				h.AssertNotNil(t, err)
				h.AssertError(t, err, "--publish and --pull-policy=never cannot be used together. The --publish flag requires the use of remote images.")
			})
		})

		it("logs an error and exits when package toml is invalid", func() {
			expectedErr := errors.New("it went wrong")

			cmd := packageExtensionCommand(
				withExtensionLogger(logger),
				withExtensionPackageConfigReader(
					fakes.NewFakePackageConfigReader(whereReadReturns(pubbldpkg.Config{}, expectedErr)),
				),
			)

			err := cmd.Execute()
			h.AssertNotNil(t, err)

			h.AssertContains(t, outBuf.String(), fmt.Sprintf("ERROR: reading config: %s", expectedErr))
		})

		when("package-config is specified", func() {
			it("errors with a descriptive message", func() {
				cmd := packageExtensionCommand()
				cmd.SetArgs([]string{"some-name", "--package-config", "some-path"})

				err := cmd.Execute()
				h.AssertError(t, err, "unknown flag: --package-config")
			})
		})

		when("--pull-policy unknown-policy", func() {
			it("fails to run", func() {
				cmd := packageExtensionCommand()
				cmd.SetArgs([]string{
					"some-image-name",
					"--config", "/path/to/some/file",
					"--pull-policy",
					"unknown-policy",
				})

				h.AssertError(t, cmd.Execute(), "parsing pull policy")
			})
		})

		when("--target cannot be parsed", func() {
			it("errors with a descriptive message", func() {
				cmd := packageCommand()
				cmd.SetArgs([]string{
					"some-image-name", "--config", "/path/to/some/file",
					"--target", "something/wrong", "--publish",
				})

				err := cmd.Execute()
				h.AssertNotNil(t, err)
				h.AssertError(t, err, "unknown target: 'something/wrong'")
			})
		})
	})
}

type packageExtensionCommandConfig struct {
	logger              *logging.LogWithWriters
	packageConfigReader *fakes.FakePackageConfigReader
	extensionPackager   *fakes.FakeBuildpackPackager
	clientConfig        config.Config
	imageName           string
	configPath          string
}

type packageExtensionCommandOption func(config *packageExtensionCommandConfig)

func packageExtensionCommand(ops ...packageExtensionCommandOption) *cobra.Command {
	config := &packageExtensionCommandConfig{
		logger:              logging.NewLogWithWriters(&bytes.Buffer{}, &bytes.Buffer{}),
		packageConfigReader: fakes.NewFakePackageConfigReader(),
		extensionPackager:   &fakes.FakeBuildpackPackager{},
		clientConfig:        config.Config{},
		imageName:           "some-image-name",
		configPath:          "/path/to/some/file",
	}

	for _, op := range ops {
		op(config)
	}

	cmd := commands.ExtensionPackage(config.logger, config.clientConfig, config.extensionPackager, config.packageConfigReader)
	cmd.SetArgs([]string{config.imageName, "--config", config.configPath})

	return cmd
}

func withExtensionLogger(logger *logging.LogWithWriters) packageExtensionCommandOption {
	return func(config *packageExtensionCommandConfig) {
		config.logger = logger
	}
}

func withExtensionPackageConfigReader(reader *fakes.FakePackageConfigReader) packageExtensionCommandOption {
	return func(config *packageExtensionCommandConfig) {
		config.packageConfigReader = reader
	}
}

func withExtensionPackager(creator *fakes.FakeBuildpackPackager) packageExtensionCommandOption {
	return func(config *packageExtensionCommandConfig) {
		config.extensionPackager = creator
	}
}

func withExtensionImageName(name string) packageExtensionCommandOption {
	return func(config *packageExtensionCommandConfig) {
		config.imageName = name
	}
}

func withExtensionPackageConfigPath(path string) packageExtensionCommandOption {
	return func(config *packageExtensionCommandConfig) {
		config.configPath = path
	}
}

func withExtensionClientConfig(clientCfg config.Config) packageExtensionCommandOption {
	return func(config *packageExtensionCommandConfig) {
		config.clientConfig = clientCfg
	}
}
