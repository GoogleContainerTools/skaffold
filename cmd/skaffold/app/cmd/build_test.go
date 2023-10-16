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
	"os"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/cmd/skaffold/app/flags"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

type mockRunner struct {
	runner.Runner
}

func (r *mockRunner) Build(ctx context.Context, out io.Writer, artifacts []*latest.Artifact) ([]graph.Artifact, error) {
	out.Write([]byte("Build Completed"))
	graphArtifacts := make([]graph.Artifact, len(artifacts))
	for i, a := range artifacts {
		graphArtifacts[i] = graph.Artifact{
			ImageName:   a.ImageName,
			Tag:         "test",
			RuntimeType: a.RuntimeType,
		}
	}
	return graphArtifacts, nil
}

func (r *mockRunner) Stop() error {
	return nil
}

func newMockCreateRunner(artifacts []*latest.Artifact) func(context.Context, io.Writer, config.SkaffoldOptions) (runner.Runner, []util.VersionedConfig, *runcontext.RunContext, error) {
	return func(context.Context, io.Writer, config.SkaffoldOptions) (runner.Runner, []util.VersionedConfig, *runcontext.RunContext, error) {
		return &mockRunner{}, []util.VersionedConfig{&latest.SkaffoldConfig{
			Pipeline: latest.Pipeline{
				Build: latest.BuildConfig{
					Artifacts: artifacts,
				},
			},
		}}, nil, nil
	}
}

func TestTagFlag(t *testing.T) {
	mockCreateRunner := newMockCreateRunner([]*latest.Artifact{{ImageName: "gcr.io/skaffold/example"}})

	testutil.Run(t, "override tag with argument", func(t *testutil.T) {
		t.Override(&quietFlag, true)
		t.Override(&opts.CustomTag, "tag")
		t.Override(&createRunner, mockCreateRunner)

		var output bytes.Buffer

		err := doBuild(context.Background(), &output)

		t.CheckNoError(err)
		t.CheckDeepEqual(string([]byte(`{"builds":[{"imageName":"gcr.io/skaffold/example","tag":"test"}]}`)), output.String())
	})
}

func TestQuietFlag(t *testing.T) {
	mockCreateRunner := newMockCreateRunner([]*latest.Artifact{{ImageName: "gcr.io/skaffold/example"}})

	tests := []struct {
		description    string
		template       string
		expectedOutput []byte
		shouldErr      bool
	}{
		{
			description:    "quiet flag print build images with no template",
			expectedOutput: []byte(`{"builds":[{"imageName":"gcr.io/skaffold/example","tag":"test"}]}`),
			shouldErr:      false,
		},
		{
			description:    "quiet flag print build images applies pattern specified in template ",
			template:       "{{range .Builds}}{{.ImageName}} -> {{.Tag}}\n{{end}}",
			expectedOutput: []byte("gcr.io/skaffold/example -> test\n"),
			shouldErr:      false,
		},
		{
			description:    "build errors out when incorrect template specified",
			template:       "{{.Incorrect}}",
			expectedOutput: nil,
			shouldErr:      true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&quietFlag, true)
			t.Override(&createRunner, mockCreateRunner)
			if test.template != "" {
				t.Override(&buildFormatFlag, flags.NewTemplateFlag(test.template, flags.BuildOutput{}))
			}

			var output bytes.Buffer

			err := doBuild(context.Background(), &output)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, string(test.expectedOutput), output.String())
		})
	}
}

func TestFileOutputFlag(t *testing.T) {
	mockCreateRunner := newMockCreateRunner([]*latest.Artifact{{ImageName: "gcr.io/skaffold/example"}})

	tests := []struct {
		description         string
		filename            string
		quietFlag           bool
		template            string
		expectedOutput      []byte
		expectedFileContent []byte
	}{
		{
			description:         "build runs successfully with flag and creates a file",
			filename:            "testfile.out",
			quietFlag:           false,
			expectedOutput:      []byte("Build Completed"),
			expectedFileContent: []byte(`{"builds":[{"imageName":"gcr.io/skaffold/example","tag":"test"}]}`),
		},
		{
			description:         "file output flag with quiet flag creates a file and suppresses build output",
			filename:            "testfile.out",
			quietFlag:           true,
			expectedOutput:      []byte(`{"builds":[{"imageName":"gcr.io/skaffold/example","tag":"test"}]}`),
			expectedFileContent: []byte(`{"builds":[{"imageName":"gcr.io/skaffold/example","tag":"test"}]}`),
		},
		{
			description:         "file output flag with template properly formats output and writes to a file",
			filename:            "testfile.out",
			quietFlag:           true,
			template:            "{{range .Builds}}{{.ImageName}} -> {{.Tag}}\n{{end}}",
			expectedOutput:      []byte("gcr.io/skaffold/example -> test\n"),
			expectedFileContent: []byte("gcr.io/skaffold/example -> test\n"),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&quietFlag, test.quietFlag)
			t.Override(&buildOutputFlag, test.filename)
			t.Override(&createRunner, mockCreateRunner)
			if test.template != "" {
				t.Override(&buildFormatFlag, flags.NewTemplateFlag(test.template, flags.BuildOutput{}))
			}

			// tempDir for writing file to
			tempDir := t.NewTempDir()
			tempDir.Chdir()

			// Check that stdout is correct
			var output bytes.Buffer
			err := doBuild(context.Background(), &output)
			t.CheckNoError(err)
			t.CheckDeepEqual(string(test.expectedOutput), output.String())

			// Check that file contents are correct
			fileContent, err := os.ReadFile(test.filename)
			t.CheckNoError(err)
			t.CheckDeepEqual(string(test.expectedFileContent), string(fileContent))
		})
	}
}

func TestRunBuild(t *testing.T) {
	errRunner := func(context.Context, io.Writer, config.SkaffoldOptions) (runner.Runner, []util.VersionedConfig, *runcontext.RunContext, error) {
		return nil, nil, nil, errors.New("some error")
	}
	mockCreateRunner := newMockCreateRunner([]*latest.Artifact{{ImageName: "gcr.io/skaffold/example"}})

	tests := []struct {
		description string
		mock        func(context.Context, io.Writer, config.SkaffoldOptions) (runner.Runner, []util.VersionedConfig, *runcontext.RunContext, error)
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
			t.Override(&createRunner, test.mock)

			err := doBuild(context.Background(), io.Discard)

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestRuntimeType(t *testing.T) {
	mockCreateRunner := newMockCreateRunner([]*latest.Artifact{{
		ImageName:   "gcr.io/skaffold/example",
		RuntimeType: "go",
	}})

	testutil.Run(t, "set runtime type on artifact", func(t *testutil.T) {
		t.Override(&quietFlag, true)
		t.Override(&createRunner, mockCreateRunner)

		var output bytes.Buffer

		err := doBuild(context.Background(), &output)

		t.CheckNoError(err)
		t.CheckDeepEqual(string([]byte(`{"builds":[{"imageName":"gcr.io/skaffold/example","tag":"test","runtimeType":"go"}]}`)), output.String())
	})
}
