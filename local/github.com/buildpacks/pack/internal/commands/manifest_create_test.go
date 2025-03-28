package commands_test

import (
	"bytes"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/commands"
	"github.com/buildpacks/pack/internal/commands/testmocks"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestManifestCreateCommand(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)

	spec.Run(t, "Commands", testManifestCreateCommand, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testManifestCreateCommand(t *testing.T, when spec.G, it spec.S) {
	var (
		command        *cobra.Command
		logger         *logging.LogWithWriters
		outBuf         bytes.Buffer
		mockController *gomock.Controller
		mockClient     *testmocks.MockPackClient
	)

	it.Before(func() {
		logger = logging.NewLogWithWriters(&outBuf, &outBuf)
		mockController = gomock.NewController(t)
		mockClient = testmocks.NewMockPackClient(mockController)

		command = commands.ManifestCreate(logger, mockClient)
	})

	when("args are valid", func() {
		var indexRepoName string
		it.Before(func() {
			indexRepoName = h.NewRandomIndexRepoName()
		})

		when("index exists", func() {
			when("no extra flags are provided", func() {
				it.Before(func() {
					mockClient.
						EXPECT().
						CreateManifest(gomock.Any(),
							client.CreateManifestOptions{
								IndexRepoName: indexRepoName,
								RepoNames:     []string{"some-manifest"},
								Format:        types.OCIImageIndex,
								Insecure:      false,
								Publish:       false,
							},
						).Return(nil)
				})

				it("should call create operation with default configuration", func() {
					command.SetArgs([]string{indexRepoName, "some-manifest"})
					h.AssertNil(t, command.Execute())
				})
			})

			when("--format is docker", func() {
				it.Before(func() {
					mockClient.
						EXPECT().
						CreateManifest(gomock.Any(),
							client.CreateManifestOptions{
								IndexRepoName: indexRepoName,
								RepoNames:     []string{"some-manifest"},
								Format:        types.DockerManifestList,
								Insecure:      false,
								Publish:       false,
							},
						).Return(nil)
				})

				it("should call create operation with docker media type", func() {
					command.SetArgs([]string{indexRepoName, "some-manifest", "-f", "docker"})
					h.AssertNil(t, command.Execute())
				})
			})

			when("--publish", func() {
				when("--insecure", func() {
					it.Before(func() {
						mockClient.
							EXPECT().
							CreateManifest(gomock.Any(),
								client.CreateManifestOptions{
									IndexRepoName: indexRepoName,
									RepoNames:     []string{"some-manifest"},
									Format:        types.OCIImageIndex,
									Insecure:      true,
									Publish:       true,
								},
							).Return(nil)
					})

					it("should call create operation with publish and insecure", func() {
						command.SetArgs([]string{indexRepoName, "some-manifest", "--publish", "--insecure"})
						h.AssertNil(t, command.Execute())
					})
				})

				when("no --insecure", func() {
					it.Before(func() {
						mockClient.
							EXPECT().
							CreateManifest(gomock.Any(),
								client.CreateManifestOptions{
									IndexRepoName: indexRepoName,
									RepoNames:     []string{"some-manifest"},
									Format:        types.OCIImageIndex,
									Insecure:      false,
									Publish:       true,
								},
							).Return(nil)
					})

					it("should call create operation with publish", func() {
						command.SetArgs([]string{indexRepoName, "some-manifest", "--publish"})
						h.AssertNil(t, command.Execute())
					})
				})
			})

			when("--help", func() {
				it("should have help flag", func() {
					command.SetArgs([]string{"--help"})
					h.AssertNilE(t, command.Execute())
					h.AssertEq(t, outBuf.String(), "")
				})
			})
		})
	})

	when("invalid arguments", func() {
		when("--insecure is used without publish", func() {
			it("errors a message", func() {
				command.SetArgs([]string{"something", "some-manifest", "--insecure"})
				h.AssertError(t, command.Execute(), "insecure flag requires the publish flag")
			})
		})

		when("invalid media type", func() {
			var format string
			it.Before(func() {
				format = "invalid"
			})

			it("errors a message", func() {
				command.SetArgs([]string{"some-index", "some-manifest", "--format", format})
				h.AssertNotNil(t, command.Execute())
			})
		})
	})
}
