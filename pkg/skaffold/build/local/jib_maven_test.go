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

func TestMavenVerifyJibPackageGoal(t *testing.T) {
	var tests = []struct {
		description  string
		requiredGoal string
		mavenOutput  string
		shouldErr    bool
	}{
		{
			description:  "no goals should fail",
			requiredGoal: "xxx",
			mavenOutput:  "",
			shouldErr:    true,
		},
		{
			description:  "no goals should fail; newline stripped",
			requiredGoal: "xxx",
			mavenOutput:  "\n",
			shouldErr:    true,
		},
		{
			description:  "valid goal",
			requiredGoal: "dockerBuild",
			mavenOutput:  "dockerBuild",
			shouldErr:    false,
		},
		{
			description:  "newline stripped",
			requiredGoal: "dockerBuild",
			mavenOutput:  "dockerBuild\n",
			shouldErr:    false,
		},
		{
			description:  "invalid goal",
			requiredGoal: "dockerBuild",
			mavenOutput:  "build\n",
			shouldErr:    true,
		},
		{
			description:  "too many goals",
			requiredGoal: "dockerBuild",
			mavenOutput:  "build\ndockerBuild\n",
			shouldErr:    true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.SkipWrapperCheck, true)
			t.Override(&util.DefaultExecCommand, t.FakeRunOut(
				"mvn --quiet --projects module jib:_skaffold-package-goals",
				test.mavenOutput,
			))

			err := verifyJibPackageGoal(context.Background(), test.requiredGoal, ".", &latest.JibMavenArtifact{Module: "module"})

			t.CheckError(test.shouldErr, err)
		})
	}
}
