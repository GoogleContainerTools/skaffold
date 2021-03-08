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
	"errors"
	"fmt"
	"testing"

	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/proto/enums"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestErrors(t *testing.T) {
	tests := []struct {
		err            error
		errCode        enums.StatusCode
		suggestionCode enums.SuggestionCode
	}{
		{
			err:     ConfigParsingError(nil),
			errCode: proto.StatusCode_CONFIG_FILE_PARSING_ERR,
		},
		{
			err:            MainConfigFileNotFoundErr("", nil),
			errCode:        proto.StatusCode_CONFIG_FILE_NOT_FOUND_ERR,
			suggestionCode: proto.SuggestionCode_CONFIG_CHECK_FILE_PATH,
		},
		{
			err:            DependencyConfigFileNotFoundErr("", "", nil),
			errCode:        proto.StatusCode_CONFIG_DEPENDENCY_NOT_FOUND_ERR,
			suggestionCode: proto.SuggestionCode_CONFIG_CHECK_DEPENDENCY_DEFINITION,
		},
		{
			err:            BadConfigFilterErr(nil),
			errCode:        proto.StatusCode_CONFIG_BAD_FILTER_ERR,
			suggestionCode: proto.SuggestionCode_CONFIG_CHECK_FILTER,
		},
		{
			err:     ZeroConfigsParsedErr(""),
			errCode: proto.StatusCode_CONFIG_ZERO_FOUND_ERR,
		},
		{
			err:            DuplicateConfigNamesInSameFileErr("", ""),
			errCode:        proto.StatusCode_CONFIG_DUPLICATE_NAMES_SAME_FILE_ERR,
			suggestionCode: proto.SuggestionCode_CONFIG_CHANGE_NAMES,
		},
		{
			err:            DuplicateConfigNamesAcrossFilesErr("", "", ""),
			errCode:        proto.StatusCode_CONFIG_DUPLICATE_NAMES_ACROSS_FILES_ERR,
			suggestionCode: proto.SuggestionCode_CONFIG_CHANGE_NAMES,
		},
		{
			err:            ConfigProfileActivationErr("", "", nil),
			errCode:        proto.StatusCode_CONFIG_APPLY_PROFILES_ERR,
			suggestionCode: proto.SuggestionCode_CONFIG_CHECK_PROFILE_DEFINITION,
		},
		{
			err:     ConfigSetDefaultValuesErr("", "", nil),
			errCode: proto.StatusCode_CONFIG_DEFAULT_VALUES_ERR,
		},
		{
			err:     ConfigSetAbsFilePathsErr("", "", nil),
			errCode: proto.StatusCode_CONFIG_FILE_PATHS_SUBSTITUTION_ERR,
		},
		{
			err:            ConfigProfileConflictErr("", ""),
			errCode:        proto.StatusCode_CONFIG_MULTI_IMPORT_PROFILE_CONFLICT_ERR,
			suggestionCode: proto.SuggestionCode_CONFIG_CHECK_DEPENDENCY_PROFILES_SELECTION,
		},
	}

	for i, test := range tests {
		testutil.Run(t, fmt.Sprintf("test error definition %d", i), func(t *testutil.T) {
			var e sErrors.Error
			if errors.As(test.err, &e) {
				t.CheckDeepEqual(e.StatusCode(), test.errCode)
				if test.suggestionCode != proto.SuggestionCode_NIL {
					t.CheckDeepEqual(1, len(e.Suggestions()))
					t.CheckDeepEqual(test.suggestionCode, e.Suggestions()[0].SuggestionCode)
				}
			} else {
				t.Fail()
			}
		})
	}
}
