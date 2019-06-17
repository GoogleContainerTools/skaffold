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

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestPrintAnalyzeJSON(t *testing.T) {
	tests := []struct {
		description string
		dockerfiles []string
		images      []string
		skipBuild   bool
		shouldErr   bool
		expected    string
	}{
		{
			description: "dockerfile and image",
			dockerfiles: []string{"Dockerfile", "Dockerfile_2"},
			images:      []string{"image1", "image2"},
			expected:    "{\"dockerfiles\":[\"Dockerfile\",\"Dockerfile_2\"],\"images\":[\"image1\",\"image2\"]}",
		},
		{
			description: "no dockerfile, skip build",
			images:      []string{"image1", "image2"},
			skipBuild:   true,
			expected:    "{\"images\":[\"image1\",\"image2\"]}"},
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

			err := printAnalyzeJSON(out, test.skipBuild, test.dockerfiles, test.images)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, out.String())
		})
	}
}

func TestWalk(t *testing.T) {
	emptyFile := ""
	tests := []struct {
		description         string
		filesWithContents   map[string]string
		expectedConfigs     []string
		expectedDockerfiles []string
		force               bool
		shouldErr           bool
	}{
		{
			description: "should return correct k8 configs and dockerfiles",
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
			expectedDockerfiles: []string{
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
			expectedDockerfiles: []string{
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
			expectedDockerfiles: []string{
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
			force:               false,
			expectedConfigs:     nil,
			expectedDockerfiles: nil,
			shouldErr:           true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir()
			for file, contents := range test.filesWithContents {
				tmpDir.Write(file, contents)
			}

			potentialConfigs, dockerfiles, err := walk(tmpDir.Root(), test.force, testValidDocker)

			t.CheckError(test.shouldErr, err)
			t.CheckDeepEqual(tmpDir.Paths(test.expectedConfigs...), potentialConfigs)
			t.CheckDeepEqual(tmpDir.Paths(test.expectedDockerfiles...), dockerfiles)
		})
	}
}

func testValidDocker(path string) bool {
	return strings.HasSuffix(path, "Dockerfile")
}
