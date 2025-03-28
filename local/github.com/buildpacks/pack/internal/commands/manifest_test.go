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
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestNewManifestCommand(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)

	spec.Run(t, "Commands", testNewManifestCommand, spec.Random(), spec.Report(report.Terminal{}))
}

func testNewManifestCommand(t *testing.T, when spec.G, it spec.S) {
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

		command = commands.NewManifestCommand(logger, mockClient)
		command.SetOut(logging.GetWriterForLevel(logger, logging.InfoLevel))
	})
	it("should have help flag", func() {
		command.SetArgs([]string{})
		err := command.Execute()
		h.AssertNilE(t, err)

		output := outBuf.String()
		h.AssertContains(t, output, "Usage:")
		for _, command := range []string{"create", "add", "annotate", "inspect", "remove", "rm"} {
			h.AssertContains(t, output, command)
		}
	})
}
