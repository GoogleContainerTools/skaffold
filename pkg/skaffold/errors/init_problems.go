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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
)

var knownInitProblems = []problem{
	{
		regexp:     re(".*creating tagger.*"),
		errCode:    proto.StatusCode_INIT_CREATE_TAGGER_ERROR,
		suggestion: reportIssueSuggestion,
	},
	{
		regexp:      re(".*The control plane node must be running for this command.*"),
		errCode:     proto.StatusCode_INIT_MINIKUBE_NOT_RUNNING_ERROR,
		description: func(error) string { return "minikube is probably not running" },
		suggestion: func(runcontext.RunContext) []*proto.Suggestion {
			return []*proto.Suggestion{{
				SuggestionCode: proto.SuggestionCode_START_MINIKUBE,
				Action:         `Try running "minikube start"`,
			}}
		},
	},
	{
		regexp:     re(".*creating builder.*"),
		errCode:    proto.StatusCode_INIT_CREATE_BUILDER_ERROR,
		suggestion: reportIssueSuggestion,
	},
	{
		regexp:     re(".*unexpected artifact type.*"),
		errCode:    proto.StatusCode_INIT_CREATE_ARTIFACT_DEP_ERROR,
		suggestion: reportIssueSuggestion,
	},
	{
		regexp:     re(".*expanding test file paths.*"),
		errCode:    proto.StatusCode_INIT_CREATE_TEST_DEP_ERROR,
		suggestion: reportIssueSuggestion,
	},
	{
		regexp:     re(".*creating deployer: something went wrong"),
		errCode:    proto.StatusCode_INIT_CREATE_DEPLOYER_ERROR,
		suggestion: reportIssueSuggestion,
	},
	{
		regexp:     re(".*creating watch trigger.*"),
		errCode:    proto.StatusCode_INIT_CREATE_WATCH_TRIGGER_ERROR,
		suggestion: reportIssueSuggestion,
	},
	{
		regexp:     re(".* initializing cache.*"),
		errCode:    proto.StatusCode_INIT_CACHE_ERROR,
		suggestion: reportIssueSuggestion,
	},
}
