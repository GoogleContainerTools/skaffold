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
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/ssh/terminal"
)

// IsTerminal will check if the specified output stream is a terminal. This can be changed
// for testing to an arbitrary method.
var IsTerminal = isTerminal

func wrapTextIfTerminal(out io.Writer, c Color, a ...interface{}) string {
	if IsTerminal(out) {
		return c.Sprint(a...)
	}
	return fmt.Sprint(a...)
}

// Fprint wraps the operands in the color ANSI escape codes, and outputs the result to
// out. If out is not a terminal, the escape codes will not be added.
// It returns the number of bytes written and any errors encountered.
func Fprint(out io.Writer, c Color, a ...interface{}) (n int, err error) {
	t := wrapTextIfTerminal(out, c, a...)
	return fmt.Fprint(out, t)
}

// Fprintln wraps the operands in the color ANSI escape codes, and outputs the result to
// out, followed by a newline. If out is not a terminal, the escape codes will not be added.
// It returns the number of bytes written and any errors encountered.
func Fprintln(out io.Writer, c Color, a ...interface{}) (n int, err error) {
	t := wrapTextIfTerminal(out, c, a...)
	return fmt.Fprintln(out, t)
}

// Fprintf applies formats according to the format specifier (and the optional interfaces provided),
// wraps the result in the color ANSI escape codes, and outputs the result to
// out, followed by a newline. If out is not a terminal, the escape codes will not be added.
// It returns the number of bytes written and any errors encountered.
func Fprintf(out io.Writer, c Color, format string, a ...interface{}) (n int, err error) {
	var t string
	if IsTerminal(out) {
		t = c.Sprintf(format, a...)
	} else {
		t = fmt.Sprintf(format, a...)
	}
	return fmt.Fprint(out, t)
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
