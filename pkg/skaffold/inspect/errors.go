/*
Copyright 2021 The Skaffold Authors

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

package inspect

import (
	"fmt"

	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
)

// BadConfigFilterErr specifies no configs matched the configs filter
func BuildEnvAlreadyExists(b BuildEnv, filename string, profile string) error {
	var msg string
	if profile == "" {
		msg = fmt.Sprintf("trying to create a %q build environment definition that already exists, in file %s", b, filename)
	} else {
		msg = fmt.Sprintf("trying to create a %q build environment definition that already exists, in profile %q in file %s", b, profile, filename)
	}
	return sErrors.NewError(fmt.Errorf(msg),
		proto.ActionableErr{
			Message: msg,
			ErrCode: proto.StatusCode_INSPECT_BUILD_ENV_ALREADY_EXISTS_ERR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_INSPECT_DEDUP_NEW_BUILD_ENV,
					Action:         "Use the `modify` command instead of the `add` command to overwrite fields for an already existing build environment type. Otherwise pass the `--profile` flag with a unique name to create the new build environment definition in a new profile instead",
				},
			},
		})
}
