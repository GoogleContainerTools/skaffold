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

func cutomTestErr(err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf("running custom test command: %s", err),
			ErrCode: proto.StatusCode_TEST_CUSTOM_USER_ERR,
		},
	)
}

func parsingTestCommandErr(command string, err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf("unable to parse test command %s: %s", command, err),
			ErrCode: proto.StatusCode_TEST_CUSTOM_CMD_PARSE_ERR,
		},
	)
}

func commandNonZeroExitErr(err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf("command finished with non-0 exit code: %s", err),
			ErrCode: proto.StatusCode_TEST_CUSTOM_CMD_NON_ZERO_EXIT_ERR,
		},
	)
}

func commandTimedoutErr(err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf("command timed out: %s", err),
			ErrCode: proto.StatusCode_TEST_CUSTOM_CMD_TIMEDOUT_ERR,
		},
	)
}

func commandCancelledErr(err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf("command cancelled: %s", err),
			ErrCode: proto.StatusCode_TEST_CUSTOM_CMD_CANCELLED_ERR,
		},
	)
}

func commandExecutionErr(err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf("command execution error: %s", err),
			ErrCode: proto.StatusCode_TEST_CUSTOM_CMD_EXECUTION_ERR,
		},
	)
}

func commandExited(err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf("command exited: %s", err),
			ErrCode: proto.StatusCode_TEST_CUSTOM_CMD_EXITED_ERR,
		},
	)
}

func runCmdErr(err error) error {
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
			ErrCode: proto.StatusCode_TEST_CUSTOM_DEPS_CMD_ERR,
		},
	)
}

func dependencyOutputUnmarshallErr(err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf("unmarshalling dependency output into string array: %s", err),
			ErrCode: proto.StatusCode_TEST_CUSTOM_DEPS_UNMARSHALL_ERR,
		},
	)
}
