/*
Copyright 2022 The Skaffold Authors

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

package build

import (
	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
)

func noRegistryForMultiplatformBuildErr(err error) error {
	return sErrors.NewError(err,
		&proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_BUILD_CROSS_PLATFORM_NO_REGISTRY_ERR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_SET_PUSH_AND_CONTAINER_REGISTRY,
					Action:         "To run multi-platform builds, set --push to true and set a container registry with --default-repo or in the global config",
				},
			},
		})
}
