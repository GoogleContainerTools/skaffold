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

package ui

import (
	"bytes"
	"testing"

	"github.com/vbauerster/mpb/v5/decor"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestFill(t *testing.T) {
	tests := []struct {
		description    string
		style          []string
		expectedFrames []string
	}{
		{
			description:    "default spinner",
			style:          []string{},
			expectedFrames: []string{"⠋", "⠋⠙", "⠋⠙⠹"},
		},
		{
			description:    "default spinner nil style",
			style:          nil,
			expectedFrames: []string{"⠋", "⠋⠙", "⠋⠙⠹"},
		},
		{
			description:    "abc spinner",
			style:          []string{"a", "b", "c"},
			expectedFrames: []string{"a", "ab", "abc"},
		},
		{
			description:    "abc spinner wrapping",
			style:          []string{"a", "b", "c"},
			expectedFrames: []string{"a", "ab", "abc", "abca", "abcab"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			filler := NewSpinnerFiller(test.style)
			var output bytes.Buffer

			for _, frame := range test.expectedFrames {
				filler.Fill(&output, 0, decor.Statistics{})
				t.CheckContains(frame, output.String())
			}
		})
	}
}
