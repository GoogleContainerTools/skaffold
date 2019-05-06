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

package cmd

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestRunDeploy(t *testing.T) {
	errRunner := func(context.Context, io.Writer) ([]build.Result, error) {
		return nil, errors.New("some error")
	}
	mockCreateRunner := func(context.Context, io.Writer) ([]build.Result, error) {
		return []build.Result{{
			Target: latest.Artifact{
				ImageName: "gcr.io/skaffold/example",
			},
			Result: build.Artifact{
				ImageName: "gcr.io/skaffold/example",
				Tag:       "gcr.io/skaffold/example:test",
			},
		}}, nil
	}
	defer func(f func(context.Context, io.Writer) ([]build.Result, error)) {
		createRunnerAndBuildFunc = f
	}(createRunnerAndBuildFunc)

	var tests = []struct {
		description string
		mock        func(context.Context, io.Writer) ([]build.Result, error)
		shouldErr   bool
	}{
		{
			description: "build should return successfully when runner is successful.",
			shouldErr:   false,
			mock:        mockCreateRunner,
		},
		{
			description: "build errors out when there was runner error.",
			shouldErr:   true,
			mock:        errRunner,
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			createRunnerAndBuildFunc = test.mock
			err := runBuild(ioutil.Discard)
			testutil.CheckError(t, test.shouldErr, err)
		})
	}
}
