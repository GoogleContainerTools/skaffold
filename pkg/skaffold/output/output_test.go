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
	"context"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	eventV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/v2"
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

func TestWithEventContext(t *testing.T) {
	tests := []struct {
		name      string
		writer    io.Writer
		phase     constants.Phase
		subtaskID string

		expected io.Writer
	}{
		{
			name: "skaffoldWriter update info",
			writer: skaffoldWriter{
				MainWriter:  ioutil.Discard,
				EventWriter: eventV2.NewLogger(constants.Build, "1"),
			},
			phase:     constants.Test,
			subtaskID: "2",
			expected: skaffoldWriter{
				MainWriter:  ioutil.Discard,
				EventWriter: eventV2.NewLogger(constants.Test, "2"),
			},
		},
		{
			name:     "non skaffoldWriter returns same",
			writer:   ioutil.Discard,
			expected: ioutil.Discard,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			got, _ := WithEventContext(context.Background(), test.writer, test.phase, test.subtaskID)
			t.CheckDeepEqual(test.expected, got, cmpopts.IgnoreTypes(false, "", constants.DevLoop))
		})
	}
}

func TestWriteWithTimeStamps(t *testing.T) {
	tests := []struct {
		name        string
		writer      func(io.Writer) io.Writer
		expectedLen int
	}{
		{
			name: "skaffold writer with color and timestamps",
			writer: func(out io.Writer) io.Writer {
				return skaffoldWriter{
					MainWriter:  colorableWriter{out},
					EventWriter: ioutil.Discard,
					timestamps:  true,
				}
			},
			expectedLen: len(timestampFormat) + len(" \u001B[32mtesting!\u001B[0m"),
		},
		{
			name: "skaffold writer with color and no timestamps",
			writer: func(out io.Writer) io.Writer {
				return skaffoldWriter{
					MainWriter:  colorableWriter{out},
					EventWriter: ioutil.Discard,
				}
			},
			expectedLen: len("\u001B[32mtesting!\u001B[0m"),
		},
		{
			name: "skaffold writer with timestamps and no color",
			writer: func(out io.Writer) io.Writer {
				return skaffoldWriter{
					MainWriter:  out,
					EventWriter: ioutil.Discard,
					timestamps:  true,
				}
			},
			expectedLen: len(timestampFormat) + len(" testing!"),
		},
		{
			name: "skaffold writer with no color and no timestamps",
			writer: func(out io.Writer) io.Writer {
				return skaffoldWriter{
					MainWriter:  out,
					EventWriter: ioutil.Discard,
				}
			},
			expectedLen: len("testing!"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var buf bytes.Buffer
			out := test.writer(&buf)
			Default.Fprintf(out, "testing!")
			testutil.CheckDeepEqual(t, test.expectedLen, len(buf.String()))
		})
	}
}

func TestLog(t *testing.T) {
	tests := []struct {
		name            string
		writer          io.Writer
		expectedTask    constants.Phase
		expectedSubtask string
	}{
		{
			name: "arbitrary task and subtask from writer",
			writer: skaffoldWriter{
				task:    constants.Build,
				subtask: "test",
			},
			expectedTask:    constants.Build,
			expectedSubtask: "test",
		},
		{
			name:            "non skaffoldWriter",
			writer:          ioutil.Discard,
			expectedTask:    constants.DevLoop,
			expectedSubtask: eventV2.SubtaskIDNone,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Log(test.writer)
			testutil.CheckDeepEqual(t, test.expectedTask, got.Data["task"])
			testutil.CheckDeepEqual(t, test.expectedSubtask, got.Data["subtask"])
		})
	}
}
