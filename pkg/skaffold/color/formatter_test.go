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
	"bytes"
	"testing"
)

func compareText(t *testing.T, expected, actual string) {
	t.Helper()
	if actual != expected {
		t.Errorf("Formatting not applied to text. Expected %q but got %q", expected, actual)
	}
}

func TestFprintln(t *testing.T) {
	defer func() { SetupColors(nil, DefaultColorCode, false) }()
	var b bytes.Buffer

	SetupColors(&b, 0, true)
	Green.Fprintln(&b, "2", "less", "chars!")

	compareText(t, "\033[32m2 less chars!\033[0m\n", b.String())
}

func TestFprintf(t *testing.T) {
	defer func() { SetupColors(nil, DefaultColorCode, false) }()
	var b bytes.Buffer

	SetupColors(&b, 0, true)
	Green.Fprintf(&b, "It's been %d %s", 1, "week")

	compareText(t, "\033[32mIt's been 1 week\033[0m", b.String())
}

func TestFprintlnNoTTY(t *testing.T) {
	var b bytes.Buffer

	SetupColors(&b, 0, false)
	Green.Fprintln(&b, "2", "less", "chars!")

	compareText(t, "2 less chars!\n", b.String())
}

func TestFprintfNoTTY(t *testing.T) {
	var b bytes.Buffer

	SetupColors(&b, 0, false)
	Green.Fprintf(&b, "It's been %d %s", 1, "week")

	compareText(t, "It's been 1 week", b.String())
}

func TestFprintlnDefaultColor(t *testing.T) {
	var b bytes.Buffer

	SetupColors(&b, 34, true)
	Default.Fprintln(&b, "2", "less", "chars!")
	compareText(t, "\033[34m2 less chars!\033[0m\n", b.String())
}

func TestFprintlnChangeDefaultToNone(t *testing.T) {
	var b bytes.Buffer

	SetupColors(&b, 0, true)
	Default.Fprintln(&b, "2", "less", "chars!")
	compareText(t, "2 less chars!\n", b.String())
}

func TestFprintlnChangeDefaultToUnknown(t *testing.T) {
	var b bytes.Buffer

	SetupColors(&b, -1, true)
	Default.Fprintln(&b, "2", "less", "chars!")
	compareText(t, "2 less chars!\n", b.String())
}
