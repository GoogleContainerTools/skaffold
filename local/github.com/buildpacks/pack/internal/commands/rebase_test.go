package commands_test

import (
	"bytes"
	"testing"

	"github.com/heroku/color"

	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/image"

	"github.com/golang/mock/gomock"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/commands"
	"github.com/buildpacks/pack/internal/commands/testmocks"
	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestRebaseCommand(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)

	spec.Run(t, "Commands", testRebaseCommand, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testRebaseCommand(t *testing.T, when spec.G, it spec.S) {
	var (
		command        *cobra.Command
		logger         logging.Logger
		outBuf         bytes.Buffer
		mockController *gomock.Controller
		mockClient     *testmocks.MockPackClient
		cfg            config.Config
	)

	it.Before(func() {
		logger = logging.NewLogWithWriters(&outBuf, &outBuf)
		cfg = config.Config{}
		mockController = gomock.NewController(t)
		mockClient = testmocks.NewMockPackClient(mockController)

		command = commands.Rebase(logger, cfg, mockClient)
	})

	when("#RebaseCommand", func() {
		when("no image is provided", func() {
			it("fails to run", func() {
				err := command.Execute()
				h.AssertError(t, err, "accepts 1 arg")
			})
		})

		when("image name is provided", func() {
			var (
				repoName string
				opts     client.RebaseOptions
			)
			it.Before(func() {
				runImage := "test/image"
				testMirror1 := "example.com/some/run1"
				testMirror2 := "example.com/some/run2"

				cfg.RunImages = []config.RunImage{{
					Image:   runImage,
					Mirrors: []string{testMirror1, testMirror2},
				}}
				command = commands.Rebase(logger, cfg, mockClient)

				repoName = "test/repo-image"
				opts = client.RebaseOptions{
					RepoName:   repoName,
					Publish:    false,
					PullPolicy: image.PullAlways,
					RunImage:   "",
					AdditionalMirrors: map[string][]string{
						runImage: {testMirror1, testMirror2},
					},
				}
			})

			it("works", func() {
				mockClient.EXPECT().
					Rebase(gomock.Any(), opts).
					Return(nil)

				command.SetArgs([]string{repoName})
				h.AssertNil(t, command.Execute())
			})

			when("--pull-policy never", func() {
				it("works", func() {
					opts.PullPolicy = image.PullNever
					mockClient.EXPECT().
						Rebase(gomock.Any(), opts).
						Return(nil)

					command.SetArgs([]string{repoName, "--pull-policy", "never"})
					h.AssertNil(t, command.Execute())
				})
				it("takes precedence over config policy", func() {
					opts.PullPolicy = image.PullNever
					mockClient.EXPECT().
						Rebase(gomock.Any(), opts).
						Return(nil)

					cfg.PullPolicy = "if-not-present"
					command = commands.Rebase(logger, cfg, mockClient)

					command.SetArgs([]string{repoName, "--pull-policy", "never"})
					h.AssertNil(t, command.Execute())
				})
			})

			when("--pull-policy unknown-policy", func() {
				it("fails to run", func() {
					command.SetArgs([]string{repoName, "--pull-policy", "unknown-policy"})
					h.AssertError(t, command.Execute(), "parsing pull policy")
				})
			})
			when("--pull-policy not set", func() {
				when("no policy set in config", func() {
					it("uses the default policy", func() {
						opts.PullPolicy = image.PullAlways
						mockClient.EXPECT().
							Rebase(gomock.Any(), opts).
							Return(nil)

						command.SetArgs([]string{repoName})
						h.AssertNil(t, command.Execute())
					})
				})
				when("policy is set in config", func() {
					it("uses set policy", func() {
						opts.PullPolicy = image.PullIfNotPresent
						mockClient.EXPECT().
							Rebase(gomock.Any(), opts).
							Return(nil)

						cfg.PullPolicy = "if-not-present"
						command = commands.Rebase(logger, cfg, mockClient)

						command.SetArgs([]string{repoName})
						h.AssertNil(t, command.Execute())
					})
				})
				when("rebase is true", func() {
					it("passes it through", func() {
						opts.Force = true
						mockClient.EXPECT().Rebase(gomock.Any(), opts).Return(nil)
						command = commands.Rebase(logger, cfg, mockClient)
						command.SetArgs([]string{repoName, "--force"})
						h.AssertNil(t, command.Execute())
					})
				})
			})
			when("image name and previous image are provided", func() {
				var expectedOpts client.RebaseOptions

				it.Before(func() {
					runImage := "test/image"
					testMirror1 := "example.com/some/run1"
					testMirror2 := "example.com/some/run2"

					cfg.RunImages = []config.RunImage{{
						Image:   runImage,
						Mirrors: []string{testMirror1, testMirror2},
					}}
					command = commands.Rebase(logger, cfg, mockClient)

					repoName = "test/repo-image"
					previousImage := "example.com/previous-image:tag" // Example of previous image with tag
					opts := client.RebaseOptions{
						RepoName:   repoName,
						Publish:    false,
						PullPolicy: image.PullAlways,
						RunImage:   "",
						AdditionalMirrors: map[string][]string{
							runImage: {testMirror1, testMirror2},
						},
						PreviousImage: previousImage,
					}
					expectedOpts = opts
				})

				it("works", func() {
					mockClient.EXPECT().
						Rebase(gomock.Any(), gomock.Eq(expectedOpts)).
						Return(nil)

					command.SetArgs([]string{repoName, "--previous-image", "example.com/previous-image:tag"})
					h.AssertNil(t, command.Execute())
				})
			})
		})
	})
}
