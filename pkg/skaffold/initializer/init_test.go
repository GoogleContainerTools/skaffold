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

package initializer

import (
	"bytes"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestPrintAnalyzeJSON(t *testing.T) {
	tests := []struct {
		description string
		pairs       []builderImagePair
		builders    []InitBuilder
		images      []string
		skipBuild   bool
		shouldErr   bool
		expected    string
	}{
		{
			description: "builders and images with pairs",
			pairs:       []builderImagePair{{docker.Docker{Dockerfile: "Dockerfile1"}, "image1"}},
			builders:    []InitBuilder{docker.Docker{Dockerfile: "Dockerfile2"}},
			images:      []string{"image2"},
			expected:    "{\"builders\":[{\"name\":\"Docker\",\"payload\":{\"path\":\"Dockerfile1\"}},{\"name\":\"Docker\",\"payload\":{\"path\":\"Dockerfile2\"}}],\"images\":[{\"name\":\"image1\",\"requiresPrompt\":false},{\"name\":\"image2\",\"requiresPrompt\":true}]}",
		},
		{
			description: "builders and images with no pairs",
			builders:    []InitBuilder{docker.Docker{Dockerfile: "Dockerfile1"}, docker.Docker{Dockerfile: "Dockerfile2"}},
			images:      []string{"image1", "image2"},
			expected:    "{\"builders\":[{\"name\":\"Docker\",\"payload\":{\"path\":\"Dockerfile1\"}},{\"name\":\"Docker\",\"payload\":{\"path\":\"Dockerfile2\"}}],\"images\":[{\"name\":\"image1\",\"requiresPrompt\":true},{\"name\":\"image2\",\"requiresPrompt\":true}]}",
		},
		{
			description: "no dockerfile, skip build",
			images:      []string{"image1", "image2"},
			skipBuild:   true,
			expected:    "{\"images\":[{\"name\":\"image1\",\"requiresPrompt\":true},{\"name\":\"image2\",\"requiresPrompt\":true}]}"},
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
			out := bytes.NewBuffer([]byte{})

			err := printAnalyzeJSON(out, test.skipBuild, test.pairs, test.builders, test.images)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, out.String())
		})
	}
}

func TestWalk(t *testing.T) {
	emptyFile := ""
	tests := []struct {
		description       string
		filesWithContents map[string]string
		expectedConfigs   []string
		expectedPaths     []string
		force             bool
		shouldErr         bool
	}{
		{
			description: "should return correct k8 configs and build files",
			filesWithContents: map[string]string{
				"config/test.yaml":  emptyFile,
				"k8pod.yml":         emptyFile,
				"README":            emptyFile,
				"deploy/Dockerfile": emptyFile,
				"Dockerfile":        emptyFile,
			},
			force: false,
			expectedConfigs: []string{
				"config/test.yaml",
				"k8pod.yml",
			},
			expectedPaths: []string{
				"Dockerfile",
				"deploy/Dockerfile",
			},
			shouldErr: false,
		},
		{
			description: "should skip hidden dir",
			filesWithContents: map[string]string{
				".hidden/test.yaml":  emptyFile,
				"k8pod.yml":          emptyFile,
				"README":             emptyFile,
				".hidden/Dockerfile": emptyFile,
				"Dockerfile":         emptyFile,
			},
			force: false,
			expectedConfigs: []string{
				"k8pod.yml",
			},
			expectedPaths: []string{
				"Dockerfile",
			},
			shouldErr: false,
		},
		{
			description: "should not error when skaffold.config present and force = true",
			filesWithContents: map[string]string{
				"skaffold.yaml": `apiVersion: skaffold/v1beta6
kind: Config
deploy:
  kustomize: {}`,
				"config/test.yaml":  emptyFile,
				"k8pod.yml":         emptyFile,
				"README":            emptyFile,
				"deploy/Dockerfile": emptyFile,
				"Dockerfile":        emptyFile,
			},
			force: true,
			expectedConfigs: []string{
				"config/test.yaml",
				"k8pod.yml",
			},
			expectedPaths: []string{
				"Dockerfile",
				"deploy/Dockerfile",
			},
			shouldErr: false,
		},
		{
			description: "should  error when skaffold.config present and force = false",
			filesWithContents: map[string]string{
				"config/test.yaml":  emptyFile,
				"k8pod.yml":         emptyFile,
				"README":            emptyFile,
				"deploy/Dockerfile": emptyFile,
				"Dockerfile":        emptyFile,
				"skaffold.yaml": `apiVersion: skaffold/v1beta6
kind: Config
deploy:
  kustomize: {}`,
			},
			force:           false,
			expectedConfigs: nil,
			expectedPaths:   nil,
			shouldErr:       true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir().
				WriteFiles(test.filesWithContents)

			t.Override(&docker.ValidateDockerfileFunc, testValidDocker)

			potentialConfigs, builders, err := walk(tmpDir.Root(), test.force, detectBuilders)

			t.CheckError(test.shouldErr, err)
			t.CheckDeepEqual(tmpDir.Paths(test.expectedConfigs...), potentialConfigs)
			t.CheckDeepEqual(len(test.expectedPaths), len(builders))
			for i := range builders {
				t.CheckDeepEqual(tmpDir.Path(test.expectedPaths[i]), builders[i].Path())
			}
		})
	}
}

func testValidDocker(path string) bool {
	return strings.HasSuffix(path, "Dockerfile")
}

func TestResolveBuilderImages(t *testing.T) {
	tests := []struct {
		description      string
		buildConfigs     []InitBuilder
		images           []string
		shouldMakeChoice bool
		expectedPairs    []builderImagePair
	}{
		{
			description:      "nothing to choose from",
			buildConfigs:     []InitBuilder{},
			images:           []string{},
			shouldMakeChoice: false,
			expectedPairs:    []builderImagePair{},
		},
		{
			description:      "don't prompt for single dockerfile and image",
			buildConfigs:     []InitBuilder{docker.Docker{Dockerfile: "Dockerfile1"}},
			images:           []string{"image1"},
			shouldMakeChoice: false,
			expectedPairs: []builderImagePair{
				{
					Builder:   docker.Docker{Dockerfile: "Dockerfile1"},
					ImageName: "image1",
				},
			},
		},
		{
			description:      "prompt for multiple builders and images",
			buildConfigs:     []InitBuilder{docker.Docker{Dockerfile: "Dockerfile1"}, docker.Docker{Dockerfile: "Dockerfile2"}},
			images:           []string{"image1", "image2"},
			shouldMakeChoice: true,
			expectedPairs: []builderImagePair{
				{
					Builder:   docker.Docker{Dockerfile: "Dockerfile1"},
					ImageName: "image1",
				},
				{
					Builder:   docker.Docker{Dockerfile: "Dockerfile2"},
					ImageName: "image2",
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			// Overrides promptUserForBuildConfig to choose first option rather than using the interactive menu
			t.Override(&promptUserForBuildConfigFunc, func(image string, choices []string) string {
				if !test.shouldMakeChoice {
					t.FailNow()
				}
				return choices[0]
			})

			pairs := resolveBuilderImages(test.buildConfigs, test.images)

			t.CheckDeepEqual(test.expectedPairs, pairs)
		})
	}
}
