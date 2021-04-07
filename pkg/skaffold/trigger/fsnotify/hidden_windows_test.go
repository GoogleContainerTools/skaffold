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

package fsnotify

import (
	"fmt"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestHiddenWindows(t *testing.T) {
	tests := []struct {
		description  string
		path         string
		mockFileAttr func(string) (uint32, error)
		expected     bool
	}{
		{
			description: "file attribute is hidden",
			path:        "hidden.ast",
			mockFileAttr: func(path string) (uint32, error) {
				return syscall.FILE_ATTRIBUTE_HIDDEN, nil
			},
			expected: true,
		},
		{
			description: "file is not hidden",
			path:        "file.txt",
			mockFileAttr: func(path string) (uint32, error) {
				return syscall.FILE_ATTRIBUTE_READONLY, nil
			},
		},
		{
			description: "error reading attributes",
			path:        "err",
			mockFileAttr: func(path string) (uint32, error) {
				return 0, fmt.Errorf("error reading")
			},
		},
		{
			description: "file in a hidden dir",
			path:        filepath.Join([]string{"another", "hidden", "err.txt"}...),
			mockFileAttr: func(path string) (uint32, error) {
				if path == "hidden" {
					return syscall.FILE_ATTRIBUTE_HIDDEN, nil
				}
				return syscall.FILE_ATTRIBUTE_READONLY, nil
			},
			expected: true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			trigger := &Trigger{
				Interval: 10,
			}
			t.Override(&fileAttributes, test.mockFileAttr)
			t.CheckDeepEqual(test.expected, trigger.hidden(test.path))
		})
	}
}
