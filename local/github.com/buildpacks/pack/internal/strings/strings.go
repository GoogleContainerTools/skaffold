package strings

import (
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func ValueOrDefault(str, def string) string {
	if str == "" {
		return def
	}

	return str
}

func Title(lower string) string {
	return cases.Title(language.English).String(lower)
}
