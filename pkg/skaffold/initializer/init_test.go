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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/jib"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestPrintAnalyzeJSON(t *testing.T) {
	tests := []struct {
		description string
		builders    []InitBuilder
		images      []string
		skipBuild   bool
		shouldErr   bool
		expected    string
	}{
		{
			description: "builders and images",
			builders: []InitBuilder{
				docker.Dockerfile("Dockerfile"),
				jib.Config{Name: jib.JibGradle, Image: "image1", Path: "build.gradle", Project: "project"},
				jib.Config{Name: jib.JibMaven, Image: "image2", Path: "pom.xml"},
			},
			images:   []string{"image1", "image2"},
			expected: "{\"builders\":[{\"path\":\"Dockerfile\"},{\"path\":\"build.gradle\",\"configuredImage\":\"image1\"},{\"path\":\"pom.xml\",\"configuredImage\":\"image2\"}],\"images\":[\"image1\",\"image2\"]}",
		},
		{
			description: "no builders, skip build",
			images:      []string{"image1", "image2"},
			skipBuild:   true,
			expected:    "{\"images\":[\"image1\",\"image2\"]}"},
		{
			description: "no builders",
			images:      []string{"image1", "image2"},
			shouldErr:   true,
		},
		{
			description: "no builders or images",
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			out := bytes.NewBuffer([]byte{})

			err := printAnalyzeJSON(out, test.skipBuild, test.builders, test.images)

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
				"config/test.yaml":    emptyFile,
				"k8pod.yml":           emptyFile,
				"README":              emptyFile,
				"deploy/Dockerfile":   emptyFile,
				"gradle/build.gradle": emptyFile,
				"maven/pom.xml":       emptyFile,
				"Dockerfile":          emptyFile,
			},
			force: false,
			expectedConfigs: []string{
				"config/test.yaml",
				"k8pod.yml",
			},
			expectedPaths: []string{
				"Dockerfile",
				"deploy/Dockerfile",
				"gradle/build.gradle",
				"maven/pom.xml",
			},
			shouldErr: false,
		},
		{
			description: "skip validating nested jib configs",
			filesWithContents: map[string]string{
				"config/test.yaml":               emptyFile,
				"k8pod.yml":                      emptyFile,
				"gradle/build.gradle":            emptyFile,
				"gradle/subproject/build.gradle": emptyFile,
				"maven/pom.xml":                  emptyFile,
				"maven/subproject/pom.xml":       emptyFile,
			},
			force: false,
			expectedConfigs: []string{
				"config/test.yaml",
				"k8pod.yml",
			},
			expectedPaths: []string{
				"gradle/build.gradle",
				"maven/pom.xml",
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
			tmpDir := t.NewTempDir()
			for file, contents := range test.filesWithContents {
				tmpDir.Write(file, contents)
			}

			t.Override(&docker.ValidateDockerfile, fakeValidateDockerfile)
			t.Override(&jib.ValidateJibConfig, fakeValidateJibConfig)

			potentialConfigs, builders, err := walk(tmpDir.Root(), test.force, detectBuilders)

			t.CheckError(test.shouldErr, err)
			t.CheckDeepEqual(tmpDir.Paths(test.expectedConfigs...), potentialConfigs)
			t.CheckDeepEqual(len(test.expectedPaths), len(builders))
			for i := range builders {
				t.CheckDeepEqual(tmpDir.Path(test.expectedPaths[i]), builders[i].GetPath())
			}
		})
	}
}

func fakeValidateDockerfile(path string) bool {
	return strings.HasSuffix(path, "Dockerfile")
}

