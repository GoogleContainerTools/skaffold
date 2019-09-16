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

package local

import (
	"context"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestBazelBin(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.Override(&util.DefaultExecCommand, testutil.CmdRunOut(
			"bazel info bazel-bin --arg1 --arg2",
			"/absolute/path/bin\n",
		))

		bazelBin, err := bazelBin(context.Background(), ".", &latest.BazelArtifact{
			BuildArgs: []string{"--arg1", "--arg2"},
		})

		t.CheckNoError(err)
		t.CheckDeepEqual("/absolute/path/bin", bazelBin)
	})
}

func TestBuildTarPath(t *testing.T) {
	buildTarget := "//:skaffold_example.tar"

	tarPath := buildTarPath(buildTarget)

	testutil.CheckDeepEqual(t, "skaffold_example.tar", tarPath)
}

func TestBuildImageTag(t *testing.T) {
	buildTarget := "//:skaffold_example.tar"

	imageTag := buildImageTag(buildTarget)

	testutil.CheckDeepEqual(t, "bazel:skaffold_example", imageTag)
}
