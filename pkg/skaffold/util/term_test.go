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

package util

import (
	"bytes"
	"errors"
	"runtime"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestIsNotTerminal(t *testing.T) {
	var w bytes.Buffer

	termFd, isTerm := IsTerminal(&w)

	testutil.CheckDeepEqual(t, uintptr(0x00), termFd)
	testutil.CheckDeepEqual(t, false, isTerm)
}

func TestSupportsColor(t *testing.T) {
	tests := []struct {
		description  string
		colorsOutput string
		shouldErr    bool
		expected     bool
	}{
		{
			description:  "Supports 256 colors",
			colorsOutput: "256",
			expected:     true,
		},
		{
			description:  "Supports 0 colors",
			colorsOutput: "0",
			expected:     false,
		},
		{
			description:  "tput returns -1",
			colorsOutput: "-1",
			expected:     false,
		},
		{
			description:  "cmd run errors",
			colorsOutput: "-1",
			expected:     false,
			shouldErr:    true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			if test.shouldErr {
				t.Override(&DefaultExecCommand, testutil.CmdRunOutErr("tput colors", test.colorsOutput, errors.New("error")))
			} else {
				t.Override(&DefaultExecCommand, testutil.CmdRunOut("tput colors", test.colorsOutput))
			}
			if runtime.GOOS == constants.Windows {
				test.expected = true
				test.shouldErr = false
			}

			supportsColors, err := SupportsColor()
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, supportsColors)
		})
	}
}
