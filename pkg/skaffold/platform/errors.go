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

package platform

import (
	"fmt"

	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
)

// UnknownPlatformCLIFlag specifies that the platform provided via CLI flag couldn't be parsed
func UnknownPlatformCLIFlag(platform string, err error) error {
	return sErrors.NewError(err,
		&proto.ActionableErr{
			Message: fmt.Sprintf("unable to recognise platform %q: %v", platform, err),
			ErrCode: proto.StatusCode_BUILD_UNKNOWN_PLATFORM_FLAG,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_BUILD_FIX_UNKNOWN_PLATFORM_FLAG,
					Action:         "Check that the value provided to --platform flag is a valid platform and formatted correctly, like linux/amd64, linux/arm64, linux/arm/v7, etc.",
				},
			},
		})
}
