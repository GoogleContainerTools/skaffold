package term

import (
	"io"

	"golang.org/x/term"
)

// InvalidFileDescriptor based on https://golang.org/src/os/file_unix.go?s=2183:2210#L57
const InvalidFileDescriptor = ^(uintptr(0))

// IsTerminal returns whether a writer is a terminal
func IsTerminal(w io.Writer) (uintptr, bool) {
	if f, ok := w.(hasDescriptor); ok {
		termFd := f.Fd()
		isTerm := term.IsTerminal(int(termFd))
		return termFd, isTerm
	}

	return InvalidFileDescriptor, false
}

type hasDescriptor interface {
	Fd() uintptr
}
