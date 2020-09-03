package ui

import (
	"bytes"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/vbauerster/mpb/v5/decor"
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
