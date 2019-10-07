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

package misc

import (
	"context"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGracefulBuildCancel(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("graceful cancel doesn't work on windows")
	}

	tests := []struct {
		description string
		command     string
		shouldErr   bool
	}{
		{
			description: "terminate gracefully and exit 0",
			command:     "trap 'echo trap' INT; sleep 2",
		}, {
			description: "terminate gracefully and kill process",
			command:     "trap 'echo trap' INT; sleep 5",
			shouldErr:   true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)

			cmd := exec.Command("bash", "-c", test.command)
			t.CheckNoError(cmd.Start())

			err := HandleGracefulTermination(ctx, cmd)
			t.CheckError(test.shouldErr, err)

			cancel()
		})
	}
}
