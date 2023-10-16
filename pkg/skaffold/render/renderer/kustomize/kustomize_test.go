/*
Copyright 2022 The Skaffold Authors

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

package kustomize

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestBuildCommandArgs(t *testing.T) {
	tests := []struct {
		description   string
		buildArgs     []string
		kustomizePath string
		expectedArgs  []string
	}{
		{
			description:   "no BuildArgs, empty KustomizePaths ",
			buildArgs:     []string{},
			kustomizePath: "",
			expectedArgs:  nil,
		},
		{
			description:   "One BuildArg, empty KustomizePaths",
			buildArgs:     []string{"--foo"},
			kustomizePath: "",
			expectedArgs:  []string{"--foo"},
		},
		{
			description:   "no BuildArgs, non-empty KustomizePaths",
			buildArgs:     []string{},
			kustomizePath: "foo",
			expectedArgs:  []string{"foo"},
		},
		{
			description:   "One BuildArg, non-empty KustomizePaths",
			buildArgs:     []string{"--foo"},
			kustomizePath: "bar",
			expectedArgs:  []string{"--foo", "bar"},
		},
		{
			description:   "Multiple BuildArg, empty KustomizePaths",
			buildArgs:     []string{"--foo", "--bar"},
			kustomizePath: "",
			expectedArgs:  []string{"--foo", "--bar"},
		},
		{
			description:   "Multiple BuildArg with spaces, empty KustomizePaths",
			buildArgs:     []string{"--foo bar", "--baz"},
			kustomizePath: "",
			expectedArgs:  []string{"--foo", "bar", "--baz"},
		},
		{
			description:   "Multiple BuildArg with spaces, non-empty KustomizePaths",
			buildArgs:     []string{"--foo bar", "--baz"},
			kustomizePath: "barfoo",
			expectedArgs:  []string{"--foo", "bar", "--baz", "barfoo"},
		},
		{
			description:   "Multiple BuildArg no spaces, non-empty KustomizePaths",
			buildArgs:     []string{"--foo", "bar", "--baz"},
			kustomizePath: "barfoo",
			expectedArgs:  []string{"--foo", "bar", "--baz", "barfoo"},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			args := kustomizeBuildArgs(test.buildArgs, test.kustomizePath)
			t.CheckDeepEqual(test.expectedArgs, args)
		})
	}
}
