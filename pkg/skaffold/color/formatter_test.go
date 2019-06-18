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
	"io"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func compareText(t *testing.T, expected, actual string, expectedN int, actualN int, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("Did not expect error when formatting text but got %s", err)
	}
	if actualN != expectedN {
		t.Errorf("Expected formatter to have written %d bytes but wrote %d", expectedN, actualN)
	}
	if actual != expected {
		t.Errorf("Formatting not applied to text. Expected \"%s\" but got \"%s\"", expected, actual)
	}
}

func TestFprint(t *testing.T) {
	reset := ForceColors()
	defer reset()

	var b bytes.Buffer
	n, err := Green.Fprint(&b, "It's not easy being")

	compareText(t, "\033[32mIt's not easy being\033[0m", b.String(), 28, n, err)
}

func TestFprintln(t *testing.T) {
	reset := ForceColors()
	defer reset()

	var b bytes.Buffer
	n, err := Green.Fprintln(&b, "2", "less", "chars!")

	compareText(t, "\033[32m2 less chars!\033[0m\n", b.String(), 23, n, err)
}

func TestFprintf(t *testing.T) {
	reset := ForceColors()
	defer reset()

	var b bytes.Buffer
	n, err := Green.Fprintf(&b, "It's been %d %s", 1, "week")

	compareText(t, "\033[32mIt's been 1 week\033[0m", b.String(), 25, n, err)
}

type nopCloser struct{ io.Writer }

func (n *nopCloser) Close() error { return nil }

func TestFprintOnColoredWriter(t *testing.T) {
	var b bytes.Buffer

	coloredWriter := ColoredWriteCloser{
		WriteCloser: &nopCloser{Writer: &b},
	}

	n, err := Green.Fprint(coloredWriter, "It's not easy being")

	compareText(t, "\033[32mIt's not easy being\033[0m", b.String(), 28, n, err)
}

func TestFprintNoTTY(t *testing.T) {
	var b bytes.Buffer
	expected := "It's not easy being"
	n, err := Green.Fprint(&b, expected)
	compareText(t, expected, b.String(), 19, n, err)
}

func TestFprintlnNoTTY(t *testing.T) {
	var b bytes.Buffer
	n, err := Green.Fprintln(&b, "2", "less", "chars!")
	expected := "2 less chars!\n"
	compareText(t, expected, b.String(), 14, n, err)
}

func TestFprintfNoTTY(t *testing.T) {
	var b bytes.Buffer
	n, err := Green.Fprintf(&b, "It's been %d %s", 1, "week")
	expected := "It's been 1 week"
	compareText(t, expected, b.String(), 16, n, err)
}

func TestOverwriteDefault(t *testing.T) {
	testutil.CheckDeepEqual(t, Blue, Default)
	OverwriteDefault(Red)
	testutil.CheckDeepEqual(t, Red, Default)
}
