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

func suggestDeployFailedAction(cfg interface{}) []*proto.Suggestion {
	deployCfg, ok := cfg.(types.Config)
	if !ok {
		return nil
	}
	kCtx := deployCfg.GetKubeContext()
	isMinikube := deployCfg.MinikubeProfile() != ""
	if isMinikube {
		command := "minikube status"
		if deployCfg.GetKubeContext() != "minikube" {
			command = fmt.Sprintf("minikube status -p %s", kCtx)
		}
		return []*proto.Suggestion{{
			SuggestionCode: proto.SuggestionCode_CHECK_MINIKUBE_STATUS,
			Action:         fmt.Sprintf("Check if minikube is running using %q command and try again.", command),
		}}
	}

	return []*proto.Suggestion{{
		SuggestionCode: proto.SuggestionCode_CHECK_CLUSTER_CONNECTION,
		Action:         "Check your connection for the cluster",
	}}
}

// re is a shortcut around regexp.MustCompile
func re(s string) *regexp.Regexp {
	return regexp.MustCompile(s)
}

func init() {
	sErrors.AddPhaseProblems(constants.Deploy, []sErrors.Problem{
		{
			Regexp:  re("(?i).*unable to connect.*: Get (.*)"),
			ErrCode: proto.StatusCode_DEPLOY_CLUSTER_CONNECTION_ERR,
			Description: func(err error) string {
				matchExp := re("(?i).*unable to connect.*Get (.*)")
				if match := matchExp.FindStringSubmatch(fmt.Sprintf("%s", err)); len(match) >= 2 {
					return fmt.Sprintf("Deploy Failed. Could not connect to cluster due to %s", match[1])
				}
				return "Deploy Failed. Could not connect to cluster."
			},
			Suggestion: suggestDeployFailedAction,
		},
	})
}
