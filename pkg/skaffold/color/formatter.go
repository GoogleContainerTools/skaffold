/*
Copyright 2019 The Skaffold Authors

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
	"strings"

	colors "github.com/heroku/color"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// Maintain compatibility with the old color coding.
// 34 is the code for blue.
const DefaultColorCode = 34

func init() {
	colors.Disable(true)
}

// SetupColors enables/disables coloured output.
func SetupColors(out io.Writer, defaultColor int, forceColors bool) {
	_, isTerm := util.IsTerminal(out)
	colors.Disable(!isTerm && !forceColors)

	// Maintain compatibility with the old color coding.
	Default = map[int]Color{
		91: LightRed,
		92: LightGreen,
		93: LightYellow,
		94: LightBlue,
		95: LightPurple,
		31: Red,
		32: Green,
		33: Yellow,
		34: Blue,
		35: Purple,
		36: Cyan,
		37: White,
		0:  None,
	}[defaultColor]
}

// Color can be used to format text so it can be printed to the terminal in color.
type Color struct {
	color *colors.Color
}

var (
	// LightRed can format text to be displayed to the terminal in light red.
	LightRed = Color{color: colors.New(colors.FgHiRed)}
	// LightGreen can format text to be displayed to the terminal in light green.
	LightGreen = Color{color: colors.New(colors.FgHiGreen)}
	// LightYellow can format text to be displayed to the terminal in light yellow.
	LightYellow = Color{color: colors.New(colors.FgHiYellow)}
	// LightBlue can format text to be displayed to the terminal in light blue.
	LightBlue = Color{color: colors.New(colors.FgHiBlue)}
	// LightPurple can format text to be displayed to the terminal in light purple.
	LightPurple = Color{color: colors.New(colors.FgHiMagenta)}
	// Red can format text to be displayed to the terminal in red.
	Red = Color{color: colors.New(colors.FgRed)}
	// Green can format text to be displayed to the terminal in green.
	Green = Color{color: colors.New(colors.FgGreen)}
	// Yellow can format text to be displayed to the terminal in yellow.
	Yellow = Color{color: colors.New(colors.FgYellow)}
	// Blue can format text to be displayed to the terminal in blue.
	Blue = Color{color: colors.New(colors.FgBlue)}
	// Purple can format text to be displayed to the terminal in purple.
	Purple = Color{color: colors.New(colors.FgHiMagenta)}
	// Cyan can format text to be displayed to the terminal in cyan.
	Cyan = Color{color: colors.New(colors.FgHiCyan)}
	// White can format text to be displayed to the terminal in white.
	White = Color{color: colors.New(colors.FgWhite)}
	// None uses ANSI escape codes to reset all formatting.
	None = Color{}

	// Default default output color for output from Skaffold to the user
	Default = Blue
)

// Fprintln outputs the result to out, followed by a newline.
func (c Color) Fprintln(out io.Writer, a ...interface{}) {
	if c.color == nil {
		fmt.Fprintln(out, a...)
		return
	}

	fmt.Fprintln(out, c.color.Sprint(strings.TrimSuffix(fmt.Sprintln(a...), "\n")))
}

// Fprintf outputs the result to out.
func (c Color) Fprintf(out io.Writer, format string, a ...interface{}) {
	if c.color == nil {
		fmt.Fprintf(out, format, a...)
		return
	}

	fmt.Fprint(out, c.color.Sprintf(format, a...))
}
