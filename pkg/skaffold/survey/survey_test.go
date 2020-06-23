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

package survey

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestDisplaySurveyForm(t *testing.T) {
	tests := []struct {
		description string
		mockStdOut  bool
		expected    string
	}{
		{
			description: "std out",
			mockStdOut:  true,
			expected:    Prompt + "\n",
		},
		{
			description: "not std out",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			mock := func(io.Writer) bool { return test.mockStdOut }
			t.Override(&isStdOut, mock)
			mockOpen := func(string) error { return nil }
			t.Override(&open, mockOpen)
			t.Override(&updateConfig, func(_ string) error { return nil })
			var buf bytes.Buffer
			New("test").DisplaySurveyPrompt(&buf)
			t.CheckDeepEqual(test.expected, buf.String())
		})
	}
}

func TestIsStdOut(t *testing.T) {
	tests := []struct {
		description string
		out         io.Writer
		expected    bool
	}{
		{
			description: "std out passed",
			out:         os.Stdout,
			expected:    true,
		},
		{
			description: "out nil",
			out:         nil,
		},
		{
			description: "out bytes buffer",
			out:         new(bytes.Buffer),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.CheckDeepEqual(test.expected, isStdOut(test.out))
		})
	}
}
