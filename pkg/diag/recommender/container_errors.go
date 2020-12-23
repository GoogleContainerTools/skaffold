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

type ContainerError struct {
}

var (
	NilSuggestion = proto.Suggestion{SuggestionCode: proto.SuggestionCode_NIL}
)

func (r ContainerError) Make(errCode proto.StatusCode) proto.Suggestion {
	switch errCode {
	case proto.StatusCode_STATUSCHECK_CONTAINER_TERMINATED:
		return proto.Suggestion{
			SuggestionCode: proto.SuggestionCode_CHECK_CONTAINER_LOGS,
			Action:         "Try checking container logs",
		}
	case proto.StatusCode_STATUSCHECK_UNHEALTHY:
		return proto.Suggestion{
			SuggestionCode: proto.SuggestionCode_CHECK_READINESS_PROBE,
			Action:         "Try checking container config `readinessProbe`",
		}
	case proto.StatusCode_STATUSCHECK_IMAGE_PULL_ERR:
		return proto.Suggestion{
			SuggestionCode: proto.SuggestionCode_CHECK_CONTAINER_IMAGE,
			Action:         "Try checking container config `image`",
		}
	}
	return NilSuggestion
}
