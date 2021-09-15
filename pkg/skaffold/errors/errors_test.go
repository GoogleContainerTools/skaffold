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

package errors

import (
	"fmt"
	"testing"

	"google.golang.org/protobuf/testing/protocmp"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

var (
	dummyRunCtx = runcontext.RunContext{}
)

func TestShowAIError(t *testing.T) {
	tests := []struct {
		description string
		opts        config.SkaffoldOptions
		phase       constants.Phase
		context     *config.ContextConfig
		err         error
		expected    string
		expectedAE  *proto.ActionableErr
	}{
		// unknown errors case
		{
			description: "build unknown error",
			context:     &config.ContextConfig{DefaultRepo: "docker.io/global"},
			phase:       constants.Build,
			err:         fmt.Errorf("build failed: something went wrong"),
			expected:    "build failed: something went wrong",
			expectedAE: &proto.ActionableErr{
				ErrCode:     proto.StatusCode_BUILD_UNKNOWN,
				Message:     "build failed: something went wrong",
				Suggestions: ReportIssueSuggestion(dummyRunCtx),
			},
		},
		{
			description: "deploy unknown error",
			phase:       constants.Deploy,
			err:         fmt.Errorf("deploy failed: something went wrong"),
			expected:    "deploy failed: something went wrong",
			expectedAE: &proto.ActionableErr{
				ErrCode:     proto.StatusCode_DEPLOY_UNKNOWN,
				Message:     "deploy failed: something went wrong",
				Suggestions: ReportIssueSuggestion(dummyRunCtx),
			},
		},
		{
			description: "file sync unknown error",
			phase:       constants.Sync,
			err:         fmt.Errorf("sync failed: something went wrong"),
			expected:    "sync failed: something went wrong",
			expectedAE: &proto.ActionableErr{
				ErrCode:     proto.StatusCode_SYNC_UNKNOWN,
				Message:     "sync failed: something went wrong",
				Suggestions: ReportIssueSuggestion(dummyRunCtx),
			},
		},
		{
			description: "init unknown error",
			phase:       constants.Init,
			err:         fmt.Errorf("init failed: something went wrong"),
			expected:    "init failed: something went wrong",
			expectedAE: &proto.ActionableErr{
				ErrCode:     proto.StatusCode_INIT_UNKNOWN,
				Message:     "init failed: something went wrong",
				Suggestions: ReportIssueSuggestion(dummyRunCtx),
			},
		},
		{
			description: "cleanup unknown error",
			phase:       constants.Cleanup,
			err:         fmt.Errorf("failed: something went wrong"),
			expected:    "failed: something went wrong",
			expectedAE: &proto.ActionableErr{
				ErrCode:     proto.StatusCode_CLEANUP_UNKNOWN,
				Message:     "failed: something went wrong",
				Suggestions: ReportIssueSuggestion(dummyRunCtx),
			},
		},
		{
			description: "status check unknown error",
			phase:       constants.StatusCheck,
			err:         fmt.Errorf("failed: something went wrong"),
			expected:    "failed: something went wrong",
			expectedAE: &proto.ActionableErr{
				ErrCode:     proto.StatusCode_STATUSCHECK_UNKNOWN,
				Message:     "failed: something went wrong",
				Suggestions: ReportIssueSuggestion(dummyRunCtx),
			},
		},
		{
			description: "dev init unknown error",
			phase:       constants.DevInit,
			err:         fmt.Errorf("failed: something went wrong"),
			expected:    "failed: something went wrong",
			expectedAE: &proto.ActionableErr{
				ErrCode:     proto.StatusCode_DEVINIT_UNKNOWN,
				Message:     "failed: something went wrong",
				Suggestions: ReportIssueSuggestion(dummyRunCtx),
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			runCtx := runcontext.RunContext{KubeContext: "test_cluster", Opts: test.opts}
			actual := ShowAIError(runCtx, test.err)
			t.CheckDeepEqual(test.expected, actual.Error())
			actualAE := ActionableErr(runCtx, test.phase, test.err)
			t.CheckDeepEqual(test.expectedAE, actualAE, protocmp.Transform())
		})
	}
}
