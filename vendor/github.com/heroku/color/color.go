package color

import (
	"fmt"
	"strings"
)

const (
	escape     = "\x1b["
	endCode    = "m"
	lineFeed   = "\n"
	delimiter  = ";"
	colorReset = "\x1b[0m"
)

func chainSGRCodes(a []Attribute) string {
	codes := to_codes(a)
	if len(codes) == 0 {
		return colorReset
	}
	if len(codes) == 1 {
		return escape + codes[0] + endCode
	}
	var bld strings.Builder
	bld.Grow((len(codes) * 2) + len(escape) + len(endCode))
	bld.WriteString(escape)
	delimsAdded := 0
	for i := 0; i < len(a); i++ {
		if delimsAdded > 0 {
			_, _ = bld.WriteString(delimiter)
		}
		bld.WriteString(codes[i])
		delimsAdded++
	}
	bld.WriteString(endCode)
	return bld.String()
}

// Color contains methods to create colored strings of text.
type Color struct {
	colorStart string
}

// New creates a Color. It takes a list of Attributes to define
// the appearance of output.
func New(attrs ...Attribute) *Color {
	return cache().value(attrs...)
}

// Sprint returns text decorated with the display Attributes passed to Color constructor function.
func (v Color) Sprint(a ...interface{}) string {
	if Enabled() {
		return v.wrap(fmt.Sprint(a...))
	}
	return fmt.Sprint(a...)
}

// Sprint formats according to the format specifier and returns text decorated with the display Attributes
// passed to Color constructor function.
func (v Color) Sprintf(format string, a ...interface{}) string {
	if Enabled() {
		return v.wrap(fmt.Sprintf(format, a...))
	}
	return fmt.Sprintf(format, a...)
}

// Sprint returns text decorated with the display Attributes and terminated by a line feed.
func (v Color) Sprintln(a ...interface{}) string {
	var s string
	if Enabled() {
		s = v.wrap(fmt.Sprint(a...))
	} else {
		s = fmt.Sprint(a...)
	}
	if !strings.HasSuffix(s, lineFeed) {
		s += lineFeed
	}
	return s
}

// SprintFunc returns function that wraps Sprint.
func (v Color) SprintFunc() func(a ...interface{}) string {
	return func(a ...interface{}) string {
		return v.Sprint(a...)
	}
}

// SprintfFunc returns function that wraps Sprintf.
func (v Color) SprintfFunc() func(format string, a ...interface{}) string {
	return func(format string, a ...interface{}) string {
		return v.Sprintf(format, a...)
	}
}

// SprintlnFunc returns function that wraps Sprintln.
func (v Color) SprintlnFunc() func(a ...interface{}) string {
	return func(a ...interface{}) string {
		return v.Sprintln(a...)
	}
}

func (v Color) wrap(s ...string) string {
	var b strings.Builder
	b.Grow(len(v.colorStart) + len(s) + len(colorReset))
	b.WriteString(v.colorStart)
	for i := 0; i < len(s); i++ {
		b.WriteString(s[i])
	}
	b.WriteString(colorReset)
	return b.String()
}

func colorString(format string, attr Attribute, a ...interface{}) string {
	return cache().value(attr).Sprintf(format, a...)
}

// Black helper to produce black text to stdout.
func Black(format string, a ...interface{}) { Stdout().colorPrint(format, FgBlack, a...) }

// BlackE helper to produce black text to stderr.
func BlackE(format string, a ...interface{}) { Stderr().colorPrint(format, FgBlack, a...) }

// Red helper to produce red text to stdout.
func Red(format string, a ...interface{}) { Stdout().colorPrint(format, FgRed, a...) }

// RedE helper to produce red text to stderr.
func RedE(format string, a ...interface{}) { Stderr().colorPrint(format, FgRed, a...) }

// Green helper to produce green text to stdout.
func Green(format string, a ...interface{}) { Stdout().colorPrint(format, FgGreen, a...) }

// GreenE helper to produce green text to stderr.
func GreenE(format string, a ...interface{}) { Stderr().colorPrint(format, FgGreen, a...) }

// Yellow helper to produce yellow text to stdout.
func Yellow(format string, a ...interface{}) { Stdout().colorPrint(format, FgYellow, a...) }

// YellowE helper to produce yellow text to stderr.
func YellowE(format string, a ...interface{}) { Stderr().colorPrint(format, FgYellow, a...) }

// Blue helper to produce blue text to stdout.
func Blue(format string, a ...interface{}) { Stdout().colorPrint(format, FgBlue, a...) }

// BlueE helper to produce blue text to stderr.
func BlueE(format string, a ...interface{}) { Stderr().colorPrint(format, FgBlue, a...) }

// Magenta helper to produce magenta text to stdout.
func Magenta(format string, a ...interface{}) { Stdout().colorPrint(format, FgMagenta, a...) }

// MagentaE produces magenta text to stderr.
func MagentaE(format string, a ...interface{}) { Stderr().colorPrint(format, FgMagenta, a...) }

// Cyan helper to produce cyan text to stdout.
func Cyan(format string, a ...interface{}) { Stdout().colorPrint(format, FgCyan, a...) }

// CyanE helper to produce cyan text to stderr.
func CyanE(format string, a ...interface{}) { Stderr().colorPrint(format, FgCyan, a...) }

// White helper to produce white text to stdout.
func White(format string, a ...interface{}) { Stdout().colorPrint(format, FgWhite, a...) }

