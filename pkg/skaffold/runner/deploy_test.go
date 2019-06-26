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
<<<<<<< HEAD
<<<<<<< HEAD
	"bytes"
	"context"
	"errors"
	"strings"
=======
	"context"
	"io/ioutil"
>>>>>>> wip
=======
	"bytes"
	"context"
	"errors"
	"strings"
>>>>>>> add status check flag which does nothing for now and test
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/context"
	"github.com/GoogleContainerTools/skaffold/testutil"
<<<<<<< HEAD
<<<<<<< HEAD
)

func TestDeploy(t *testing.T) {
	expectedOutput := "Waiting for deployments to stabilize"
	var tests = []struct {
		description string
		testBench   *TestBench
		statusCheck bool
		shouldErr   bool
		shouldWait  bool
=======
	"github.com/pkg/errors"
=======
>>>>>>> add status check flag which does nothing for now and test
)

func TestDeploy(t *testing.T) {
	expectedOutput := "Waiting for deployments to stabilize"
	var tests = []struct {
		description string
		testBench   *TestBench
		statusCheck bool
<<<<<<< HEAD
>>>>>>> wip
=======
		shouldErr   bool
		shouldWait  bool
>>>>>>> add status check flag which does nothing for now and test
	}{
		{
			description: "deploy shd perform status check",
			testBench:   &TestBench{},
			statusCheck: true,
<<<<<<< HEAD
<<<<<<< HEAD
			shouldWait:  true,
=======
>>>>>>> wip
=======
			shouldWait:  true,
>>>>>>> add status check flag which does nothing for now and test
		},
		{
			description: "deploy shd not perform status check",
			testBench:   &TestBench{},
		},
		{
			description: "deploy shd not perform status check when deployer is in error",
<<<<<<< HEAD
<<<<<<< HEAD
			shouldErr:   true,
			statusCheck: true,
			testBench:   &TestBench{deployErrors: []error{errors.New("deploy error")}},
=======
			testBench:   &TestBench{deployErrors: []error{errors.New("deploy error")}},
			shouldError: true,
			statusCheck: true,
>>>>>>> wip
=======
			shouldErr:   true,
			statusCheck: true,
			testBench:   &TestBench{deployErrors: []error{errors.New("deploy error")}},
>>>>>>> add status check flag which does nothing for now and test
		},
	}

	dummyStatusCheck := func(ctx context.Context, l *deploy.DefaultLabeller, runCtx *runcontext.RunContext) error {
		return nil
	}
	originalStatusCheck := deploy.StatusCheck
	for _, test := range tests {
<<<<<<< HEAD
<<<<<<< HEAD
		testutil.Run(t, test.description, func(t *testutil.T) {

			runner := createRunner(t, test.testBench, nil)
			runner.runCtx.Opts.StatusCheck = test.statusCheck
			out := new(bytes.Buffer)

			err := runner.Deploy(context.Background(), out, []build.Artifact{
				{ImageName: "img1", Tag: "img1:tag1"},
				{ImageName: "img2", Tag: "img2:tag2"},
			})
			t.CheckError(test.shouldErr, err)
			if strings.Contains(out.String(), expectedOutput) != test.shouldWait {
				t.Errorf("expected %s to contain %s %t. But found %t", out.String(), expectedOutput, test.shouldWait, !test.shouldWait)
			}
=======
		t.Run(test.description, func(t *testing.T) {
=======
		testutil.Run(t, test.description, func(t *testutil.T) {
>>>>>>> add status check flag which does nothing for now and test

			runner := createRunner(t, test.testBench)
			runner.runCtx.Opts.StatusCheck = test.statusCheck
			out := new(bytes.Buffer)

			// Figure out why i can't use t.Override.
			// Using t.Override throws an error "reflect: call of reflect.Value.Elem on func Value"
			statusCheck = dummyStatusCheck
			defer func() { statusCheck = originalStatusCheck }()

			err := runner.Deploy(context.Background(), out, []build.Artifact{
				{ImageName: "img1", Tag: "img1:tag1"},
				{ImageName: "img2", Tag: "img2:tag2"},
			})
<<<<<<< HEAD
			testutil.CheckError(t, test.shouldError, err)
>>>>>>> wip
=======
			t.CheckError(test.shouldErr, err)
			if strings.Contains(out.String(), expectedOutput) != test.shouldWait {
				t.Errorf("expected %s to contain %s %t. But found %t", out.String(), expectedOutput, test.shouldWait, !test.shouldWait)
			}
>>>>>>> add status check flag which does nothing for now and test
		})
	}
}