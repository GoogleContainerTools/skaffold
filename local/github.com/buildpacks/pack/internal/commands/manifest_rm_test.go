package commands_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/commands"
	"github.com/buildpacks/pack/internal/commands/testmocks"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestManifestRemoveCommand(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)

	spec.Run(t, "Commands", testManifestRemoveCommand, spec.Random(), spec.Report(report.Terminal{}))
}

func testManifestRemoveCommand(t *testing.T, when spec.G, it spec.S) {
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
		command = commands.ManifestRemove(logger, mockClient)
	})
	when("args are valid", func() {
		var indexRepoName string
		it.Before(func() {
			indexRepoName = h.NewRandomIndexRepoName()
		})

		when("index exists", func() {
			when("no extra flags are provided", func() {
				it.Before(func() {
					mockClient.EXPECT().RemoveManifest(
						gomock.Eq(indexRepoName),
						gomock.Eq([]string{"some-image"}),
					).Return(nil)
				})
				it("should remove index", func() {
					command.SetArgs([]string{indexRepoName, "some-image"})
					h.AssertNil(t, command.Execute())
				})
			})

			when("--help", func() {
				it("should have help flag", func() {
					command.SetArgs([]string{"--help"})
					h.AssertNil(t, command.Execute())
					h.AssertEq(t, outBuf.String(), "")
				})
			})
		})

		when("index does not exist", func() {
			it.Before(func() {
				mockClient.EXPECT().RemoveManifest(
					gomock.Eq(indexRepoName),
					gomock.Eq([]string{"some-image"}),
				).Return(errors.New("image index doesn't exists"))
			})
			it("should return an error", func() {
				command.SetArgs([]string{indexRepoName, "some-image"})
				h.AssertNotNil(t, command.Execute())
			})
		})
	})

	when("args are invalid", func() {
		it("errors when missing mandatory arguments", func() {
			command.SetArgs([]string{"some-index"})
			err := command.Execute()
			h.AssertNotNil(t, err)
			h.AssertError(t, err, "requires at least 2 arg(s), only received 1")
		})
	})
}
