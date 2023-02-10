/*
Copyright 2020 The Skaffold Authors

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
	"io"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestFilterIsHidden(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.NewTempDir().Chdir()
		t.Override(&opts, config.SkaffoldOptions{})

		cmd := NewCmdFilter()

		t.CheckDeepEqual(true, cmd.Hidden)
	})
}

func TestFilterTransform(t *testing.T) {
	tests := []struct {
		description    string
		manifestsStr   string
		buildArtifacts []graph.Artifact
		labels         []string
		expected       string
	}{
		{
			description: "manifests with images",
			manifestsStr: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: chartName
  labels:
    app: chartName
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: chartName
        image: image1`,
			buildArtifacts: []graph.Artifact{
				{ImageName: "image1", Tag: "image1:tag1"},
				{ImageName: "image2", Tag: "image2:tag2"}},
			labels: []string{"label1=foo", "run.id=random"},
			expected: `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: chartName
    label1: foo
    run.id: random
  name: chartName
spec:
  replicas: 3
  template:
    spec:
      containers:
      - image: image1:tag1
        name: chartName`,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&opts, config.SkaffoldOptions{
				CustomLabels: test.labels,
			})
			mockRunner := &mockDevRunner{}
			t.Override(&createRunner, func(context.Context, io.Writer, config.SkaffoldOptions) (runner.Runner, []util.VersionedConfig, *runcontext.RunContext, error) {
				return mockRunner, []util.VersionedConfig{&latest.SkaffoldConfig{}}, nil, nil
			})
			t.SetStdin([]byte(test.manifestsStr))
			var b bytes.Buffer
			err := runFilter(context.TODO(), &b, false, test.buildArtifacts)
			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, b.String(), testutil.YamlObj(t.T))
		})
	}
}
