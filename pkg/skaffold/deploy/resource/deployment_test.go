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

package resource

import (
	"context"
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestDeploymentCheckStatus(t *testing.T) {
	rolloutCmd := "kubectl --context kubecontext rollout status deployment dep --namespace test --watch=false"
	tests := []struct {
		description     string
		commands        util.Command
		expectedErr     string
		expectedDetails string
		complete        bool
	}{
		{
			description: "rollout status success",
			commands: testutil.CmdRunOut(
				rolloutCmd,
				"deployment dep successfully rolled out",
			),
			expectedDetails: "deployment dep successfully rolled out",
			complete:        true,
		},
		{
			description: "resource not complete",
			commands: testutil.CmdRunOut(
				rolloutCmd,
				"Waiting for replicas to be available",
			),
			expectedDetails: "Waiting for replicas to be available",
		},
		{
			description: "no output",
			commands: testutil.CmdRunOut(
				rolloutCmd,
				"",
			),
		},
		{
			description: "rollout status error",
			commands: testutil.CmdRunOutErr(
				rolloutCmd,
				"",
				fmt.Errorf("error"),
			),
			expectedErr: "error",
			complete:    true,
		},
		{
			description: "rollout kubectl client connection error",
			commands: testutil.CmdRunOutErr(
				rolloutCmd,
				"",
				fmt.Errorf("Unable to connect to the server"),
			),
			expectedErr: "kubectl connection error",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.commands)
			r := Deployment{Base: &Base{namespace: "test", name: "dep"}}
			runCtx := &runcontext.RunContext{
				KubeContext: "kubecontext",
			}

			r.CheckStatus(context.Background(), runCtx)
			t.CheckDeepEqual(test.complete, r.IsDone())
			if test.expectedErr != "" {
				t.CheckErrorContains(test.expectedErr, r.Status().Error())
			} else {
				t.CheckDeepEqual(r.status.details, test.expectedDetails)
			}
		})
	}
}
