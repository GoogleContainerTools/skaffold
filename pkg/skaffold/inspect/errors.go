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
	proto "github.com/GoogleContainerTools/skaffold/proto/v1"
)

// BuildEnvAlreadyExists specifies that there's an existing build environment definition for the same type.
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
					SuggestionCode: proto.SuggestionCode_INSPECT_USE_MODIFY_OR_NEW_PROFILE,
					Action:         "Use the `modify` command instead of the `add` command to overwrite fields for an already existing build environment type. Otherwise pass the `--profile` flag with a unique name to create the new build environment definition in a new profile instead",
				},
			},
		})
}

// BuildEnvNotFound specifies that the target build environment definition doesn't exist.
func BuildEnvNotFound(b BuildEnv, filename string, profile string) error {
	var msg string
	if profile == "" {
		msg = fmt.Sprintf("trying to modify a %q build environment definition that doesn't exist, in file %s", b, filename)
	} else {
		msg = fmt.Sprintf("trying to modify a %q build environment definition that doesn't exist, in profile %q in file %s", b, profile, filename)
	}
	return sErrors.NewError(fmt.Errorf(msg),
		proto.ActionableErr{
			Message: msg,
			ErrCode: proto.StatusCode_INSPECT_BUILD_ENV_INCORRECT_TYPE_ERR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_INSPECT_USE_ADD_BUILD_ENV,
					Action:         "Check that the target build environment definition already exists. Otherwise use the `add` command instead of the `modify` command to create it",
				},
			},
		})
}

// ProfileNotFound specifies that the target profile doesn't exist
func ProfileNotFound(profile string) error {
	msg := fmt.Sprintf("trying to modify a profile %q that doesn't exist", profile)
	return sErrors.NewError(fmt.Errorf(msg),
		proto.ActionableErr{
			Message: msg,
			ErrCode: proto.StatusCode_INSPECT_PROFILE_NOT_FOUND_ERR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_INSPECT_CHECK_INPUT_PROFILE,
					Action:         "Check that the `--profile` flag matches at least one existing profile. Otherwise use the `add` command instead of the `modify` command to create it",
				},
			},
		})
}
