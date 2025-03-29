package commands

import (
	"bytes"
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestStackCommand(t *testing.T) {
	spec.Run(t, "StackCommand", testStackCommand, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testStackCommand(t *testing.T, when spec.G, it spec.S) {
	var (
		command *cobra.Command
		outBuf  bytes.Buffer
	)

	it.Before(func() {
		command = NewStackCommand(logging.NewLogWithWriters(&outBuf, &outBuf))
	})

	when("#Stack", func() {
		it("displays stack information", func() {
			command.SetArgs([]string{})
			bb := bytes.NewBufferString("") // In most tests we don't seem to need to this, not sure why it's necessary here.
			command.SetOut(bb)
			h.AssertNil(t, command.Execute())
			h.AssertEq(t, bb.String(), `(Deprecated)
Stacks are deprecated in favor of using BuildImages and RunImages directly, but will continue to be supported throughout all of 2023 and '24 if not longer. Please see our docs for more details- https://buildpacks.io/docs/concepts/components/stack

Usage:
  stack [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  suggest     (deprecated) List the recommended stacks

Flags:
  -h, --help   help for stack

Use "stack [command] --help" for more information about a command.
`)
		})
	})
}
