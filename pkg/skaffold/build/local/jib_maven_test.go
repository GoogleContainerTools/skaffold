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
	var testCases = []struct {
		requiredGoal string
		mavenOutput  string
		shouldError  bool
	}{
		{"xxx", "", true},   // no goals should fail
		{"xxx", "\n", true}, // no goals should fail; newline stripped
		{"dockerBuild", "dockerBuild", false},
		{"dockerBuild", "dockerBuild\n", false}, // newline stripped
		{"dockerBuild", "build\n", true},
		{"dockerBuild", "build\ndockerBuild\n", true},
	}

	reset := testutil.Override(t, &util.SkipWrapperCheck, true)
	defer reset()

	for _, tt := range testCases {
		reset := testutil.Override(t, &util.DefaultExecCommand, testutil.NewFakeCmd(t).WithRunOut(
			"mvn --quiet --projects module jib:_skaffold-package-goals",
			tt.mavenOutput,
		))
		defer reset()

		err := verifyJibPackageGoal(context.Background(), tt.requiredGoal, ".", &latest.JibMavenArtifact{Module: "module"})

		if hasError := err != nil; tt.shouldError != hasError {
			t.Error("Unexpected return result")
		}
	}
}
