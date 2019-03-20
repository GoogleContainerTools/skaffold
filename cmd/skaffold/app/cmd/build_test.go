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
	"errors"
	"io"
	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/flags"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestQuietFlag(t *testing.T) {
	mockCreateRunner := func(buildOut io.Writer) ([]build.Artifact, func(), error) {
		return []build.Artifact{{
			ImageName: "gcr.io/skaffold/example",
			Tag:       "test",
		}}, func() {}, nil
	}

	orginalCreateRunner := createRunnerAndBuildFunc
	defer func(c func(buildOut io.Writer) ([]build.Artifact, func(), error)) {
		createRunnerAndBuildFunc = c
	}(orginalCreateRunner)
	var tests = []struct {
		name           string
		template       string
		expectedOutput []byte
		mock           func(io.Writer) ([]build.Artifact, func(), error)
		shdErr         bool
	}{
		{
			name:           "quiet flag print build images with no template",
			expectedOutput: []byte("{[{gcr.io/skaffold/example test}]}"),
			shdErr:         false,
			mock:           mockCreateRunner,
		},
		{
			name:           "quiet flag print build images applies pattern specified in template ",
			template:       "{{range .Builds}}{{.ImageName}} -> {{.Tag}}\n{{end}}",
			expectedOutput: []byte("gcr.io/skaffold/example -> test\n"),
			shdErr:         false,
			mock:           mockCreateRunner,
		},
		{
			name:           "build errors out when incorrect template specified",
			template:       "{{.Incorrect}}",
			expectedOutput: nil,
			shdErr:         true,
			mock:           mockCreateRunner,
		},
	}

	for _, test := range tests {
		quietFlag = true
		defer func() { quietFlag = false }()
		if test.template != "" {
			buildFormatFlag = flags.NewTemplateFlag(test.template, BuildOutput{})
		}
		defer func() { buildFormatFlag = nil }()
		createRunnerAndBuildFunc = test.mock
		var output bytes.Buffer
		err := runBuild(&output)
		testutil.CheckErrorAndDeepEqual(t, test.shdErr, err, string(test.expectedOutput), output.String())
	}
}

func TestRunBuild(t *testing.T) {
	mockCreateRunner := func(buildOut io.Writer) ([]build.Artifact, func(), error) {
		return []build.Artifact{{
			ImageName: "gcr.io/skaffold/example",
			Tag:       "test",
		}}, func() {}, nil
	}
	errRunner := func(buildOut io.Writer) ([]build.Artifact, func(), error) {
		return nil, func() {}, errors.New("some error")
	}

	orginalCreateRunner := createRunnerAndBuildFunc
	defer func(c func(buildOut io.Writer) ([]build.Artifact, func(), error)) {
		createRunnerAndBuildFunc = c
	}(orginalCreateRunner)

	var tests = []struct {
		name   string
		mock   func(io.Writer) ([]build.Artifact, func(), error)
		shdErr bool
	}{
		{
			name:   "buod should return successfully when runner is successful.",
			shdErr: false,
			mock:   mockCreateRunner,
		},
		{
			name:   "build errors out when there was runner error.",
			shdErr: true,
			mock:   errRunner,
		},
	}
	for _, test := range tests {
		createRunnerAndBuildFunc = test.mock
		err := runBuild(ioutil.Discard)
		testutil.CheckError(t, test.shdErr, err)
	}

}
