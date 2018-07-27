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

package output

import (
	"bytes"
	"io"
	"testing"
)

func compareText(t *testing.T, expected string, actual string, expectedN int, actualN int, err error) {
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

func TestPrint(t *testing.T) {
	var b bytes.Buffer
	c := ColorFormatter{
		Color:      ColorCodeGreen,
		out:        &b,
		isTerminal: func(_ io.Writer) bool { return true },
	}

	n, err := c.Print("It's not easy being")
	expected := "\033[32mIt's not easy being\033[0m"
	compareText(t, expected, b.String(), 28, n, err)
}

func TestPrintWithPrefix(t *testing.T) {
	var b bytes.Buffer
	c := ColorFormatter{
		Color:      ColorCodeGreen,
		out:        &b,
		isTerminal: func(_ io.Writer) bool { return true },
	}

	n, err := c.PrintWithPrefix("Motto", "Nothing, what's a motto with you?")
	expected := "\033[32mMotto\033[0m Nothing, what's a motto with you?"
	compareText(t, expected, b.String(), 48, n, err)
}

func TestPrintln(t *testing.T) {
	var b bytes.Buffer
	c := ColorFormatter{
		Color:      ColorCodeGreen,
		out:        &b,
		isTerminal: func(_ io.Writer) bool { return true },
	}

	n, err := c.Println("2 less chars!")
	expected := "\033[32m2 less chars!\033[0m\n"
	compareText(t, expected, b.String(), 23, n, err)
}

func TestPrintf(t *testing.T) {
	var b bytes.Buffer
	c := ColorFormatter{
		Color:      ColorCodeGreen,
		out:        &b,
		isTerminal: func(_ io.Writer) bool { return true },
	}

	n, err := c.Printf("It's been %d %s", 1, "week")
	expected := "\033[32mIt's been 1 week\033[0m"
	compareText(t, expected, b.String(), 25, n, err)
}

func TestPrintNoTTY(t *testing.T) {
	var b bytes.Buffer
	c := ColorFormatter{
		Color:      ColorCodeGreen,
		out:        &b,
		isTerminal: func(_ io.Writer) bool { return false },
	}

	expected := "It's not easy being"
	n, err := c.Print(expected)
	compareText(t, expected, b.String(), 19, n, err)
}

func TestPrintWithPrefixNoTTY(t *testing.T) {
	var b bytes.Buffer
	c := ColorFormatter{
		Color:      ColorCodeGreen,
		out:        &b,
		isTerminal: func(_ io.Writer) bool { return false },
	}

	n, err := c.PrintWithPrefix("Motto", "Nothing, what's a motto with you?")
	expected := "Motto Nothing, what's a motto with you?"
	compareText(t, expected, b.String(), 39, n, err)
}

func TestPrintlnNoTTY(t *testing.T) {
	var b bytes.Buffer
	c := ColorFormatter{
		Color:      ColorCodeGreen,
		out:        &b,
		isTerminal: func(_ io.Writer) bool { return false },
	}

	n, err := c.Println("2 less chars!")
	expected := "2 less chars!\n"
	compareText(t, expected, b.String(), 14, n, err)
}
func TestPrintfNoTTY(t *testing.T) {
	var b bytes.Buffer
	c := ColorFormatter{
		Color:      ColorCodeGreen,
		out:        &b,
		isTerminal: func(_ io.Writer) bool { return false },
	}

	n, err := c.Printf("It's been %d %s", 1, "week")
	expected := "It's been 1 week"
	compareText(t, expected, b.String(), 16, n, err)
}
