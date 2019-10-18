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
	"errors"
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
				errors.New("error"),
			),
			expectedErr: "error",
			complete:    true,
		},
		{
			description: "rollout kubectl client connection error",
			commands: testutil.CmdRunOutErr(
				rolloutCmd,
				"",
				errors.New("Unable to connect to the server"),
			),
			expectedErr: ErrKubectlConnection.Error(),
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.commands)
			r := NewDeployment("dep", "test", 0)
			runCtx := &runcontext.RunContext{
				KubeContext: "kubecontext",
			}

			r.CheckStatus(context.Background(), runCtx)
			t.CheckDeepEqual(test.complete, r.IsStatusCheckComplete())
			if test.expectedErr != "" {
				t.CheckErrorContains(test.expectedErr, r.Status().Error())
			} else {
				t.CheckDeepEqual(r.status.details, test.expectedDetails)
			}
		})
	}
}

func TestParseKubectlError(t *testing.T) {
	tests := []struct {
		description string
		err         error
		expected    string
		shouldErr   bool
	}{
		{
			description: "rollout status connection error",
			err:         errors.New("Unable to connect to the server"),
			expected:    ErrKubectlConnection.Error(),
			shouldErr:   true,
		},
		{
			description: "rollout status kubectl command killed",
			err:         errors.New("signal: killed"),
			expected:    errKubectlKilled.Error(),
			shouldErr:   true,
		},
		{
			description: "rollout status random error",
			err:         errors.New("deployment test not found"),
			expected:    "deployment test not found",
			shouldErr:   true,
		},
		{
			description: "rollout status nil error",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			actual := parseKubectlRolloutError(test.err)
			t.CheckError(test.shouldErr, actual)
			if test.shouldErr {
				t.CheckErrorContains(test.expected, actual)
			}
		})
	}
}

func TestIsErrAndNotRetriable(t *testing.T) {
	tests := []struct {
		description string
		err         error
		expected    bool
	}{
		{
			description: "rollout status connection error",
			err:         ErrKubectlConnection,
		},
		{
			description: "rollout status kubectl command killed",
			err:         errKubectlKilled,
			expected:    true,
		},
		{
			description: "rollout status random error",
			err:         errors.New("deployment test not found"),
			expected:    true,
		},
		{
			description: "rollout status parent context cancelled",
			err:         context.Canceled,
			expected:    true,
		},
		{
			description: "rollout status parent conetct timed out",
			err:         context.DeadlineExceeded,
			expected:    true,
		},
		{
			description: "rollout status nil error",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			actual := isErrAndNotRetryAble(test.err)
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}