// WhiteE helper to produce white text to stderr.
func WhiteE(format string, a ...interface{}) { Stderr().colorPrint(format, FgWhite, a...) }

// BlackString returns a string decorated with black attributes.
func BlackString(format string, a ...interface{}) string { return colorString(format, FgBlack, a...) }

// RedString returns a string decorated with red attributes.
func RedString(format string, a ...interface{}) string { return colorString(format, FgRed, a...) }

// GreenString returns a string decorated with green attributes.
func GreenString(format string, a ...interface{}) string { return colorString(format, FgGreen, a...) }

// YellowString returns a string decorated with yellow attributes.
func YellowString(format string, a ...interface{}) string { return colorString(format, FgYellow, a...) }

// BlueString returns a string decorated with blue attributes.
func BlueString(format string, a ...interface{}) string { return colorString(format, FgBlue, a...) }

// MagentaString returns a string decorated with magenta attributes.
func MagentaString(format string, a ...interface{}) string {
	return colorString(format, FgMagenta, a...)
}

// CyanString returns a string decorated with cyan attributes.
func CyanString(format string, a ...interface{}) string { return colorString(format, FgCyan, a...) }

// WhiteString returns a string decorated with white attributes.
func WhiteString(format string, a ...interface{}) string { return colorString(format, FgWhite, a...) }

// HiBlack helper to produce black text to stdout.
func HiBlack(format string, a ...interface{}) { Stdout().colorPrint(format, FgHiBlack, a...) }

// HiBlackE helper to produce black text to stderr.
func HiBlackE(format string, a ...interface{}) { Stderr().colorPrint(format, FgHiBlack, a...) }

// HiRed helper to write high contrast red text to stdout.
func HiRed(format string, a ...interface{}) { Stdout().colorPrint(format, FgHiRed, a...) }

// HiRedE helper to write high contrast red text to stderr.
func HiRedE(format string, a ...interface{}) { Stderr().colorPrint(format, FgHiRed, a...) }

// HiGreen helper writes high contrast green text to stdout.
func HiGreen(format string, a ...interface{}) { Stdout().colorPrint(format, FgHiGreen, a...) }

// HiGreenE helper writes high contrast green text to stderr.
func HiGreenE(format string, a ...interface{}) { Stderr().colorPrint(format, FgHiGreen, a...) }

// HiYellow helper writes high contrast yellow text to stdout.
func HiYellow(format string, a ...interface{}) { Stdout().colorPrint(format, FgHiYellow, a...) }

// HiYellowE helper writes high contrast yellow text to stderr.
func HiYellowE(format string, a ...interface{}) { Stderr().colorPrint(format, FgHiYellow, a...) }

// HiBlue helper writes high contrast blue text to stdout.
func HiBlue(format string, a ...interface{}) { Stdout().colorPrint(format, FgHiBlue, a...) }

// HiBlueE helper writes high contrast blue text to stderr.
func HiBlueE(format string, a ...interface{}) { Stderr().colorPrint(format, FgHiBlue, a...) }

// HiMagenta writes high contrast magenta text to stdout.
func HiMagenta(format string, a ...interface{}) { Stdout().colorPrint(format, FgHiMagenta, a...) }

// HiMagentaE writes high contrast magenta text to stderr.
func HiMagentaE(format string, a ...interface{}) { Stderr().colorPrint(format, FgHiMagenta, a...) }

// HiCyan writes high contrast cyan colored text to stdout.
func HiCyan(format string, a ...interface{}) { Stdout().colorPrint(format, FgHiCyan, a...) }

// HiCyanE writes high contrast contrast cyan colored text to stderr.
func HiCyanE(format string, a ...interface{}) { Stderr().colorPrint(format, FgHiCyan, a...) }

// HiWhite writes high contrast white colored text to stdout.
func HiWhite(format string, a ...interface{}) { Stdout().colorPrint(format, FgHiWhite, a...) }

// HiWhiteE writes high contrast white colored text to stderr.
func HiWhiteE(format string, a ...interface{}) { Stderr().colorPrint(format, FgHiWhite, a...) }

// HiBlackString returns a high contrast black string.
func HiBlackString(format string, a ...interface{}) string {
	return colorString(format, FgHiBlack, a...)
}

// HiRedString returns a high contrast contrast black string.
func HiRedString(format string, a ...interface{}) string { return colorString(format, FgHiRed, a...) }

// HiGreenString returns a high contrast green string.
func HiGreenString(format string, a ...interface{}) string {
	return colorString(format, FgHiGreen, a...)
}

// HiYellowString returns a high contrast yellow string.
func HiYellowString(format string, a ...interface{}) string {
	return colorString(format, FgHiYellow, a...)
}

// HiBlueString returns a high contrast blue string.
func HiBlueString(format string, a ...interface{}) string { return colorString(format, FgHiBlue, a...) }

// HiMagentaString returns a high contrast magenta string.
func HiMagentaString(format string, a ...interface{}) string {
	return colorString(format, FgHiMagenta, a...)
}

// HiCyanString returns a high contrast cyan string.
func HiCyanString(format string, a ...interface{}) string { return colorString(format, FgHiCyan, a...) }

// HiWhiteString returns a high contrast white string.
func HiWhiteString(format string, a ...interface{}) string {
	return colorString(format, FgHiWhite, a...)
}
