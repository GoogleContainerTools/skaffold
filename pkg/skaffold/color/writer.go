/*
Copyright 2018 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package color

import (
	"io"
	"os"

	"golang.org/x/crypto/ssh/terminal"
)

// Writer implements the `io.Writer` interface, allowing the caller to wrap output
// in color (via ANSI escape codes) when output is to a terminal. It will write to out, which
// in most cases will be stdout. Skaffold follows a pattern of logging to stderr via logrus
// for diagnostic output, and uses stdout for messages from Skaffold itself to the user.
type Writer struct {
	Color      Color
	Out        io.Writer
	isTerminal func(w io.Writer) bool
}

func (o *Writer) wrapTextIfTerminal(p []byte) []byte {
	if o.isTerminal(o.Out) {
		return []byte(o.Color.Sprint(string(p)))
	}
	return p
}

// NewWriter instantiates an instance of ColorWriter that will output to out in
// color, as long as out is a terminal. If out is not a terminal, the color will not be used.
func NewWriter(out io.Writer, color Color) *Writer {
	return &Writer{
		Color: color,

		Out:        out,
		isTerminal: isTerminal,
	}
}

// Write wraps the bytes to write in the color ANSI escape codes using o's Color object, and outputs
// the result to o's out object. If output is not being directed to a terminal, the escape codes will not be added.
// It returns the number of bytes written and any errors encountered.
func (o *Writer) Write(p []byte) (n int, err error) {
	t := o.wrapTextIfTerminal(p)
	return o.Out.Write(t)
}

// This implementation comes from logrus (https://github.com/sirupsen/logrus/blob/master/terminal_check_notappengine.go),
// unfortunately logrus doesn't expose a public interface we can use to call it.
func isTerminal(w io.Writer) bool {
	switch v := w.(type) {
	case *os.File:
		return terminal.IsTerminal(int(v.Fd()))
	default:
		return false
	}
}
