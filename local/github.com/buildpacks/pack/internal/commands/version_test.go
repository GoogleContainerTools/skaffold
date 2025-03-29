package commands_test

import (
	"bytes"
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/commands"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestVersionCommand(t *testing.T) {
	spec.Run(t, "Commands", testVersionCommand, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testVersionCommand(t *testing.T, when spec.G, it spec.S) {
	var (
		command     *cobra.Command
		outBuf      bytes.Buffer
		testVersion = "1.3.4"
	)

	it.Before(func() {
		command = commands.Version(logging.NewLogWithWriters(&outBuf, &outBuf), testVersion)
	})

	when("#Version", func() {
		it("returns version", func() {
			command.SetArgs([]string{})
			h.AssertNil(t, command.Execute())
			h.AssertEq(t, outBuf.String(), testVersion+"\n")
		})
	})
}
