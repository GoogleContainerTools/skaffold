package commands_test

import (
	"bytes"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/commands"
	"github.com/buildpacks/pack/internal/commands/testmocks"
	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestYankCommand(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "YankCommand", testYankCommand, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testYankCommand(t *testing.T, when spec.G, it spec.S) {
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

		cmd = commands.BuildpackYank(logger, cfg, mockClient)
	})

	when("#YankBuildpackCommand", func() {
		when("no buildpack id@version is provided", func() {
			it("fails to run", func() {
				err := cmd.Execute()
				h.AssertError(t, err, "accepts 1 arg")
			})
		})

		when("id@version argument is provided", func() {
			var (
				buildpackIDVersion string
			)

			it.Before(func() {
				buildpackIDVersion = "heroku/rust@0.0.1"
			})

			it("should work for required args", func() {
				opts := client.YankBuildpackOptions{
					ID:      "heroku/rust",
					Version: "0.0.1",
					Type:    "github",
					URL:     "https://github.com/buildpacks/registry-index",
					Yank:    true,
				}

				mockClient.EXPECT().
					YankBuildpack(opts).
					Return(nil)

				cmd.SetArgs([]string{buildpackIDVersion})
				h.AssertNil(t, cmd.Execute())
			})

			it("should fail for invalid buildpack id/version", func() {
				cmd.SetArgs([]string{"mybuildpack"})
				err := cmd.Execute()

				h.AssertError(t, err, "invalid buildpack id@version 'mybuildpack'")
			})

			it("should use the default registry defined in config.toml", func() {
				cfg = config.Config{
					DefaultRegistryName: "official",
					Registries: []config.Registry{
						{
							Name: "official",
							Type: "github",
							URL:  "https://github.com/buildpacks/registry-index",
						},
					},
				}

				opts := client.YankBuildpackOptions{
					ID:      "heroku/rust",
					Version: "0.0.1",
					Type:    "github",
					URL:     "https://github.com/buildpacks/registry-index",
					Yank:    true,
				}

				mockClient.EXPECT().
					YankBuildpack(opts).
					Return(nil)

				cmd.SetArgs([]string{buildpackIDVersion})
				h.AssertNil(t, cmd.Execute())
			})

			it("should undo", func() {
				opts := client.YankBuildpackOptions{
					ID:      "heroku/rust",
					Version: "0.0.1",
					Type:    "github",
					URL:     "https://github.com/buildpacks/registry-index",
					Yank:    false,
				}
				mockClient.EXPECT().
					YankBuildpack(opts).
					Return(nil)

				cmd.SetArgs([]string{buildpackIDVersion, "--undo"})
				h.AssertNil(t, cmd.Execute())
			})

			when("buildpack-registry flag is used", func() {
				it("should use the specified buildpack registry", func() {
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
					opts := client.YankBuildpackOptions{
						ID:      "heroku/rust",
						Version: "0.0.1",
						Type:    "github",
						URL:     "https://github.com/override/buildpack-registry",
						Yank:    true,
					}
					mockClient.EXPECT().
						YankBuildpack(opts).
						Return(nil)

					cmd = commands.BuildpackYank(logger, cfg, mockClient)
					cmd.SetArgs([]string{buildpackIDVersion, "--buildpack-registry", buildpackRegistry})
					h.AssertNil(t, cmd.Execute())
				})

				it("should handle config errors", func() {
					cfg = config.Config{
						DefaultRegistryName: "missing registry",
					}
					cmd = commands.BuildpackYank(logger, cfg, mockClient)
					cmd.SetArgs([]string{buildpackIDVersion})

					err := cmd.Execute()
					h.AssertNotNil(t, err)
				})
			})
		})
	})
}
