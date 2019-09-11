/*
Copyright 2019 The Skaffold Authors

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

package structure

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNewRunner(t *testing.T) {
	const (
		imageName         = "foo.io/baz"
		structureTestName = "foo.testcase"
	)

	testutil.Run(t, "", func(t *testutil.T) {
		extraEnv := []string{"SOME=env_var", "OTHER=env_value"}
		t.Override(&util.DefaultExecCommand, testutil.CmdRunEnv(
			"container-structure-test test -v warn --image "+imageName+" --config "+structureTestName,
			extraEnv,
		))

		testRunner := NewRunner([]string{structureTestName}, extraEnv)
		err := testRunner.Test(context.Background(), ioutil.Discard, imageName)
		t.CheckNoError(err)
	})
}
