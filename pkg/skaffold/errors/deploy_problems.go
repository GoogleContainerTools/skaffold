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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/cluster"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/proto"
	"github.com/sirupsen/logrus"
)

var (
	// for testing
	currentConfig = kubectx.CurrentConfig
)

func suggestDeployFailedAction(opts config.SkaffoldOptions) []*proto.Suggestion {
	kubeconfig, parsederr := currentConfig()
	logrus.Debugf("Error retrieving the config: %q", parsederr)

	var curctx = kubeconfig.CurrentContext
	var isminikube = cluster.GetClient().IsMinikube(opts.KubeContext)

	if isminikube {
		if curctx == "minkube" {
			// Check if minikube is running using `minikube status` command and try again
			return []*proto.Suggestion{{
				SuggestionCode: proto.SuggestionCode_CHECK_MINIKUBE_STAUTUS,
				Action:         "Check if minikube is running using `minikube status` command and try again",
			}}
		} else {
			// Check if minikube is running using `minikube status -p cloud-run-dev-internal` command and try again.
			return []*proto.Suggestion{{
				SuggestionCode: proto.SuggestionCode_CHECK_MINIKUBE_STAUTUS,
				Action:         "Check if minikube is running using `minikube status -p <clustername>` command and try again.",
			}}
		}
	}

	return []*proto.Suggestion{{
		SuggestionCode: proto.SuggestionCode_CHECK_CLUSTER_CONNECTION,
		Action:         "Check your cluster connection",
	}}
}
