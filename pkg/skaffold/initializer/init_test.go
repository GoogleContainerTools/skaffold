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
	"os"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestPrintAnalyzeJSON(t *testing.T) {
	tests := []struct {
		name        string
		dockerfiles []string
		images      []string
		skipBuild   bool
		shouldErr   bool
		expected    string
	}{
		{
			name:        "dockerfile and image",
			dockerfiles: []string{"Dockerfile", "Dockerfile_2"},
			images:      []string{"image1", "image2"},
			expected:    "{\"dockerfiles\":[\"Dockerfile\",\"Dockerfile_2\"],\"images\":[\"image1\",\"image2\"]}",
		},
		{
			name:      "no dockerfile, skip build",
			images:    []string{"image1", "image2"},
			skipBuild: true,
			expected:  "{\"images\":[\"image1\",\"image2\"]}"},
		{
			name:      "no dockerfile",
			images:    []string{"image1", "image2"},
			shouldErr: true,
		},
		{
			name:      "no dockerfiles or images",
			shouldErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			out := bytes.NewBuffer([]byte{})
			err := printAnalyzeJSON(out, test.skipBuild, test.dockerfiles, test.images)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, out.String())
		})
	}
}

func TestWalk(t *testing.T) {
	emptyFile := ""
	tests := []struct {
		name                string
		filesWithContents   map[string]string
		expectedConfigs     []string
		expectedDockerfiles []string
		force               bool
		shouldErr           bool
	}{
		{
			name: "should return correct k8 configs and dockerfiles",
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
			name: "should skip hidden dir",
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
			name: "should not error when skaffold.config present and force = true",
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
			name: "should  error when skaffold.config present and force = false",
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
		t.Run(test.name, func(t *testing.T) {
			testDir, deleteFunc := testutil.NewTempDir(t)
			rootDir := testDir.Root()
			deleteFunc()
			writeAllFiles(testDir, test.filesWithContents)
			potentialConfigs, dockerfiles, err := walk(rootDir, test.force, testValidDocker)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err,
				testDir.Paths(test.expectedConfigs), potentialConfigs)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err,
				testDir.Paths(test.expectedDockerfiles), dockerfiles)
			os.Remove(rootDir)
		})
	}
}

func testValidDocker(path string) bool {
	return strings.HasSuffix(path, "Dockerfile")
}

func writeAllFiles(tmpDir *testutil.TempDir, filesWithContents map[string]string) {
	for file, contents := range filesWithContents {
		tmpDir.Write(file, contents)
	}
}
