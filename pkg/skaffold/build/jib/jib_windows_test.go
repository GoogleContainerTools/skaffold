// +build windows

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

package jib

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestRelativize(t *testing.T) {
	tests := []struct {
		description string
		path        string
		roots       []string
		shouldErr   bool
		result      string
	}{
		{"relative passthrough 0", "relative", []string{}, false, "relative"},
		{"relative passthrough 1", "relative", []string{`a:\`}, false, "relative"},
		{"error if abs and no roots", `c:\abs`, []string{}, true, ""},
		{"error if not relative to roots", `c:\abs`, []string{`a:\`, `c:\a`, `c:\b`}, true, ""},
		{"found in root 0", `a:\z`, []string{`a:\`}, false, "z"},
		{"found in root 1", `b:\z`, []string{`a:\`, `b:\`}, false, "z"},
		{"multilevel found", `b:\c\d\z`, []string{`a:\`, `b:\`}, false, `c\d\z`},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			rel, err := relativize(test.path, test.roots...)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.result, rel)
		})
	}
}
