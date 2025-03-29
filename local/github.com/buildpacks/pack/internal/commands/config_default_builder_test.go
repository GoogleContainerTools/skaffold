package commands_test

import (
	"bytes"
	"fmt"
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
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestConfigDefaultBuilder(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "ConfigDefaultBuilderCommand", testConfigDefaultBuilder, spec.Random(), spec.Report(report.Terminal{}))
}

func testConfigDefaultBuilder(t *testing.T, when spec.G, it spec.S) {
	var (
		cmd            *cobra.Command
		logger         logging.Logger
		outBuf         bytes.Buffer
		mockController *gomock.Controller
		mockClient     *testmocks.MockPackClient
		tempPackHome   string
		configPath     string
	)

	it.Before(func() {
		var err error

		mockController = gomock.NewController(t)
		mockClient = testmocks.NewMockPackClient(mockController)
		logger = logging.NewLogWithWriters(&outBuf, &outBuf)
		tempPackHome, err = os.MkdirTemp("", "pack-home")
		h.AssertNil(t, err)
		configPath = filepath.Join(tempPackHome, "config.toml")
		cmd = commands.ConfigDefaultBuilder(logger, config.Config{}, configPath, mockClient)
	})

	it.After(func() {
		mockController.Finish()
		h.AssertNil(t, os.RemoveAll(tempPackHome))
	})

	when("#ConfigDefaultBuilder", func() {
		when("no args", func() {
			it("lists current default builder if one is set", func() {
				cmd = commands.ConfigDefaultBuilder(logger, config.Config{DefaultBuilder: "some/builder"}, configPath, mockClient)
				cmd.SetArgs([]string{})
				h.AssertNil(t, cmd.Execute())
				h.AssertContains(t, outBuf.String(), "some/builder")
			})

			it("suggests setting a builder if none is set", func() {
				cmd.SetArgs([]string{})
				h.AssertNil(t, cmd.Execute())
				h.AssertContains(t, outBuf.String(), "No default builder is set.")
				h.AssertContains(t, outBuf.String(), "run `pack builder suggest`")
			})
		})

		when("unset", func() {
			it("unsets current default builder", func() {
				cfg := config.Config{DefaultBuilder: "some/builder"}
				h.AssertNil(t, config.Write(cfg, configPath))
				cmd = commands.ConfigDefaultBuilder(logger, cfg, configPath, mockClient)
				cmd.SetArgs([]string{"--unset"})
				err := cmd.Execute()
				h.AssertNil(t, err)
				cfg, err = config.Read(configPath)
				h.AssertNil(t, err)
				h.AssertEq(t, cfg.DefaultBuilder, "")
				h.AssertContains(t, outBuf.String(), fmt.Sprintf("Successfully unset default builder %s", style.Symbol("some/builder")))
			})

			it("clarifies if no builder was set", func() {
				cmd.SetArgs([]string{"--unset"})
				h.AssertNil(t, cmd.Execute())
				h.AssertContains(t, outBuf.String(), "No default builder was set")
			})

			it("gives clear error if unable to write to config", func() {
				h.AssertNil(t, os.WriteFile(configPath, []byte("some-data"), 0001))
				cmd = commands.ConfigDefaultBuilder(logger, config.Config{DefaultBuilder: "some/builder"}, configPath, mockClient)
				cmd.SetArgs([]string{"--unset"})
				err := cmd.Execute()
				h.AssertError(t, err, "failed to write to config at "+configPath)
			})
		})

		when("set", func() {
			when("valid builder is provider", func() {
				when("in local", func() {
					var imageName = "some/image"

					it("sets default builder", func() {
						mockClient.EXPECT().InspectBuilder(imageName, true).Return(&client.BuilderInfo{
							Stack: "test.stack.id",
						}, nil)

						cmd.SetArgs([]string{imageName})
						h.AssertNil(t, cmd.Execute())
						h.AssertContains(t, outBuf.String(), fmt.Sprintf("Builder '%s' is now the default builder", imageName))

						cfg, err := config.Read(configPath)
						h.AssertNil(t, err)
						h.AssertEq(t, cfg.DefaultBuilder, "some/image")
					})

					it("gives clear error if unable to write to config", func() {
						h.AssertNil(t, os.WriteFile(configPath, []byte("some-data"), 0001))
						mockClient.EXPECT().InspectBuilder(imageName, true).Return(&client.BuilderInfo{
							Stack: "test.stack.id",
						}, nil)
						cmd = commands.ConfigDefaultBuilder(logger, config.Config{}, configPath, mockClient)
						cmd.SetArgs([]string{imageName})
						err := cmd.Execute()
						h.AssertError(t, err, "failed to write to config at "+configPath)
					})
				})

				when("in remote", func() {
					it("sets default builder", func() {
						imageName := "some/image"

						localCall := mockClient.EXPECT().InspectBuilder(imageName, true).Return(nil, nil)

						mockClient.EXPECT().InspectBuilder(imageName, false).Return(&client.BuilderInfo{
							Stack: "test.stack.id",
						}, nil).After(localCall)

						cmd.SetArgs([]string{imageName})
						h.AssertNil(t, cmd.Execute())
						h.AssertContains(t, outBuf.String(), fmt.Sprintf("Builder '%s' is now the default builder", imageName))
					})

					it("gives clear error if unable to inspect remote image", func() {
						imageName := "some/image"

						localCall := mockClient.EXPECT().InspectBuilder(imageName, true).Return(nil, nil)

						mockClient.EXPECT().InspectBuilder(imageName, false).Return(&client.BuilderInfo{
							Stack: "test.stack.id",
						}, client.SoftError{}).After(localCall)

						cmd.SetArgs([]string{imageName})
						err := cmd.Execute()
						h.AssertError(t, err, fmt.Sprintf("failed to inspect remote image %s", style.Symbol(imageName)))
					})
				})
			})

			when("invalid builder is provided", func() {
				it("error is presented", func() {
					imageName := "nonbuilder/image"

					mockClient.EXPECT().InspectBuilder(imageName, true).Return(
						nil,
						fmt.Errorf("failed to inspect image %s", imageName))

					cmd.SetArgs([]string{imageName})

					h.AssertNotNil(t, cmd.Execute())
					h.AssertContains(t, outBuf.String(), fmt.Sprintf("validating that builder %s exists", style.Symbol("nonbuilder/image")))
				})
			})

			when("non-existent builder is provided", func() {
				it("error is present", func() {
					imageName := "nonexisting/image"

					localCall := mockClient.EXPECT().InspectBuilder(imageName, true).Return(
						nil,
						nil)

					mockClient.EXPECT().InspectBuilder(imageName, false).Return(
						nil,
						nil).After(localCall)

					cmd.SetArgs([]string{imageName})

					h.AssertNotNil(t, cmd.Execute())
					h.AssertContains(t, outBuf.String(), "builder 'nonexisting/image' not found")
				})
			})
		})
	})
}
