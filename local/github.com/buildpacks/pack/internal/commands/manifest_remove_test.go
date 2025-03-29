package commands_test

import (
	"bytes"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/heroku/color"
	"github.com/pkg/errors"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/commands"
	"github.com/buildpacks/pack/internal/commands/testmocks"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestManifestDeleteCommand(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)

	spec.Run(t, "Commands", testManifestDeleteCommand, spec.Random(), spec.Report(report.Terminal{}))
}

func testManifestDeleteCommand(t *testing.T, when spec.G, it spec.S) {
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
		command = commands.ManifestDelete(logger, mockClient)
	})

	when("args are valid", func() {
		var indexRepoName string
		it.Before(func() {
			indexRepoName = h.NewRandomIndexRepoName()
		})

		when("index exists", func() {
			when("no extra flags are provided", func() {
				it.Before(func() {
					mockClient.EXPECT().DeleteManifest(
						gomock.Eq([]string{indexRepoName}),
					).Return(nil)
				})

				it("should delete index", func() {
					command.SetArgs([]string{indexRepoName})
					h.AssertNil(t, command.Execute())
				})
			})

			when("--help", func() {
				it("should have help flag", func() {
					command.SetArgs([]string{"--help"})
					h.AssertNil(t, command.Execute())
				})
			})
		})

		when("index does not exist", func() {
			it.Before(func() {
				mockClient.EXPECT().DeleteManifest(
					gomock.Eq([]string{"some-none-existent-index"}),
				).Return(errors.New("image index doesn't exists"))
			})

			it("should return an error", func() {
				command.SetArgs([]string{"some-none-existent-index"})
				h.AssertNotNil(t, command.Execute())
			})
		})
	})
}
