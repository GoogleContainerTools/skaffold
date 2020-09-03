package ui

import (
	"io"
	"io/ioutil"
)

var (
	out io.Writer = ioutil.Discard
)

// SetOutput sets the destination for messages from the ui package
func SetOutput(w io.Writer) {
	out = w
}
