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

package deploy

import (
	"fmt"
	"regexp"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/types"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
)

const (
	defaultMinikubeProfile = "minikube"
)

var (
	ClusterInternalSystemErr = regexp.MustCompile(".*Internal Server Error")
	clusterConnectionErr     = regexp.MustCompile("(?i).*unable to connect.*: Get (.*)")
)

func suggestDeployFailedAction(cfg interface{}) []*proto.Suggestion {
	deployCfg, ok := cfg.(types.Config)
	if !ok {
		return nil
	}
	if deployCfg.MinikubeProfile() != "" {
		return []*proto.Suggestion{
			checkMinikubeSuggestion(deployCfg),
		}
	}
	return []*proto.Suggestion{{
		SuggestionCode: proto.SuggestionCode_CHECK_CLUSTER_CONNECTION,
		Action:         "Check your connection for the cluster",
	}}
}

func init() {
	sErrors.AddPhaseProblems(constants.Deploy, []sErrors.Problem{
		{
			Regexp:  clusterConnectionErr,
			ErrCode: proto.StatusCode_DEPLOY_CLUSTER_CONNECTION_ERR,
			Description: func(err error) string {
				if match := clusterConnectionErr.FindStringSubmatch(err.Error()); len(match) >= 2 {
					return fmt.Sprintf("Deploy Failed. Could not connect to cluster due to %s", match[1])
				}
				return "Deploy Failed. Could not connect to cluster."
			},
			Suggestion: suggestDeployFailedAction,
		},
		{
			Regexp:  ClusterInternalSystemErr,
			ErrCode: proto.StatusCode_DEPLOY_CLUSTER_INTERNAL_SYSTEM_ERR,
			Description: func(err error) string {
				return fmt.Sprintf("Deploy Failed. %v", err)
			},
			Suggestion: func(cfg interface{}) []*proto.Suggestion {
				deployCfg, ok := cfg.(types.Config)
				if !ok {
					return nil
				}
				if deployCfg.MinikubeProfile() != "" {
					return []*proto.Suggestion{
						checkMinikubeSuggestion(deployCfg),
						{
							SuggestionCode: proto.SuggestionCode_OPEN_ISSUE,
							// TODO: show tip to run minikube logs command and attach logs.
							Action: fmt.Sprintf("open an issue at %s", constants.GithubIssueLink),
						}}
				}
				return []*proto.Suggestion{{
					SuggestionCode: proto.SuggestionCode_OPEN_ISSUE,
					Action:         fmt.Sprintf("Something went wrong with your cluster %q. Try again.\nIf this keeps happening please open an issue at %s", deployCfg.GetKubeContext(), constants.GithubIssueLink),
				}}
			},
		},
	})
}

func checkMinikubeSuggestion(cfg types.Config) *proto.Suggestion {
	return &proto.Suggestion{
		SuggestionCode: proto.SuggestionCode_CHECK_MINIKUBE_STATUS,
		Action: fmt.Sprintf("Check if minikube is running using %q command and try again",
			getMinikubeStatusCommand(cfg.GetKubeContext())),
	}
}

func getMinikubeStatusCommand(p string) string {
	if p == defaultMinikubeProfile {
		return "minikube status"
	}
	return fmt.Sprintf("minikube status -p %s", p)
}
