/*
Copyright 2020 The Skaffold Authors

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
	"bytes"
	"context"
	"testing"
)

func compareText(t *testing.T, expected, actual string) {
	t.Helper()
	if actual != expected {
		t.Errorf("Formatting not applied to text. Expected %q but got %q", expected, actual)
	}
}

func TestFprintln(t *testing.T) {
	defer func() { SetupColors(context.Background(), nil, DefaultColorCode, false) }()
	var b bytes.Buffer

	cw := SetupColors(context.Background(), &b, 0, true)
	Green.Fprintln(cw, "2", "less", "chars!")

	compareText(t, "\033[32m2 less chars!\033[0m\n", b.String())
}

func TestFprintf(t *testing.T) {
	defer func() { SetupColors(context.Background(), nil, DefaultColorCode, false) }()
	var b bytes.Buffer

	cw := SetupColors(context.Background(), &b, 0, true)
	Green.Fprintf(cw, "It's been %d %s", 1, "week")

	compareText(t, "\033[32mIt's been 1 week\033[0m", b.String())
}

func TestFprintlnNoTTY(t *testing.T) {
	var b bytes.Buffer

	cw := SetupColors(context.Background(), &b, 0, false)
	Green.Fprintln(cw, "2", "less", "chars!")

	compareText(t, "2 less chars!\n", b.String())
}

func TestFprintfNoTTY(t *testing.T) {
	var b bytes.Buffer

	cw := SetupColors(context.Background(), &b, 0, false)
	Green.Fprintf(cw, "It's been %d %s", 1, "week")

	compareText(t, "It's been 1 week", b.String())
}

func TestFprintlnDefaultColor(t *testing.T) {
	var b bytes.Buffer

	cw := SetupColors(context.Background(), &b, 34, true)
	Default.Fprintln(cw, "2", "less", "chars!")
	compareText(t, "\033[34m2 less chars!\033[0m\n", b.String())
}

func TestFprintlnChangeDefaultToNone(t *testing.T) {
	var b bytes.Buffer

	cw := SetupColors(context.Background(), &b, 0, true)
	Default.Fprintln(cw, "2", "less", "chars!")
	compareText(t, "2 less chars!\n", b.String())
}

func TestFprintlnChangeDefaultToUnknown(t *testing.T) {
	var b bytes.Buffer

	cw := SetupColors(context.Background(), &b, -1, true)
	Default.Fprintln(cw, "2", "less", "chars!")
	compareText(t, "2 less chars!\n", b.String())
}

func TestSprintf(t *testing.T) {
	// Set Default to original blue as it gets modified by other tests
	Default = Blue

	tests := []struct {
		name     string
		color    Color
		params   []interface{}
		expected string
	}{
		{
			name:     "default color",
			color:    Default,
			params:   []interface{}{"a", "few", "words"},
			expected: "\u001B[34ma few words\u001B[0m",
		},
		{
			name:     "red color",
			color:    Red,
			params:   []interface{}{"a", "few", "words"},
			expected: "\u001B[31ma few words\u001B[0m",
		},
		{
			name: "nil color",
			color: Color{
				color: nil,
			},
			params:   []interface{}{"a", "few", "words"},
			expected: "a few words",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.color.Sprintf("%s %s %s", test.params...)
			compareText(t, test.expected, got)
		})
	}
}
