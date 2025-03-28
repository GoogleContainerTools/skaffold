package commands_test

import (
	"bytes"
	"testing"

	"github.com/buildpacks/pack/internal/commands"
	"github.com/buildpacks/pack/pkg/client"

	"github.com/golang/mock/gomock"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/commands/testmocks"
	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestRegisterCommand(t *testing.T) {
	spec.Run(t, "RegisterCommand", testRegisterCommand, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testRegisterCommand(t *testing.T, when spec.G, it spec.S) {
	var (
		cmd            *cobra.Command
		logger         logging.Logger
		outBuf         bytes.Buffer
		mockController *gomock.Controller
		mockClient     *testmocks.MockPackClient
		cfg            config.Config
	)

	it.Before(func() {
		logger = logging.NewLogWithWriters(&outBuf, &outBuf)
		mockController = gomock.NewController(t)
		mockClient = testmocks.NewMockPackClient(mockController)
		cfg = config.Config{}

		cmd = commands.BuildpackRegister(logger, cfg, mockClient)
	})

	it.After(func() {})

	when("#RegisterBuildpackCommand", func() {
		when("no image is provided", func() {
			it("fails to run", func() {
				err := cmd.Execute()
				h.AssertError(t, err, "accepts 1 arg")
			})
		})

		when("image name is provided", func() {
			var buildpackImage = "buildpack/image"

			it("should work for required args", func() {
				opts := client.RegisterBuildpackOptions{
					ImageName: buildpackImage,
					Type:      "github",
					URL:       "https://github.com/buildpacks/registry-index",
					Name:      "official",
				}

				mockClient.EXPECT().
					RegisterBuildpack(gomock.Any(), opts).
					Return(nil)

				cmd.SetArgs([]string{buildpackImage})
				h.AssertNil(t, cmd.Execute())
			})

			when("config.toml exists", func() {
				it("should consume registry config values", func() {
					cfg = config.Config{
						DefaultRegistryName: "berneuse",
						Registries: []config.Registry{
							{
								Name: "berneuse",
								Type: "github",
								URL:  "https://github.com/berneuse/buildpack-registry",
							},
						},
					}
					cmd = commands.BuildpackRegister(logger, cfg, mockClient)
					opts := client.RegisterBuildpackOptions{
						ImageName: buildpackImage,
						Type:      "github",
						URL:       "https://github.com/berneuse/buildpack-registry",
						Name:      "berneuse",
					}

					mockClient.EXPECT().
						RegisterBuildpack(gomock.Any(), opts).
						Return(nil)

					cmd.SetArgs([]string{buildpackImage})
					h.AssertNil(t, cmd.Execute())
				})

				it("should handle config errors", func() {
					cfg = config.Config{
						DefaultRegistryName: "missing registry",
					}
					cmd = commands.BuildpackRegister(logger, cfg, mockClient)
					cmd.SetArgs([]string{buildpackImage})

					err := cmd.Execute()
					h.AssertNotNil(t, err)
				})
			})

			it("should support buildpack-registry flag", func() {
				buildpackRegistry := "override"
				cfg = config.Config{
					DefaultRegistryName: "default",
					Registries: []config.Registry{
						{
							Name: "default",
							Type: "github",
							URL:  "https://github.com/default/buildpack-registry",
						},
						{
							Name: "override",
							Type: "github",
							URL:  "https://github.com/override/buildpack-registry",
						},
					},
				}
				opts := client.RegisterBuildpackOptions{
					ImageName: buildpackImage,
					Type:      "github",
					URL:       "https://github.com/override/buildpack-registry",
					Name:      "override",
				}
				mockClient.EXPECT().
					RegisterBuildpack(gomock.Any(), opts).
					Return(nil)

				cmd = commands.BuildpackRegister(logger, cfg, mockClient)
				cmd.SetArgs([]string{buildpackImage, "--buildpack-registry", buildpackRegistry})
				h.AssertNil(t, cmd.Execute())
			})
		})
	})
}
