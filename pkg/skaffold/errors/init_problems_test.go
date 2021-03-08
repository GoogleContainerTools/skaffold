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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
)

var (
	initTestCases = []struct {
		description string
		opts        config.SkaffoldOptions
		phase       Phase
		context     *config.ContextConfig
		err         error
		expected    string
		expectedAE  *proto.ActionableErr
	}{
		{
			description: "creating tagger error",
			context:     &config.ContextConfig{},
			phase:       Init,
			err:         fmt.Errorf("creating tagger: something went wrong"),
			expected:    "creating tagger: something went wrong\n. If above error is unexpected, please open an issue https://github.com/GoogleContainerTools/skaffold/issues/new to report this error.",
			expectedAE: &proto.ActionableErr{
				ErrCode:     proto.StatusCode_INIT_CREATE_TAGGER_ERROR,
				Message:     "creating tagger: something went wrong",
				Suggestions: reportIssueSuggestion(dummyRunCtx),
			},
		},
		{
			description: "minikube not started error",
			context:     &config.ContextConfig{},
			phase:       Init,
			err:         fmt.Errorf("creating runner: creating builder: getting docker client: getting minikube env: running [/Users/tejaldesai/Downloads/google-cloud-sdk2/bin/minikube docker-env --shell none -p minikube]\n - stdout: \"* The control plane node must be running for this command\\n  - To fix this, run: \\\"minikube start\\\"\\n\"\n - stderr: \"\"\n - cause: exit status 89"),
			expected:    "minikube is probably not running. Try running \"minikube start\".",
			expectedAE: &proto.ActionableErr{
				ErrCode: proto.StatusCode_INIT_MINIKUBE_NOT_RUNNING_ERROR,
				Message: "creating runner: creating builder: getting docker client: getting minikube env: running [/Users/tejaldesai/Downloads/google-cloud-sdk2/bin/minikube docker-env --shell none -p minikube]\n - stdout: \"* The control plane node must be running for this command\\n  - To fix this, run: \\\"minikube start\\\"\\n\"\n - stderr: \"\"\n - cause: exit status 89",
				Suggestions: []*proto.Suggestion{{
					SuggestionCode: proto.SuggestionCode_START_MINIKUBE,
					Action:         "Try running \"minikube start\"",
				}},
			},
		},
		{
			description: "create builder error",
			context:     &config.ContextConfig{},
			phase:       Init,
			err:         fmt.Errorf("creating runner: creating builder: something went wrong"),
			expected:    "creating runner: creating builder: something went wrong\n. If above error is unexpected, please open an issue https://github.com/GoogleContainerTools/skaffold/issues/new to report this error.",
			expectedAE: &proto.ActionableErr{
				ErrCode:     proto.StatusCode_INIT_CREATE_BUILDER_ERROR,
				Message:     "creating runner: creating builder: something went wrong",
				Suggestions: reportIssueSuggestion(dummyRunCtx),
			},
		},
		{
			description: "build dependency error",
			context:     &config.ContextConfig{},
			phase:       Init,
			err:         fmt.Errorf("creating runner: unexpected artifact type `DockerrArtifact`"),
			expected:    "creating runner: unexpected artifact type `DockerrArtifact`\n. If above error is unexpected, please open an issue https://github.com/GoogleContainerTools/skaffold/issues/new to report this error.",
			expectedAE: &proto.ActionableErr{
				ErrCode:     proto.StatusCode_INIT_CREATE_ARTIFACT_DEP_ERROR,
				Message:     "creating runner: unexpected artifact type `DockerrArtifact`",
				Suggestions: reportIssueSuggestion(dummyRunCtx),
			},
		},
		{
			description: "test dependency error",
			context:     &config.ContextConfig{},
			phase:       Init,
			err:         fmt.Errorf("creating runner: expanding test file paths: .src/test"),
			expected:    "creating runner: expanding test file paths: .src/test\n. If above error is unexpected, please open an issue https://github.com/GoogleContainerTools/skaffold/issues/new to report this error.",
			expectedAE: &proto.ActionableErr{
				ErrCode:     proto.StatusCode_INIT_CREATE_TEST_DEP_ERROR,
				Message:     "creating runner: expanding test file paths: .src/test",
				Suggestions: reportIssueSuggestion(dummyRunCtx),
			},
		},
		{
			description: "init cache error",
			context:     &config.ContextConfig{},
			phase:       Init,
			err:         fmt.Errorf("creating runner: initializing cache at some error"),
			expected:    "creating runner: initializing cache at some error\n. If above error is unexpected, please open an issue https://github.com/GoogleContainerTools/skaffold/issues/new to report this error.",
			expectedAE: &proto.ActionableErr{
				ErrCode:     proto.StatusCode_INIT_CACHE_ERROR,
				Message:     "creating runner: initializing cache at some error",
				Suggestions: reportIssueSuggestion(dummyRunCtx),
			},
		},
	}
)
