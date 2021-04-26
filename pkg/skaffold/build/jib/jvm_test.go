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

package jib

import (
	"errors"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestResolveJVM(t *testing.T) {
	tests := []struct {
		name string
		cmd      *testutil.FakeCmd
		expected bool
	}{
		{name: "found", cmd: testutil.CmdRun("java -version"), expected: true},
		{name: "not found", cmd: testutil.CmdRunErr("java -version", errors.New("not found")), expected: false},
	}
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.cmd)

			result := resolveJVM()
			t.CheckDeepEqual(test.expected, result)
		})
	}
}
