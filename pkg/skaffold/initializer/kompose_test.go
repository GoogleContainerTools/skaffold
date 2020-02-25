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

package initializer

import (
	"context"
	"errors"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestRunKompose(t *testing.T) {
	tests := []struct {
		description   string
		composeFile   string
		commands      util.Command
		expectedError string
	}{
		{
			description: "success",
			composeFile: "docker-compose.yaml",
			commands:    testutil.CmdRunOut("kompose convert -f docker-compose.yaml", ""),
		},
		{
			description:   "not found",
			composeFile:   "not-found.yaml",
			expectedError: "(no such file or directory|cannot find the file specified)",
		},
		{
			description:   "failure",
			composeFile:   "docker-compose.yaml",
			commands:      testutil.CmdRunOutErr("kompose convert -f docker-compose.yaml", "", errors.New("BUG")),
			expectedError: "BUG",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.NewTempDir().Touch("docker-compose.yaml").Chdir()
			t.Override(&util.DefaultExecCommand, test.commands)

			err := runKompose(context.Background(), test.composeFile)

			if test.expectedError != "" {
				t.CheckMatches(test.expectedError, err.Error())
			}
		})
	}
}