func fakeValidateJibConfig(path string) []jib.Config {
	if strings.HasSuffix(path, "build.gradle") {
		return []jib.Config{{Name: jib.JibGradle, Path: path}}
	} else if strings.HasSuffix(path, "pom.xml") {
		return []jib.Config{{Name: jib.JibMaven, Path: path}}
	}
	return nil
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
			buildConfigs:     []InitBuilder{docker.Dockerfile("Dockerfile1")},
			images:           []string{"image1"},
			shouldMakeChoice: false,
			expectedPairs: []builderImagePair{
				{
					Builder:   docker.Dockerfile("Dockerfile1"),
					ImageName: "image1",
				},
			},
		},
		{
			description:      "prompt for multiple builders and images",
			buildConfigs:     []InitBuilder{docker.Dockerfile("Dockerfile1"), jib.Config{Name: jib.JibGradle, Path: "build.gradle"}, jib.Config{Name: jib.JibMaven, Project: "project", Path: "pom.xml"}},
			images:           []string{"image1", "image2"},
			shouldMakeChoice: true,
			expectedPairs: []builderImagePair{
				{
					Builder:   docker.Dockerfile("Dockerfile1"),
					ImageName: "image1",
				},
				{
					Builder:   jib.Config{Name: jib.JibGradle, Path: "build.gradle"},
					ImageName: "image2",
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			// Overrides promptUserForBuildConfig to choose first option rather than using the interactive menu
			t.Override(&promptUserForBuildConfig, func(image string, choices []string) string {
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

func TestAutoSelectBuilders(t *testing.T) {
	tests := []struct {
		description            string
		builderConfigs         []InitBuilder
		images                 []string
		expectedPairs          []builderImagePair
		expectedBuildersLeft   []InitBuilder
		expectedFilteredImages []string
	}{
		{
			description: "no automatic matches",
			builderConfigs: []InitBuilder{
				docker.Dockerfile("Dockerfile"),
				jib.Config{Name: jib.JibGradle, Path: "build.gradle"},
				jib.Config{Name: jib.JibMaven, Path: "pom.xml", Image: "not a k8s image"},
			},
			images:        []string{"image1", "image2"},
			expectedPairs: []builderImagePair{},
			expectedBuildersLeft: []InitBuilder{
				docker.Dockerfile("Dockerfile"),
				jib.Config{Name: jib.JibGradle, Path: "build.gradle"},
				jib.Config{Name: jib.JibMaven, Path: "pom.xml", Image: "not a k8s image"},
			},
			expectedFilteredImages: []string{"image1", "image2"},
		},
		{
			description: "automatic jib matches",
			builderConfigs: []InitBuilder{
				docker.Dockerfile("Dockerfile"),
				jib.Config{Name: jib.JibGradle, Path: "build.gradle", Image: "image1"},
				jib.Config{Name: jib.JibMaven, Path: "pom.xml", Image: "image2"},
			},
			images: []string{"image1", "image2", "image3"},
			expectedPairs: []builderImagePair{
				{
					jib.Config{Name: jib.JibGradle, Path: "build.gradle", Image: "image1"},
					"image1",
				},
				{
					jib.Config{Name: jib.JibMaven, Path: "pom.xml", Image: "image2"},
					"image2",
				},
			},
			expectedBuildersLeft:   []InitBuilder{docker.Dockerfile("Dockerfile")},
			expectedFilteredImages: []string{"image3"},
		},
		{
			description: "multiple matches for one image",
			builderConfigs: []InitBuilder{
				jib.Config{Name: jib.JibGradle, Path: "build.gradle", Image: "image1"},
				jib.Config{Name: jib.JibMaven, Path: "pom.xml", Image: "image1"},
			},
			images:        []string{"image1", "image2"},
			expectedPairs: []builderImagePair{},
			expectedBuildersLeft: []InitBuilder{
				jib.Config{Name: jib.JibGradle, Path: "build.gradle", Image: "image1"},
				jib.Config{Name: jib.JibMaven, Path: "pom.xml", Image: "image1"},
			},
			expectedFilteredImages: []string{"image1", "image2"},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {

			pairs, builderConfigs, filteredImages := autoSelectBuilders(test.builderConfigs, test.images)

			t.CheckDeepEqual(test.expectedPairs, pairs)
			t.CheckDeepEqual(test.expectedBuildersLeft, builderConfigs)
			t.CheckDeepEqual(test.expectedFilteredImages, filteredImages)
		})
	}
}
