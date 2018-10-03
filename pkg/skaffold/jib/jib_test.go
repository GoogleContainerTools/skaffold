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

package jib

import (
	"os/exec"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGetDependencies(t *testing.T) {
	var tests = []struct {
		stdout       string
		expectedDeps []string
	}{
		{
			stdout:       "",
			expectedDeps: nil,
		},
		{
			stdout:       "deps1\ndeps2",
			expectedDeps: []string{"deps1", "deps2"},
		},
		{
			stdout:       "deps1\ndeps2\n",
			expectedDeps: []string{"deps1", "deps2"},
		},
		{
			stdout:       "\n\n\n",
			expectedDeps: nil,
		},
		{
			stdout:       "\n\ndeps1\n\ndeps2\n\n\n",
			expectedDeps: []string{"deps1", "deps2"},
		},
	}

	for _, test := range tests {
		t.Run("getDependencies", func(t *testing.T) {
			defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
			util.DefaultExecCommand = testutil.NewFakeCmdOut(
				"ignored",
				test.stdout,
				nil,
			)

			deps, err := getDependencies(&exec.Cmd{Args: []string{"ignored"}})
			testutil.CheckErrorAndDeepEqual(t, false, err, test.expectedDeps, deps)
		})
	}
}
