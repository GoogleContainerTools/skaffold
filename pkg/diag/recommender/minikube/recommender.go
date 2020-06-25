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

package minikube

import (

	"github.com/GoogleContainerTools/skaffold/proto"
)

type Recommender struct {
}

func (r Recommender) Make(ae proto.ActionableErr) {
	// Suggestion actions for all K8 infra error
	switch ae.ErrCode {
	case proto.StatusCode_STATUSCHECK_NODE_DISK_PRESSURE:
		ae.Suggestions = append(ae.Suggestions, proto.Suggestion{
			SuggestionCode:
		})
	case proto.StatusCode_STATUSCHECK_NODE_MEMORY_PRESSURE:

	case proto.StatusCode_STATUSCHECK_NODE_NETWORK_UNAVAILABLE:

	case proto.StatusCode_STATUSCHECK_NODE_PID_PRESSURE:

	case proto.StatusCode_STATUSCHECK_NODE_UNSCHEDULABLE:
	case proto.StatusCode_STATUSCHECK_NODE_UNREACHABLE:

	case proto.StatusCode_STATUSCHECK_NODE_NOT_READY:

	case proto.StatusCode_STATUSCHECK_FAILED_SCHEDULING:

	}
}
