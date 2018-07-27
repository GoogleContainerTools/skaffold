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
)

// Color can be used to format text using ANSI escape codes so it can be printed to
// the terminal in color.
type Color int

var (
	// ColorCodeLightRed can format text to be displayed to the terminal in light red, using ANSI escape codes.
	ColorCodeLightRed = Color(91)
	// ColorCodeLightGreen can format text to be displayed to the terminal in light green, using ANSI escape codes.
	ColorCodeLightGreen = Color(92)
	// ColorCodeLightYellow can format text to be displayed to the terminal in light yellow, using ANSI escape codes.
	ColorCodeLightYellow = Color(93)
	// ColorCodeLightBlue can format text to be displayed to the terminal in light blue, using ANSI escape codes.
	ColorCodeLightBlue = Color(94)
	// ColorCodeLightPurple can format text to be displayed to the terminal in light purple, using ANSI escape codes.
	ColorCodeLightPurple = Color(95)
	// ColorCodeRed can format text to be displayed to the terminal in red, using ANSI escape codes.
	ColorCodeRed = Color(31)
	// ColorCodeGreen can format text to be displayed to the terminal in green, using ANSI escape codes.
	ColorCodeGreen = Color(32)
	// ColorCodeYellow can format text to be displayed to the terminal in yellow, using ANSI escape codes.
	ColorCodeYellow = Color(33)
	// ColorCodeBlue can format text to be displayed to the terminal in blue, using ANSI escape codes.
	ColorCodeBlue = Color(34)
	// ColorCodePurple can format text to be displayed to the terminal in purple, using ANSI escape codes.
	ColorCodePurple = Color(35)
	// ColorCodeCyan can format text to be displayed to the terminal in cyan, using ANSI escape codes.
	ColorCodeCyan = Color(36)
	// ColorCodeNone uses ANSI escape codes to reset all formatting.
	ColorCodeNone = Color(0)

	// SkaffoldOutputColor default output color for output from Skaffold to the user
	SkaffoldOutputColor = ColorCodeBlue
)

// Sprint will format the operands such that they are surrounded by the ANSI escape sequence
// required to display the text to the terminal in color.
func (c Color) Sprint(a ...interface{}) string {
	text := fmt.Sprint(a...)
	return fmt.Sprintf("\033[%dm%s\033[0m", c, text)
}

// Sprintf will format the operands according ot the format specifier and wrap the resulting text
// with the ANSI escape sequence required to display the text to the terminal in color.
func (c Color) Sprintf(format string, a ...interface{}) string {
	formatSpecifier := c.Sprint(format)
	return fmt.Sprintf(formatSpecifier, a...)
}
