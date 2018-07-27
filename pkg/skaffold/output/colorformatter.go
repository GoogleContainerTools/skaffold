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

package output

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/ssh/terminal"
)

// ColorFormatter implements formatted i/o which allows the caller to wrap output to the
// in color (via ANSI escape codes) when output is to a terminal. It will write to out, which
// in most cases will be stdout. Skaffold follows a pattern of logging to stderr via logrus
// for diagnostic output, and uses stdout for messages from Skaffold itself to the user.
type ColorFormatter struct {
	Color Color

	out        io.Writer
	isTerminal func(w io.Writer) bool
}

func (o *ColorFormatter) wrapTextIfTerminal(a ...interface{}) string {
	if o.isTerminal(o.out) {
		return o.Color.Sprint(a...)
	}
	return fmt.Sprint(a...)
}

// NewColorFormatter instantiates an instance of ColorFormatter that will output to out in
// color, as long as out is a terminal. If out is not a terminal, the color will not be used.
func NewColorFormatter(out io.Writer, color Color) ColorFormatter {
	return ColorFormatter{
		Color: color,

		out:        out,
		isTerminal: isTerminal,
	}
}

// Print wraps the operands in the color ANSI escape codes using o's Color object, and outputs the result to
// o's out object. If output is not being directed to a terminal, the escape codes will not be added.
// It returns the number of bytes written and any errors encountered.
func (o *ColorFormatter) Print(a ...interface{}) (n int, err error) {
	t := o.wrapTextIfTerminal(a...)
	return fmt.Fprint(o.out, t)
}

// PrintWithPrefix wraps the prefix in the color ANSI escape codes using o's Color object,
// and outputs the resulting formatted prefix, followed by the rest of the operands, to o's out object.
// If output is not being directed to a terminal, the escape codes will not be added.
// It returns the number of bytes written and any errors encountered.
func (o *ColorFormatter) PrintWithPrefix(prefix string, a ...interface{}) (n int, err error) {
	p := o.wrapTextIfTerminal(prefix)
	text := fmt.Sprint(a...)
	return fmt.Fprintf(o.out, "%s %s", p, text)
}

// Println wraps the operands in the color ANSI escape codes using o's Color object, and outputs the result to
// o's out object, followed by a newline. If output is not being directed to a terminal, the escape codes will not be added.
// It returns the number of bytes written and any errors encountered.
func (o *ColorFormatter) Println(a ...interface{}) (n int, err error) {
	t := o.wrapTextIfTerminal(a...)
	return fmt.Fprintln(o.out, t)
}

// Printf applies formats according to the format specifier (and the optional interfaces provided)
// wraps the text in the color ANSI escape codes using o's Color object, and outputs the result to
// o's out object. If output is not being directed to a terminal, the escape codes will not be added.
// It returns the number of bytes written and any errors encountered.
func (o *ColorFormatter) Printf(format string, a ...interface{}) (n int, err error) {
	var t string
	if o.isTerminal(o.out) {
		t = o.Color.Sprintf(format, a...)
	} else {
		t = fmt.Sprintf(format, a...)
	}
	return fmt.Fprint(o.out, t)
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
