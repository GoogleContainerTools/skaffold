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

package kubernetes

import (
	"reflect"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGetDebugger(t *testing.T) {
	tests := []struct {
		description string
		runMode     config.RunMode
		isNoop      bool
	}{
		{
			description: "unspecified run mode defaults to disabled",
			isNoop:      true,
		},
		{
			description: "run mode set to debug",
			runMode:     config.RunModes.Debug,
		},
		{
			description: "run mode set to dev",
			runMode:     config.RunModes.Dev,
			isNoop:      true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			d := NewDebugger(test.runMode, nil, nil, "")
			t.CheckDeepEqual(test.isNoop, reflect.Indirect(reflect.ValueOf(d)).Type() == reflect.TypeOf(debug.NoopDebugger{}))
		})
	}
}
