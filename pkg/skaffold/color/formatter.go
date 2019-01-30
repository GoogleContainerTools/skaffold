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
	"strings"

	"golang.org/x/crypto/ssh/terminal"
)

// IsTerminal will check if the specified output stream is a terminal. This can be changed
// for testing to an arbitrary method.
var IsTerminal = isTerminal

// Color can be used to format text using ANSI escape codes so it can be printed to
// the terminal in color.
type Color int

var (
	// LightRed can format text to be displayed to the terminal in light red, using ANSI escape codes.
	LightRed = Color(91)
	// LightGreen can format text to be displayed to the terminal in light green, using ANSI escape codes.
	LightGreen = Color(92)
	// LightYellow can format text to be displayed to the terminal in light yellow, using ANSI escape codes.
	LightYellow = Color(93)
	// LightBlue can format text to be displayed to the terminal in light blue, using ANSI escape codes.
	LightBlue = Color(94)
	// LightPurple can format text to be displayed to the terminal in light purple, using ANSI escape codes.
	LightPurple = Color(95)
	// Red can format text to be displayed to the terminal in red, using ANSI escape codes.
	Red = Color(31)
	// Green can format text to be displayed to the terminal in green, using ANSI escape codes.
	Green = Color(32)
	// Yellow can format text to be displayed to the terminal in yellow, using ANSI escape codes.
	Yellow = Color(33)
	// Blue can format text to be displayed to the terminal in blue, using ANSI escape codes.
	Blue = Color(34)
	// Purple can format text to be displayed to the terminal in purple, using ANSI escape codes.
	Purple = Color(35)
	// Cyan can format text to be displayed to the terminal in cyan, using ANSI escape codes.
	Cyan = Color(36)
	// None uses ANSI escape codes to reset all formatting.
	None = Color(0)

	// Default default output color for output from Skaffold to the user
	Default = Blue
)

// Fprint wraps the operands in c's ANSI escape codes, and outputs the result to
// out. If out is not a terminal, the escape codes will not be added.
// It returns the number of bytes written and any errors encountered.
func (c Color) Fprint(out io.Writer, a ...interface{}) (n int, err error) {
	if IsTerminal(out) {
		return fmt.Fprintf(out, "\033[%dm%s\033[0m", c, fmt.Sprint(a...))
	}
	return fmt.Fprint(out, a...)
}

// Fprintln wraps the operands in c's ANSI escape codes, and outputs the result to
// out, followed by a newline. If out is not a terminal, the escape codes will not be added.
// It returns the number of bytes written and any errors encountered.
func (c Color) Fprintln(out io.Writer, a ...interface{}) (n int, err error) {
	if IsTerminal(out) {
		return fmt.Fprintf(out, "\033[%dm%s\033[0m\n", c, strings.TrimSuffix(fmt.Sprintln(a...), "\n"))
	}
	return fmt.Fprintln(out, a...)
}

// Fprintf applies formats according to the format specifier (and the optional interfaces provided),
// wraps the result in c's ANSI escape codes, and outputs the result to
// out, followed by a newline. If out is not a terminal, the escape codes will not be added.
// It returns the number of bytes written and any errors encountered.
func (c Color) Fprintf(out io.Writer, format string, a ...interface{}) (n int, err error) {
	if IsTerminal(out) {
		return fmt.Fprintf(out, "\033[%dm%s\033[0m", c, fmt.Sprintf(format, a...))
	}
	return fmt.Fprintf(out, format, a...)
}

// ColoredWriteCloser forces printing with colors to an io.WriteCloser.
type ColoredWriteCloser struct {
	io.WriteCloser
}

// This implementation comes from logrus (https://github.com/sirupsen/logrus/blob/master/terminal_check_notappengine.go),
// unfortunately logrus doesn't expose a public interface we can use to call it.
func isTerminal(w io.Writer) bool {
	if _, ok := w.(ColoredWriteCloser); ok {
		return true
	}

	switch v := w.(type) {
	case *os.File:
		return terminal.IsTerminal(int(v.Fd()))
	default:
		return false
	}
}
