// +build integration

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

package integration

import (
	"os/exec"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

func TestBuild(t *testing.T) {
	tests := []struct {
		description string
		dir         string
		args        []string
	}{
		{
			description: "docker build",
			dir:         "testdata/build",
		}, {
			description: "git tagger",
			dir:         "testdata/tagPolicy",
			args:        []string{"-p", "gitCommit"},
		}, {
			description: "sha256 tagger",
			dir:         "testdata/tagPolicy",
			args:        []string{"-p", "sha256"},
		}, {
			description: "dateTime tagger",
			dir:         "testdata/tagPolicy",
			args:        []string{"-p", "dateTime"},
		}, {
			description: "envTemplate tagger",
			dir:         "testdata/tagPolicy",
			args:        []string{"-p", "envTemplate"},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			buildCmd := exec.Command("skaffold", append([]string{"build"}, test.args...)...)
			buildCmd.Dir = test.dir

			out, err := util.RunCmdOut(buildCmd)
			if err != nil {
				t.Fatalf("testing error: %v, %s", err, out)
			}
		})
	}
}
