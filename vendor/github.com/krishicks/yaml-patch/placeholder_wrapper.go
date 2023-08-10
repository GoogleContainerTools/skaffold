package yamlpatch

import (
	"fmt"
	"regexp"
)

// PlaceholderWrapper can be used to wrap placeholders that make YAML invalid
// in single quotes to make otherwise valid YAML
type PlaceholderWrapper struct {
	LeftSide       string
	RightSide      string
	unwrappedRegex *regexp.Regexp
	wrappedRegex   *regexp.Regexp
}

// NewPlaceholderWrapper returns a new PlaceholderWrapper which knows how to
// wrap and unwrap the provided left and right sides of a placeholder, e.g. {{
// and }}
func NewPlaceholderWrapper(left, right string) *PlaceholderWrapper {
	escapedLeft := regexp.QuoteMeta(left)
	escapedRight := regexp.QuoteMeta(right)
	unwrappedRegex := regexp.MustCompile(`\s` + escapedLeft + `([^` + escapedRight + `]+)` + escapedRight)
	wrappedRegex := regexp.MustCompile(`\s'` + escapedLeft + `([^` + escapedRight + `]+)` + escapedRight + `'`)

	return &PlaceholderWrapper{
		LeftSide:       left,
		RightSide:      right,
		unwrappedRegex: unwrappedRegex,
		wrappedRegex:   wrappedRegex,
	}
}

// Wrap the placeholder in single quotes to make it valid YAML
func (w *PlaceholderWrapper) Wrap(input []byte) []byte {
	if !w.unwrappedRegex.Match(input) {
		return input
	}

	return w.unwrappedRegex.ReplaceAll(input, []byte(fmt.Sprintf(` '%s$1%s'`, w.LeftSide, w.RightSide)))
}

// Unwrap the single quotes from the placeholder to make it invalid YAML
// (again)
func (w *PlaceholderWrapper) Unwrap(input []byte) []byte {
	if !w.wrappedRegex.Match(input) {
		return input
	}

	return w.wrappedRegex.ReplaceAll(input, []byte(fmt.Sprintf(` %s$1%s`, w.LeftSide, w.RightSide)))
}
