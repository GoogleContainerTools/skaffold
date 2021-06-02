/*
Copyright 2021 The Skaffold Authors

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
	"io/ioutil"
	"os"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

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
		{
			description: "colorable std out passed",
			out: skaffoldWriter{
				MainWriter: NewColorWriter(os.Stdout),
			},
			expected: true,
		},
		{
			description: "colorableWriter passed",
			out:         NewColorWriter(os.Stdout),
			expected:    true,
		},
		{
			description: "invalid colorableWriter passed",
			out: skaffoldWriter{
				MainWriter: NewColorWriter(ioutil.Discard),
			},
			expected: false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.CheckDeepEqual(test.expected, IsStdout(test.out))
		})
	}
}

func TestGetUnderlyingWriter(t *testing.T) {
	tests := []struct {
		description string
		out         io.Writer
		expected    io.Writer
	}{
		{
			description: "colorable os.Stdout returns os.Stdout",
			out: skaffoldWriter{
				MainWriter: colorableWriter{os.Stdout},
			},
			expected: os.Stdout,
		},
		{
			description: "skaffold writer returns os.Stdout without colorableWriter",
			out: skaffoldWriter{
				MainWriter: os.Stdout,
			},
			expected: os.Stdout,
		},
		{
			description: "return ioutil.Discard from skaffoldWriter",
			out: skaffoldWriter{
				MainWriter: NewColorWriter(ioutil.Discard),
			},
			expected: ioutil.Discard,
		},
		{
			description: "os.Stdout returned from colorableWriter",
			out:         NewColorWriter(os.Stdout),
			expected:    os.Stdout,
		},
		{
			description: "GetWriter returns original writer if not colorable",
			out:         os.Stdout,
			expected:    os.Stdout,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.CheckDeepEqual(true, test.expected == GetUnderlyingWriter(test.out))
		})
	}
}
