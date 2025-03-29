package term_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/term"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestTerm(t *testing.T) {
	spec.Run(t, "Term", testTerm, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testTerm(t *testing.T, when spec.G, it spec.S) {
	when("#IsTerminal", func() {
		it("returns false for a pipe", func() {
			r, _, _ := os.Pipe()
			fd, isTerm := term.IsTerminal(r)
			h.AssertFalse(t, isTerm)
			h.AssertNotEq(t, fd, term.InvalidFileDescriptor) // The mock writer is a pipe, and therefore has a file descriptor
		})

		it("returns InvalidFileDescriptor if passed a normal Writer", func() {
			fd, isTerm := term.IsTerminal(&bytes.Buffer{})
			h.AssertFalse(t, isTerm)
			h.AssertEq(t, fd, term.InvalidFileDescriptor)
		})
	})
}
