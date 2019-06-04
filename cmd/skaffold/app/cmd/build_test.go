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
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/flags"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestQuietFlag(t *testing.T) {
	mockCreateRunner := func(context.Context, io.Writer) ([]build.Artifact, error) {
		return []build.Artifact{{
			ImageName: "gcr.io/skaffold/example",
			Tag:       "test",
		}}, nil
	}

	var tests = []struct {
		description    string
		template       string
		expectedOutput []byte
		mock           func(context.Context, io.Writer) ([]build.Artifact, error)
		shouldErr      bool
	}{
		{
			description:    "quiet flag print build images with no template",
			expectedOutput: []byte(`{"builds":[{"imageName":"gcr.io/skaffold/example","tag":"test"}]}`),
			shouldErr:      false,
			mock:           mockCreateRunner,
		},
		{
			description:    "quiet flag print build images applies pattern specified in template ",
			template:       "{{range .Builds}}{{.ImageName}} -> {{.Tag}}\n{{end}}",
			expectedOutput: []byte("gcr.io/skaffold/example -> test\n"),
			shouldErr:      false,
			mock:           mockCreateRunner,
		},
		{
			description:    "build errors out when incorrect template specified",
			template:       "{{.Incorrect}}",
			expectedOutput: nil,
			shouldErr:      true,
			mock:           mockCreateRunner,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&quietFlag, true)
			t.Override(&createRunnerAndBuildFunc, test.mock)
			if test.template != "" {
				t.Override(&buildFormatFlag, flags.NewTemplateFlag(test.template, flags.BuildOutput{}))
			}

			var output bytes.Buffer

			err := doBuild(context.Background(), &output)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, string(test.expectedOutput), output.String())
		})
	}
}

func TestRunBuild(t *testing.T) {
	errRunner := func(context.Context, io.Writer) ([]build.Artifact, error) {
		return nil, errors.New("some error")
	}
	mockCreateRunner := func(context.Context, io.Writer) ([]build.Artifact, error) {
		return []build.Artifact{{
			ImageName: "gcr.io/skaffold/example",
			Tag:       "test",
		}}, nil
	}

	var tests = []struct {
		description string
		mock        func(context.Context, io.Writer) ([]build.Artifact, error)
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
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&createRunnerAndBuildFunc, test.mock)

			err := doBuild(context.Background(), ioutil.Discard)

			t.CheckError(test.shouldErr, err)
		})
	}
}
