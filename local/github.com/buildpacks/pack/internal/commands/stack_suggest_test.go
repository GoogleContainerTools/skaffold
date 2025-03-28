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

func TestStacksSuggestCommand(t *testing.T) {
	spec.Run(t, "StacksSuggestCommand", testStacksSuggestCommand, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testStacksSuggestCommand(t *testing.T, when spec.G, it spec.S) {
	var (
		command *cobra.Command
		outBuf  bytes.Buffer
	)

	it.Before(func() {
		command = stackSuggest(logging.NewLogWithWriters(&outBuf, &outBuf))
	})

	when("#SuggestStacks", func() {
		it("displays stack information", func() {
			command.SetArgs([]string{})
			h.AssertNil(t, command.Execute())
			h.AssertEq(t, outBuf.String(), `Stacks maintained by the community:

    Stack ID: Deprecation Notice
    Description: Stacks are deprecated in favor of using BuildImages and RunImages directly, but will continue to be supported throughout all of 2023 and 2024 if not longer. Please see our docs for more details- https://buildpacks.io/docs/concepts/components/stack
    Maintainer: CNB
    Build Image: 
    Run Image: 

    Stack ID: heroku-20
    Description: The official Heroku stack based on Ubuntu 20.04
    Maintainer: Heroku
    Build Image: heroku/heroku:20-cnb-build
    Run Image: heroku/heroku:20-cnb

    Stack ID: io.buildpacks.stacks.jammy
    Description: A minimal Paketo stack based on Ubuntu 22.04
    Maintainer: Paketo Project
    Build Image: paketobuildpacks/build-jammy-base
    Run Image: paketobuildpacks/run-jammy-base

    Stack ID: io.buildpacks.stacks.jammy
    Description: A large Paketo stack based on Ubuntu 22.04
    Maintainer: Paketo Project
    Build Image: paketobuildpacks/build-jammy-full
    Run Image: paketobuildpacks/run-jammy-full

    Stack ID: io.buildpacks.stacks.jammy.static
    Description: A static Paketo stack based on Ubuntu 22.04, similar to distroless
    Maintainer: Paketo Project
    Build Image: paketobuildpacks/build-jammy-static
    Run Image: paketobuildpacks/run-jammy-static

    Stack ID: io.buildpacks.stacks.jammy.tiny
    Description: A tiny Paketo stack based on Ubuntu 22.04, similar to distroless
    Maintainer: Paketo Project
    Build Image: paketobuildpacks/build-jammy-tiny
    Run Image: paketobuildpacks/run-jammy-tiny
`)
		})
	})
}
