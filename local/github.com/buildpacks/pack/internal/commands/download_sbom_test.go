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
	cpkg "github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestDownloadSBOMCommand(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "DownloadSBOMCommand", testDownloadSBOMCommand, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testDownloadSBOMCommand(t *testing.T, when spec.G, it spec.S) {
	var (
		command        *cobra.Command
		logger         logging.Logger
		outBuf         bytes.Buffer
		mockController *gomock.Controller
		mockClient     *testmocks.MockPackClient
	)

	it.Before(func() {
		mockController = gomock.NewController(t)
		mockClient = testmocks.NewMockPackClient(mockController)
		logger = logging.NewLogWithWriters(&outBuf, &outBuf)
		command = commands.DownloadSBOM(logger, mockClient)
	})

	it.After(func() {
		mockController.Finish()
	})

	when("#DownloadSBOM", func() {
		when("happy path", func() {
			it("returns no error", func() {
				mockClient.EXPECT().DownloadSBOM("some/image", cpkg.DownloadSBOMOptions{
					Daemon:         true,
					DestinationDir: ".",
				})
				command.SetArgs([]string{"some/image"})

				err := command.Execute()
				h.AssertNil(t, err)
			})
		})

		when("the remote flag is specified", func() {
			it("respects the remote flag", func() {
				mockClient.EXPECT().DownloadSBOM("some/image", cpkg.DownloadSBOMOptions{
					Daemon:         false,
					DestinationDir: ".",
				})
				command.SetArgs([]string{"some/image", "--remote"})

				err := command.Execute()
				h.AssertNil(t, err)
			})
		})

		when("the output-dir flag is specified", func() {
			it("respects the output-dir flag", func() {
				mockClient.EXPECT().DownloadSBOM("some/image", cpkg.DownloadSBOMOptions{
					Daemon:         true,
					DestinationDir: "some-destination-dir",
				})
				command.SetArgs([]string{"some/image", "--output-dir", "some-destination-dir"})

				err := command.Execute()
				h.AssertNil(t, err)
			})
		})

		when("the client returns an error", func() {
			it("returns the error", func() {
				mockClient.EXPECT().DownloadSBOM("some/image", cpkg.DownloadSBOMOptions{
					Daemon:         true,
					DestinationDir: ".",
				}).Return(errors.New("some-error"))

				command.SetArgs([]string{"some/image"})

				err := command.Execute()
				h.AssertError(t, err, "some-error")
			})
		})
	})
}
