package style

import (
	"fmt"
	"sort"
	"strings"

	"github.com/heroku/color"
)

var Symbol = func(value string) string {
	if color.Enabled() {
		return Key(value)
	}
	return "'" + value + "'"
}

var Map = func(value map[string]string, prefix, separator string) string {
	result := ""

	var keys []string

	for key := range value {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		result += fmt.Sprintf("%s%s=%s%s", prefix, key, value[key], separator)
	}

	if color.Enabled() {
		return Key(strings.TrimSpace(result))
	}
	return "'" + strings.TrimSpace(result) + "'"
}

var SymbolF = func(format string, a ...interface{}) string {
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
var Waiting = color.HiBlackString
var Working = color.HiBlueString
var Complete = color.GreenString
var ProgressBar = color.HiBlueString
