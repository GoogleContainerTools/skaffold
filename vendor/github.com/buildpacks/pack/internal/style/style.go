package style

import (
	"fmt"

	"github.com/heroku/color"
)

var Noop = func(format string, a ...interface{}) string {
	return color.WhiteString("") + fmt.Sprintf(format, a...)
}

var Symbol = func(format string, a ...interface{}) string {
	if color.Enabled() {
		return Key(format, a...)
	}
	return "'" + fmt.Sprintf(format, a...) + "'"
}

var Key = color.HiBlueString

var Tip = color.New(color.FgGreen, color.Bold).SprintfFunc()

var Warn = color.New(color.FgYellow, color.Bold).SprintfFunc()

var Error = color.New(color.FgRed, color.Bold).SprintfFunc()

var Step = func(format string, a ...interface{}) string {
	return color.CyanString("===> "+format, a...)
}

var Prefix = color.CyanString

var TimestampColorCode = color.FgHiBlack

var Waiting = color.HiBlackString
var Working = color.HiBlueString
var Complete = color.GreenString
var ProgressBar = color.HiBlueString
