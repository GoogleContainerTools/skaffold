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

package runner

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/pkg/errors"
)

func TestDeploy(t *testing.T) {
	expectedOutput := "Waiting for deployments to stabalize"
	var tests = []struct {
		description string
		testBench   *TestBench
		shdWait     bool
		statusCheck bool
		shouldError bool
	}{
		{
			description: "deploy shd perform status check",
			testBench:   &TestBench{},
			statusCheck: true,
			shdWait:     true,
		},
		{
			description: "deploy shd not perform status check",
			testBench:   &TestBench{},
		},
		{
			description: "deploy shd not perform status check when deployer is in error",
			testBench:   &TestBench{deployErrors: []error{errors.New("deploy error")}},
			shouldError: true,
			statusCheck: true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {

			runner := createRunner(t, test.testBench)
			runner.runCtx.Opts.StatusCheck = test.statusCheck
			out := new(bytes.Buffer)

			err := runner.Deploy(context.Background(), out, []build.Artifact{
				{ImageName: "img1", Tag: "img1:tag1"},
				{ImageName: "img2", Tag: "img2:tag2"},
			})
			t.CheckError(test.shouldError, err)
			if strings.Contains(out.String(), expectedOutput) != test.shdWait {
				t.Errorf("expected %s to contain %s %t. But found %t", out.String(), expectedOutput, test.shdWait, !test.shdWait)
			}
		})
	}
}
