package commands_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/commands"
	"github.com/buildpacks/pack/internal/commands/testmocks"
	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestCreateBuilderCommand(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "CreateBuilderCommand", testCreateBuilderCommand, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testCreateBuilderCommand(t *testing.T, when spec.G, it spec.S) {
	var (
		command           *cobra.Command
		logger            logging.Logger
		outBuf            bytes.Buffer
		mockController    *gomock.Controller
		mockClient        *testmocks.MockPackClient
		tmpDir            string
		builderConfigPath string
		cfg               config.Config
	)

	it.Before(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "create-builder-test")
		h.AssertNil(t, err)
		builderConfigPath = filepath.Join(tmpDir, "builder.toml")
		cfg = config.Config{}

		mockController = gomock.NewController(t)
		mockClient = testmocks.NewMockPackClient(mockController)
		logger = logging.NewLogWithWriters(&outBuf, &outBuf)
		command = commands.CreateBuilder(logger, cfg, mockClient)
	})

	it.After(func() {
		mockController.Finish()
	})

	when("#CreateBuilder", func() {
		it("gives deprecation warning", func() {
			h.AssertNil(t, os.WriteFile(builderConfigPath, []byte(validConfig), 0666))
			mockClient.EXPECT().CreateBuilder(gomock.Any(), gomock.Any()).Return(nil)
			command.SetArgs([]string{
				"some/builder",
				"--config", builderConfigPath,
			})

			h.AssertNil(t, command.Execute())

			h.AssertContains(t, outBuf.String(), "Warning: Command 'pack create-builder' has been deprecated, please use 'pack builder create' instead")
		})

		when("both --publish and pull-policy=never flags are specified", func() {
			it("errors with a descriptive message", func() {
				command.SetArgs([]string{
					"some/builder",
					"--config", "some-config-path",
					"--publish",
					"--pull-policy",
					"never",
				})
				err := command.Execute()
				h.AssertNotNil(t, err)
				h.AssertError(t, err, "--publish and --pull-policy never cannot be used together. The --publish flag requires the use of remote images.")
			})
		})

		when("--pull-policy", func() {
			it("returns error for unknown policy", func() {
				command.SetArgs([]string{
					"some/builder",
					"--config", builderConfigPath,
					"--pull-policy", "unknown-policy",
				})
				h.AssertError(t, command.Execute(), "parsing pull policy")
			})
		})

		when("--pull-policy is not specified", func() {
			when("configured pull policy is invalid", func() {
				it("returns error for when config set with unknown policy", func() {
					cfg = config.Config{PullPolicy: "unknown-policy"}
					command = commands.BuilderCreate(logger, cfg, mockClient)
					command.SetArgs([]string{
						"some/builder",
						"--config", builderConfigPath,
					})
					h.AssertError(t, command.Execute(), "parsing pull policy")
				})
			})
		})

		when("--buildpack-registry flag is specified but experimental isn't set in the config", func() {
			it("errors with a descriptive message", func() {
				command.SetArgs([]string{
					"some/builder",
					"--config", "some-config-path",
					"--buildpack-registry", "some-registry",
				})
				err := command.Execute()
				h.AssertNotNil(t, err)
				h.AssertError(t, err, "Support for buildpack registries is currently experimental.")
			})
		})

		when("warnings encountered in builder.toml", func() {
			it.Before(func() {
				h.AssertNil(t, os.WriteFile(builderConfigPath, []byte(`
[[buildpacks]]
  id = "some.buildpack"
`), 0666))
			})

			it("logs the warnings", func() {
				mockClient.EXPECT().CreateBuilder(gomock.Any(), gomock.Any()).Return(nil)

				command.SetArgs([]string{
					"some/builder",
					"--config", builderConfigPath,
				})
				h.AssertNil(t, command.Execute())

				h.AssertContains(t, outBuf.String(), "Warning: builder configuration: empty 'order' definition")
			})
		})

		when("uses --builder-config", func() {
			it.Before(func() {
				h.AssertNil(t, os.WriteFile(builderConfigPath, []byte(validConfig), 0666))
			})

			it("errors with a descriptive message", func() {
				command.SetArgs([]string{
					"some/builder",
					"--builder-config", builderConfigPath,
				})
				h.AssertError(t, command.Execute(), "unknown flag: --builder-config")
			})
		})

		when("no config provided", func() {
			it("errors with a descriptive message", func() {
				command.SetArgs([]string{
					"some/builder",
				})
				h.AssertError(t, command.Execute(), "Please provide a builder config path")
			})
		})

		when("builder config has extensions but experimental isn't set in the config", func() {
			it.Before(func() {
				h.AssertNil(t, os.WriteFile(builderConfigPath, []byte(validConfigWithExtensions), 0666))
			})

			it("errors", func() {
				command.SetArgs([]string{
					"some/builder",
					"--config", builderConfigPath,
				})
				h.AssertError(t, command.Execute(), "builder config contains image extensions; support for image extensions is currently experimental")
			})
		})
	})
}
