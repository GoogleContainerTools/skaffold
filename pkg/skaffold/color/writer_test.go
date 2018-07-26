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
	"bytes"
	"io"
	"testing"
)

func compareText(t *testing.T, expected, actual []byte, expectedN int, actualN int, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("Did not expect error when formatting text but got %s", err)
	}
	if actualN != expectedN {
		t.Errorf("Expected formatter to have written %d bytes but wrote %d", expectedN, actualN)
	}
	if !bytes.Equal(actual, expected) {
		t.Errorf("Formatting not applied to text. Expected \"%s\" but got \"%s\"", expected, actual)
	}
}

func TestWrite(t *testing.T) {
	var b bytes.Buffer
	c := Writer{
		Color:      Green,
		Out:        &b,
		isTerminal: func(_ io.Writer) bool { return true },
	}

	n, err := c.Write([]byte("It's not easy being"))
	expected := []byte("\033[32mIt's not easy being\033[0m")
	compareText(t, expected, b.Bytes(), 28, n, err)
}

func TestWriteNoTTY(t *testing.T) {
	var b bytes.Buffer
	c := Writer{
		Color:      Green,
		Out:        &b,
		isTerminal: func(_ io.Writer) bool { return false },
	}

	expected := []byte("It's not easy being")
	n, err := c.Write(expected)
	compareText(t, expected, b.Bytes(), 19, n, err)
}
