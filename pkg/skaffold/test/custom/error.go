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

package custom

import (
	"fmt"

	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
)

func cmdRunRetrieveErr(command string, imageName string, err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf("retrieving cmd %s: %s", command, err),
			ErrCode: proto.StatusCode_TEST_CUSTOM_CMD_RETRIEVE_ERR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_CHECK_TEST_COMMAND_AND_IMAGE_NAME,
					Action:         fmt.Sprintf("Check the image name: %q and the command: %q", command, imageName),
				},
			},
		},
	)
}

func cmdRunParsingErr(command string, err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf("unable to parse test command %s: %s", command, err),
			ErrCode: proto.StatusCode_TEST_CUSTOM_CMD_PARSE_ERR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_CHECK_CUSTOM_COMMAND,
					Action:         fmt.Sprintf("Check the custom command contents: %q", command),
				},
			},
		},
	)
}

func cmdRunNonZeroExitErr(command string, err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf("command %s finished with non-0 exit code: %s", command, err),
			ErrCode: proto.StatusCode_TEST_CUSTOM_CMD_RUN_NON_ZERO_EXIT_ERR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_CHECK_CUSTOM_COMMAND,
					Action:         fmt.Sprintf("Check the custom command contents: %q", command),
				},
			},
		},
	)
}

func cmdRunTimedoutErr(timeoutSeconds int, err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf("command timed out: %s", err),
			ErrCode: proto.StatusCode_TEST_CUSTOM_CMD_RUN_TIMEDOUT_ERR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_FIX_CUSTOM_COMMAND_TIMEOUT,
					Action:         fmt.Sprintf("Fix the custom command timeoutSeconds: %d", timeoutSeconds),
				},
			},
		},
	)
}

func cmdRunCancelledErr(err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf("command cancelled: %s", err),
			ErrCode: proto.StatusCode_TEST_CUSTOM_CMD_RUN_CANCELLED_ERR,
		},
	)
}

func cmdRunExecutionErr(err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf("command execution error: %s", err),
			ErrCode: proto.StatusCode_TEST_CUSTOM_CMD_RUN_EXECUTION_ERR,
		},
	)
}

func cmdRunExited(err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf("command exited: %s", err),
			ErrCode: proto.StatusCode_TEST_CUSTOM_CMD_RUN_EXITED_ERR,
		},
	)
}

func cmdRunErr(err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf("error running cmd: %s", err),
			ErrCode: proto.StatusCode_TEST_CUSTOM_CMD_RUN_ERR,
		},
	)
}

func gettingDependenciesCommandErr(command string, err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf("getting dependencies from command: %s: %s", command, err),
			ErrCode: proto.StatusCode_TEST_CUSTOM_DEPENDENCIES_CMD_ERR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_CHECK_CUSTOM_COMMAND_DEPENDENCIES_CMD,
					Action:         fmt.Sprintf("Check the custom command dependencies command: %q", command),
				},
			},
		},
	)
}

func dependencyOutputUnmarshallErr(paths []string, err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf("unmarshalling dependency output into string array: %s", err),
			ErrCode: proto.StatusCode_TEST_CUSTOM_DEPENDENCIES_UNMARSHALL_ERR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_CHECK_CUSTOM_COMMAND_DEPENDENCIES_PATHS,
					Action:         fmt.Sprintf("Check the custom command dependencies paths: %q", paths),
				},
			},
		},
	)
}
