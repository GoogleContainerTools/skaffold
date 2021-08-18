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
package app

import (
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestExitCode(t *testing.T) {
	testutil.CheckDeepEqual(t, 1, ExitCode(fmt.Errorf("some error")))
	testutil.CheckDeepEqual(t, 127, ExitCode(invalidUsageError{err: fmt.Errorf("some error")}))
	testutil.CheckDeepEqual(t, 127, ExitCode(fmt.Errorf("wrapped: %w", invalidUsageError{err: fmt.Errorf("some error")})))
}

func Test_extractInvalidUsageError(t *testing.T) {
	tests := []struct {
		name                string
		err                 error
		wantInvalidUsageErr bool
	}{
		{name: "nil err",
			err:                 nil,
			wantInvalidUsageErr: false},
		{name: "other error",
			err:                 fmt.Errorf("some error"),
			wantInvalidUsageErr: false},
		{name: "cobra unknown cmd error",
			err:                 fmt.Errorf(`unknown command "x" for "skaffold"`),
			wantInvalidUsageErr: true},
		{name: "cobra unknown shorthand flag error",
			err:                 fmt.Errorf(`unknown shorthand flag: 'x' in -x`),
			wantInvalidUsageErr: true},
		{name: "cobra unknown flag error",
			err:                 fmt.Errorf(`unknown flag: --unknown`),
			wantInvalidUsageErr: true},
		{name: "cobra argument validation error",
			err:                 fmt.Errorf(`invalid argument "1" for "-a, --build-artifacts" flag: stat 1: no such file or directory`),
			wantInvalidUsageErr: true},
		{name: "cobra exactargs validation error",
			err:                 fmt.Errorf(`accepts 2 arg(s), received 0`),
			wantInvalidUsageErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := extractInvalidUsageError(tt.err)
			_, ok := out.(invalidUsageError)
			if tt.wantInvalidUsageErr && !ok {
				t.Errorf("wanted invalidUsageError, got %T", out)
			} else if !tt.wantInvalidUsageErr && ok {
				t.Errorf("unwanted invalidUsageError: %v", out)
			}
		})
	}
}
