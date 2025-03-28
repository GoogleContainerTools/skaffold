package commands_test

import (
	"bytes"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/heroku/color"
	"github.com/pkg/errors"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/commands"
	"github.com/buildpacks/pack/internal/commands/testmocks"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestManifestPushCommand(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)

	spec.Run(t, "Commands", testManifestPushCommand, spec.Random(), spec.Report(report.Terminal{}))
}

func testManifestPushCommand(t *testing.T, when spec.G, it spec.S) {
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

		command = commands.ManifestPush(logger, mockClient)
	})

	when("args are valid", func() {
		var indexRepoName string
		it.Before(func() {
			indexRepoName = h.NewRandomIndexRepoName()
		})

		when("index exists", func() {
			when("no extra flag is provided", func() {
				it.Before(func() {
					mockClient.EXPECT().
						PushManifest(gomock.Eq(client.PushManifestOptions{
							IndexRepoName: indexRepoName,
							Format:        types.OCIImageIndex,
							Insecure:      false,
							Purge:         false,
						})).Return(nil)
				})

				it("should call push operation with default configuration", func() {
					command.SetArgs([]string{indexRepoName})
					h.AssertNil(t, command.Execute())
				})
			})

			when("--format is docker", func() {
				it.Before(func() {
					mockClient.EXPECT().
						PushManifest(gomock.Eq(client.PushManifestOptions{
							IndexRepoName: indexRepoName,
							Format:        types.DockerManifestList,
							Insecure:      false,
							Purge:         false,
						})).Return(nil)
				})

				it("should call push operation with docker media type", func() {
					command.SetArgs([]string{indexRepoName, "-f", "docker"})
					h.AssertNil(t, command.Execute())
				})
			})

			when("--purge", func() {
				it.Before(func() {
					mockClient.EXPECT().
						PushManifest(gomock.Eq(client.PushManifestOptions{
							IndexRepoName: indexRepoName,
							Format:        types.OCIImageIndex,
							Insecure:      false,
							Purge:         true,
						})).Return(nil)
				})

				it("should call push operation with purge enabled", func() {
					command.SetArgs([]string{indexRepoName, "--purge"})
					h.AssertNil(t, command.Execute())
				})
			})

			when("--insecure", func() {
				it.Before(func() {
					mockClient.EXPECT().
						PushManifest(gomock.Eq(client.PushManifestOptions{
							IndexRepoName: indexRepoName,
							Format:        types.OCIImageIndex,
							Insecure:      true,
							Purge:         false,
						})).Return(nil)
				})

				it("should call push operation with insecure enabled", func() {
					command.SetArgs([]string{indexRepoName, "--insecure"})
					h.AssertNil(t, command.Execute())
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

		when("index doesn't exist", func() {
			it.Before(func() {
				mockClient.
					EXPECT().
					PushManifest(
						gomock.Any(),
					).
					AnyTimes().
					Return(errors.New("unable to push Image"))
			})

			it("should return an error when index not exists locally", func() {
				command.SetArgs([]string{"some-index"})
				err := command.Execute()
				h.AssertNotNil(t, err)
			})
		})
	})

	when("args are invalid", func() {
		when("--format is invalid", func() {
			it("should return an error when index not exists locally", func() {
				command.SetArgs([]string{"some-index", "-f", "bad-media-type"})
				err := command.Execute()
				h.AssertNotNil(t, err)
				h.AssertError(t, err, "invalid media type format")
			})
		})
	})
}
