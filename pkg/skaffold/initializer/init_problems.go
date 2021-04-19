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
	"regexp"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
)

// re is a shortcut around regexp.MustCompile
func re(s string) *regexp.Regexp {
	return regexp.MustCompile(s)
}

func init() {
	sErrors.AddPhaseProblems(constants.Init, []sErrors.Problem{
		{
			Regexp:     re(".*creating tagger.*"),
			ErrCode:    proto.StatusCode_INIT_CREATE_TAGGER_ERROR,
			Suggestion: sErrors.ReportIssueSuggestion,
		},
		{
			Regexp:      re(".*The control plane node must be running for this command.*"),
			ErrCode:     proto.StatusCode_INIT_MINIKUBE_NOT_RUNNING_ERROR,
			Description: func(error) string { return "minikube is probably not running" },
			Suggestion: func(_ interface{}) []*proto.Suggestion {
				return []*proto.Suggestion{{
					SuggestionCode: proto.SuggestionCode_START_MINIKUBE,
					Action:         `Try running "minikube start"`,
				}}
			},
		},
		{
			Regexp:     re(".*creating builder.*"),
			ErrCode:    proto.StatusCode_INIT_CREATE_BUILDER_ERROR,
			Suggestion: sErrors.ReportIssueSuggestion,
		},
		{
			Regexp:     re(".*unexpected artifact type.*"),
			ErrCode:    proto.StatusCode_INIT_CREATE_ARTIFACT_DEP_ERROR,
			Suggestion: sErrors.ReportIssueSuggestion,
		},
		{
			Regexp:     re(".*expanding test file paths.*"),
			ErrCode:    proto.StatusCode_INIT_CREATE_TEST_DEP_ERROR,
			Suggestion: sErrors.ReportIssueSuggestion,
		},
		{
			Regexp:     re(".*creating deployer: something went wrong"),
			ErrCode:    proto.StatusCode_INIT_CREATE_DEPLOYER_ERROR,
			Suggestion: sErrors.ReportIssueSuggestion,
		},
		{
			Regexp:     re(".*creating watch trigger.*"),
			ErrCode:    proto.StatusCode_INIT_CREATE_WATCH_TRIGGER_ERROR,
			Suggestion: sErrors.ReportIssueSuggestion,
		},
		{
			Regexp:     re(".* initializing cache.*"),
			ErrCode:    proto.StatusCode_INIT_CACHE_ERROR,
			Suggestion: sErrors.ReportIssueSuggestion,
		},
	})
}
