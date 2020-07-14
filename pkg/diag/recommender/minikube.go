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


package recommender

import (
	"github.com/GoogleContainerTools/skaffold/proto"
)

type Minikube struct {}

func (r *Minikube) Make(errCode proto.StatusCode) proto.Suggestion {
	switch errCode {
	case proto.StatusCode_STATUSCHECK_NODE_MEMORY_PRESSURE:
		return proto.Suggestion{
			SuggestionCode: proto.SuggestionCode_ADDRESS_NODE_MEMORY_PRESSURE,
			Action:         "Try checking container logs",
		}
	case proto.StatusCode_STATUSCHECK_NODE_DISK_PRESSURE:
		return proto.Suggestion{
			SuggestionCode: proto.SuggestionCode_ADDRESS_NODE_DISK_PRESSURE,
			Action:         "Try checking container config `readinessProbe`",
		}
	case proto.StatusCode_STATUSCHECK_NODE_NETWORK_UNAVAILABLE:
		return proto.Suggestion{
			SuggestionCode: proto.SuggestionCode_ADDRESS_NODE_NETWORK_UNAVAILABLE,
			Action:         "Try checking container config `image`",
		}
	case proto.StatusCode_STATUSCHECK_NODE_PID_PRESSURE:
		return proto.Suggestion{
			SuggestionCode: proto.SuggestionCode_ADDRESS_NODE_PID_PRESSURE,
			Action:         "Try checking container config `image`",
		}
	case proto.StatusCode_STATUSCHECK_NODE_NOT_READY:
		return proto.Suggestion{
			SuggestionCode: proto.SuggestionCode_ADDRESS_NODE_NOT_READY,
			Action:         "Try checking container config `image`",
		}
	case proto.StatusCode_STATUSCHECK_NODE_UNSCHEDULABLE:
		return proto.Suggestion{
			SuggestionCode: proto.SuggestionCode_ADDRESS_NODE_UNSCHEDULABLE,
			Action:         "Try checking container config `image`",
		}
	default:
		return NilSuggestion
	}
}



