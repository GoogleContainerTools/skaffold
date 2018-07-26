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
)

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

	// SkaffoldOutputColor default output color for output from Skaffold to the user
	SkaffoldOutputColor = Blue
)

// Sprint will format the operands such that they are surrounded by the ANSI escape sequence
// required to display the text to the terminal in color.
func (c Color) Sprint(a ...interface{}) string {
	text := fmt.Sprint(a...)
	return fmt.Sprintf("\033[%dm%s\033[0m", c, text)
}
