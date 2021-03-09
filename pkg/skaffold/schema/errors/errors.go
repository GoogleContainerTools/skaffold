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

package errors

import (
	"fmt"

	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
)

// ConfigParsingError returns a generic config parsing error
func ConfigParsingError(err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: "error parsing skaffold configuration file",
			ErrCode: proto.StatusCode_CONFIG_FILE_PARSING_ERR,
		})
}

// MainConfigFileNotFoundErr specifies main configuration file not found
func MainConfigFileNotFoundErr(file string, err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf("unable to find configuration file %q", file),
			ErrCode: proto.StatusCode_CONFIG_FILE_NOT_FOUND_ERR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_CONFIG_CHECK_FILE_PATH,
					Action:         fmt.Sprintf("Check that the specified configuration file exists at %q", file),
				},
			},
		})
}

// DependencyConfigFileNotFoundErr specifies dependency configuration file not found
func DependencyConfigFileNotFoundErr(depFile, parentFile string, err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf("could not find skaffold config file %q that is referenced as a dependency in config file %q", depFile, parentFile),
			ErrCode: proto.StatusCode_CONFIG_DEPENDENCY_NOT_FOUND_ERR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_CONFIG_CHECK_DEPENDENCY_DEFINITION,
					Action:         fmt.Sprintf("Modify the `requires` definition in the configuration file %q to point to the correct path for the dependency %q", parentFile, depFile),
				},
			},
		})
}

// BadConfigFilterErr specifies no configs matched the configs filter
func BadConfigFilterErr(filter []string) error {
	msg := fmt.Sprintf("did not find any configs matching selection %v", filter)
	return sErrors.NewError(fmt.Errorf(msg),
		proto.ActionableErr{
			Message: msg,
			ErrCode: proto.StatusCode_CONFIG_BAD_FILTER_ERR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_CONFIG_CHECK_FILTER,
					Action:         "Check that the arguments to the `-m` or `--module` flag are valid config names",
				},
			},
		})
}

// ZeroConfigsParsedErr specifies that the config file is empty
func ZeroConfigsParsedErr(file string) error {
	msg := fmt.Sprintf("failed to get any valid configs from file %q", file)
	return sErrors.NewError(fmt.Errorf(msg),
		proto.ActionableErr{
			Message: msg,
			ErrCode: proto.StatusCode_CONFIG_ZERO_FOUND_ERR,
		})
}

// DuplicateConfigNamesInSameFileErr specifies that multiple configs have the same name in current config file
func DuplicateConfigNamesInSameFileErr(config, file string) error {
	msg := fmt.Sprintf("multiple skaffold configs named %q found in file %q", config, file)
	return sErrors.NewError(fmt.Errorf(msg),
		proto.ActionableErr{
			Message: msg,
			ErrCode: proto.StatusCode_CONFIG_DUPLICATE_NAMES_SAME_FILE_ERR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_CONFIG_CHANGE_NAMES,
					Action:         fmt.Sprintf("Change the name of one of the occurrences of config %q in file %q to make it unique", config, file),
				},
			},
		})
}

// DuplicateConfigNamesAcrossFilesErr specifies that multiple configs have the same name in different files
func DuplicateConfigNamesAcrossFilesErr(config, file1, file2 string) error {
	msg := fmt.Sprintf("skaffold config named %q found in multiple files: %q and %q", config, file1, file2)
	return sErrors.NewError(fmt.Errorf(msg),
		proto.ActionableErr{
			Message: msg,
			ErrCode: proto.StatusCode_CONFIG_DUPLICATE_NAMES_ACROSS_FILES_ERR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_CONFIG_CHANGE_NAMES,
					Action:         fmt.Sprintf("Change the name of config %q in file %q or file %q to make it unique", config, file1, file2),
				},
			},
		})
}

// ConfigProfileActivationErr specifies that profile activation failed for this config
func ConfigProfileActivationErr(config, file string, err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf("failed to apply profiles to config %q defined in file %q", config, file),
			ErrCode: proto.StatusCode_CONFIG_APPLY_PROFILES_ERR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_CONFIG_CHECK_PROFILE_DEFINITION,
					Action:         fmt.Sprintf("There's an issue with one of the profiles defined in config %q in file %q; refer to the documentation on how to author valid profiles: https://skaffold.dev/docs/environment/profiles/", config, file),
				},
			},
		})
}

// ConfigSetDefaultValuesErr specifies that default values failed to be applied for this config
func ConfigSetDefaultValuesErr(config, file string, err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf("failed to set default values for config %q defined in file %q", config, file),
			ErrCode: proto.StatusCode_CONFIG_DEFAULT_VALUES_ERR,
		})
}

// ConfigSetAbsFilePathsErr specifies that substituting absolute filepaths failed for this config
func ConfigSetAbsFilePathsErr(config, file string, err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf("failed to set absolute filepaths for config %q defined in file %q", config, file),
			ErrCode: proto.StatusCode_CONFIG_FILE_PATHS_SUBSTITUTION_ERR,
		})
}

// ConfigProfileConflictErr specifies that the same config is imported with different set of profiles.
func ConfigProfileConflictErr(config, file string) error {
	msg := fmt.Sprintf("config %q defined in file %q imported multiple times with different set of profiles", config, file)
	return sErrors.NewError(fmt.Errorf(msg),
		proto.ActionableErr{
			Message: msg,
			ErrCode: proto.StatusCode_CONFIG_MULTI_IMPORT_PROFILE_CONFLICT_ERR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_CONFIG_CHECK_DEPENDENCY_PROFILES_SELECTION,
					Action:         "Check that all occurrences of the specified config dependency use the same set of profiles; refer to the documentation on how to author config dependencies: https://skaffold.dev/docs/design/config/#configuration-dependencies",
				},
			},
		})
}
