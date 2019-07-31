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
	"errors"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"k8s.io/client-go/tools/clientcmd/api"
)

func TestDeploy(t *testing.T) {
	expectedOutput := "Waiting for deployments to stabilize"
	tests := []struct {
		description string
		testBench   *TestBench
		statusCheck bool
		shouldErr   bool
		shouldWait  bool
	}{
		{
			description: "deploy shd perform status check",
			testBench:   &TestBench{},
			statusCheck: true,
			shouldWait:  true,
		},
		{
			description: "deploy shd not perform status check",
			testBench:   &TestBench{},
		},
		{
			description: "deploy shd not perform status check when deployer is in error",
			testBench:   &TestBench{deployErrors: []error{errors.New("deploy error")}},
			shouldErr:   true,
			statusCheck: true,
		},
	}

	dummyStatusCheck := func(ctx context.Context, l *deploy.DefaultLabeller, runCtx *runcontext.RunContext) error {
		return nil
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.SetupFakeKubernetesContext(api.Config{CurrentContext: "cluster1"})
			t.Override(&statusCheck, dummyStatusCheck)

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
		})
	}
}
