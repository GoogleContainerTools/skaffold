package commands_test

import (
	"bytes"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/commands"
	"github.com/buildpacks/pack/internal/commands/fakes"
	"github.com/buildpacks/pack/internal/commands/testmocks"
	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestExtensionCommand(t *testing.T) {
	spec.Run(t, "ExtensionCommand", testExtensionCommand, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testExtensionCommand(t *testing.T, when spec.G, it spec.S) {
	var (
		cmd        *cobra.Command
		logger     logging.Logger
		outBuf     bytes.Buffer
		mockClient *testmocks.MockPackClient
	)

	it.Before(func() {
		logger = logging.NewLogWithWriters(&outBuf, &outBuf)
		mockController := gomock.NewController(t)
		mockClient = testmocks.NewMockPackClient(mockController)
		cmd = commands.NewExtensionCommand(logger, config.Config{}, mockClient, fakes.NewFakePackageConfigReader())
		cmd.SetOut(logging.GetWriterForLevel(logger, logging.InfoLevel))
	})

	when("extension", func() {
		it("prints help text", func() {
			cmd.SetArgs([]string{})
			h.AssertNil(t, cmd.Execute())
			output := outBuf.String()
			h.AssertContains(t, output, "Interact with extensions")
			for _, command := range []string{"Usage", "package", "register", "yank", "pull", "inspect"} {
				h.AssertContains(t, output, command)
			}
		})
	})
}
