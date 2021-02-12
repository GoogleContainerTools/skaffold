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

	"github.com/docker/docker/client"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNewRunner(t *testing.T) {
	const (
		imageName = "foo.io/baz"
	)

	testutil.Run(t, "", func(t *testutil.T) {
		tmpDir := t.NewTempDir().Touch("test.yaml")
		t.Override(&util.DefaultExecCommand, testutil.CmdRun("container-structure-test test -v warn --image "+imageName+" --config "+tmpDir.Path("test.yaml")))

		workingDir := tmpDir.Root()
		structureTests := []string{"test.yaml"}

		localDaemon := fakeLocalDaemon(&testutil.FakeAPIClient{ErrImagePull: true})

		testRunner := NewRunner(structureTests, workingDir, localDaemon, func(imageName string) (bool, error) { return true, nil })
		err := testRunner.Test(context.Background(), ioutil.Discard, imageName, []build.Artifact{{
			ImageName: "image",
			Tag:       "image:tag",
		}})
		t.CheckNoError(err)
	})
}

func fakeLocalDaemon(api client.CommonAPIClient) docker.LocalDaemon {
	return docker.NewLocalDaemon(api, nil, false, nil)
}
