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

package build

import (
	"bytes"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/jib"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestPrintAnalyzeJSON(t *testing.T) {
	tests := []struct {
		description string
		pairs       []BuilderImagePair
		builders    []InitBuilder
		images      []string
		skipBuild   bool
		shouldErr   bool
		expected    string
	}{
		{
			description: "builders and images with pairs",
			pairs:       []BuilderImagePair{{jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibGradle), Image: "image1", File: "build.gradle", Project: "project"}, "image1"}},
			builders:    []InitBuilder{docker.ArtifactConfig{File: "Dockerfile"}},
			images:      []string{"image2"},
			expected:    `{"builders":[{"name":"Jib Gradle Plugin","payload":{"image":"image1","path":"build.gradle","project":"project"}},{"name":"Docker","payload":{"path":"Dockerfile"}}],"images":[{"name":"image1","foundMatch":true},{"name":"image2","foundMatch":false}]}` + "\n",
		},
		{
			description: "builders and images with no pairs",
			builders:    []InitBuilder{jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibGradle), File: "build.gradle", Project: "project"}, docker.ArtifactConfig{File: "Dockerfile"}},
			images:      []string{"image1", "image2"},
			expected:    `{"builders":[{"name":"Jib Gradle Plugin","payload":{"path":"build.gradle","project":"project"}},{"name":"Docker","payload":{"path":"Dockerfile"}}],"images":[{"name":"image1","foundMatch":false},{"name":"image2","foundMatch":false}]}` + "\n",
		},
		{
			description: "no dockerfile, skip build",
			images:      []string{"image1", "image2"},
			skipBuild:   true,
			expected:    `{"images":[{"name":"image1","foundMatch":false},{"name":"image2","foundMatch":false}]}` + "\n",
		},
		{
			description: "no dockerfile",
			images:      []string{"image1", "image2"},
			shouldErr:   true,
		},
		{
			description: "no dockerfiles or images",
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			var out bytes.Buffer

			err := PrintAnalyzeJSON(&out, test.skipBuild, test.pairs, test.builders, test.images)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, out.String())
		})
	}
}

func TestPrintAnalyzeJSONNoJib(t *testing.T) {
	tests := []struct {
		description string
		pairs       []BuilderImagePair
		builders    []InitBuilder
		images      []string
		skipBuild   bool
		shouldErr   bool
		expected    string
	}{
		{
			description: "builders and images (backwards compatibility)",
			builders:    []InitBuilder{docker.ArtifactConfig{File: "Dockerfile1"}, docker.ArtifactConfig{File: "Dockerfile2"}},
			images:      []string{"image1", "image2"},
			expected:    `{"dockerfiles":["Dockerfile1","Dockerfile2"],"images":["image1","image2"]}` + "\n",
		},
		{
			description: "no dockerfile, skip build (backwards compatibility)",
			images:      []string{"image1", "image2"},
			skipBuild:   true,
			expected:    `{"images":["image1","image2"]}` + "\n",
		},
		{
			description: "no dockerfile",
			images:      []string{"image1", "image2"},
			shouldErr:   true,
		},
		{
			description: "no dockerfiles or images",
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			var out bytes.Buffer

			err := PrintAnalyzeOldFormat(&out, test.skipBuild, test.pairs, test.builders, test.images)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, out.String())
		})
	}
}
