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
		pairs       []builderImagePair
		builders    []InitBuilder
		images      []string
		skipBuild   bool
		shouldErr   bool
		expected    string
	}{
		{
			description: "builders and images with pairs",
			pairs:       []builderImagePair{{jib.Jib{BuilderName: jib.JibGradle.Name(), Image: "image1", FilePath: "build.gradle", Project: "project"}, "image1"}},
			builders:    []InitBuilder{docker.Docker{File: "Dockerfile"}},
			images:      []string{"image2"},
			expected:    `{"builders":[{"name":"Jib Gradle Plugin","payload":{"image":"image1","path":"build.gradle","project":"project"}},{"name":"Docker","payload":{"path":"Dockerfile"}}],"images":[{"name":"image1","foundMatch":true},{"name":"image2","foundMatch":false}]}`,
		},
		{
			description: "builders and images with no pairs",
			builders:    []InitBuilder{jib.Jib{BuilderName: jib.JibGradle.Name(), FilePath: "build.gradle", Project: "project"}, docker.Docker{File: "Dockerfile"}},
			images:      []string{"image1", "image2"},
			expected:    `{"builders":[{"name":"Jib Gradle Plugin","payload":{"path":"build.gradle","project":"project"}},{"name":"Docker","payload":{"path":"Dockerfile"}}],"images":[{"name":"image1","foundMatch":false},{"name":"image2","foundMatch":false}]}`,
		},
		{
			description: "no dockerfile, skip build",
			images:      []string{"image1", "image2"},
			skipBuild:   true,
			expected:    `{"images":[{"name":"image1","foundMatch":false},{"name":"image2","foundMatch":false}]}`,
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

			err := printAnalyzeJSON(&out, test.skipBuild, test.pairs, test.builders, test.images)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, out.String())
		})
	}
}

func TestPrintAnalyzeJSONNoJib(t *testing.T) {
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
			description: "builders and images (backwards compatibility)",
			builders:    []InitBuilder{docker.Docker{File: "Dockerfile1"}, docker.Docker{File: "Dockerfile2"}},
			images:      []string{"image1", "image2"},
			expected:    `{"dockerfiles":["Dockerfile1","Dockerfile2"],"images":["image1","image2"]}`,
		},
		{
			description: "no dockerfile, skip build (backwards compatibility)",
			images:      []string{"image1", "image2"},
			skipBuild:   true,
			expected:    `{"images":["image1","image2"]}`,
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

			err := printAnalyzeJSONNoJib(&out, test.skipBuild, test.pairs, test.builders, test.images)

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
		enableJibInit     bool
		shouldErr         bool
	}{
		{
			description: "should return correct k8 configs and build files (backwards compatibility)",
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
			},
			shouldErr: false,
		},
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
			force:         false,
			enableJibInit: true,
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
			force:         false,
			enableJibInit: true,
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
			force:         false,
			enableJibInit: true,
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
			force:         true,
			enableJibInit: true,
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
			enableJibInit:   true,
			expectedConfigs: nil,
			expectedPaths:   nil,
			shouldErr:       true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir().
				WriteFiles(test.filesWithContents)

			t.Override(&docker.ValidateDockerfileFunc, fakeValidateDockerfile)
			t.Override(&jib.ValidateJibConfigFunc, fakeValidateJibConfig)

			potentialConfigs, builders, err := walk(tmpDir.Root(), test.force, test.enableJibInit, detectBuilders)

			t.CheckError(test.shouldErr, err)
			t.CheckDeepEqual(tmpDir.Paths(test.expectedConfigs...), potentialConfigs)
			t.CheckDeepEqual(len(test.expectedPaths), len(builders))
			for i := range builders {
				t.CheckDeepEqual(tmpDir.Path(test.expectedPaths[i]), builders[i].Path())
			}
		})
	}
}

func fakeValidateDockerfile(path string) bool {
	return strings.HasSuffix(path, "Dockerfile")
}

func fakeValidateJibConfig(path string) []jib.Jib {
	if strings.HasSuffix(path, "build.gradle") {
		return []jib.Jib{{BuilderName: jib.JibGradle.Name(), FilePath: path}}
	}
	if strings.HasSuffix(path, "pom.xml") {
		return []jib.Jib{{BuilderName: jib.JibMaven.Name(), FilePath: path}}
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
			buildConfigs:     []InitBuilder{docker.Docker{File: "Dockerfile1"}},
			images:           []string{"image1"},
			shouldMakeChoice: false,
			expectedPairs: []builderImagePair{
				{
					Builder:   docker.Docker{File: "Dockerfile1"},
					ImageName: "image1",
				},
			},
		},
		{
			description:      "prompt for multiple builders and images",
			buildConfigs:     []InitBuilder{docker.Docker{File: "Dockerfile1"}, jib.Jib{BuilderName: jib.JibGradle.Name(), FilePath: "build.gradle"}, jib.Jib{BuilderName: jib.JibMaven.Name(), Project: "project", FilePath: "pom.xml"}},
			images:           []string{"image1", "image2"},
			shouldMakeChoice: true,
			expectedPairs: []builderImagePair{
				{
					Builder:   docker.Docker{File: "Dockerfile1"},
					ImageName: "image1",
				},
				{
					Builder:   jib.Jib{BuilderName: jib.JibGradle.Name(), FilePath: "build.gradle"},
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
				docker.Docker{File: "Dockerfile"},
				jib.Jib{BuilderName: jib.JibGradle.Name(), FilePath: "build.gradle"},
				jib.Jib{BuilderName: jib.JibMaven.Name(), FilePath: "pom.xml", Image: "not a k8s image"},
			},
			images:        []string{"image1", "image2"},
			expectedPairs: nil,
			expectedBuildersLeft: []InitBuilder{
				docker.Docker{File: "Dockerfile"},
				jib.Jib{BuilderName: jib.JibGradle.Name(), FilePath: "build.gradle"},
				jib.Jib{BuilderName: jib.JibMaven.Name(), FilePath: "pom.xml", Image: "not a k8s image"},
			},
			expectedFilteredImages: []string{"image1", "image2"},
		},
		{
			description: "automatic jib matches",
			builderConfigs: []InitBuilder{
				docker.Docker{File: "Dockerfile"},
				jib.Jib{BuilderName: jib.JibGradle.Name(), FilePath: "build.gradle", Image: "image1"},
				jib.Jib{BuilderName: jib.JibMaven.Name(), FilePath: "pom.xml", Image: "image2"},
			},
			images: []string{"image1", "image2", "image3"},
			expectedPairs: []builderImagePair{
				{
					jib.Jib{BuilderName: jib.JibGradle.Name(), FilePath: "build.gradle", Image: "image1"},
					"image1",
				},
				{
					jib.Jib{BuilderName: jib.JibMaven.Name(), FilePath: "pom.xml", Image: "image2"},
					"image2",
				},
			},
			expectedBuildersLeft:   []InitBuilder{docker.Docker{File: "Dockerfile"}},
			expectedFilteredImages: []string{"image3"},
		},
		{
			description: "multiple matches for one image",
			builderConfigs: []InitBuilder{
				jib.Jib{BuilderName: jib.JibGradle.Name(), FilePath: "build.gradle", Image: "image1"},
				jib.Jib{BuilderName: jib.JibMaven.Name(), FilePath: "pom.xml", Image: "image1"},
			},
			images:        []string{"image1", "image2"},
			expectedPairs: nil,
			expectedBuildersLeft: []InitBuilder{
				jib.Jib{BuilderName: jib.JibGradle.Name(), FilePath: "build.gradle", Image: "image1"},
				jib.Jib{BuilderName: jib.JibMaven.Name(), FilePath: "pom.xml", Image: "image1"},
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

func TestProcessCliArtifacts(t *testing.T) {
	tests := []struct {
		description   string
		artifacts     []string
		shouldErr     bool
		expectedPairs []builderImagePair
	}{
		{
			description: "Invalid pairs",
			artifacts:   []string{"invalid"},
			shouldErr:   true,
		},
		{
			description: "Invalid builder",
			artifacts:   []string{`{"builder":"Not real","payload":{},"image":"image"}`},
			shouldErr:   true,
		},
		{
			description: "Valid (backwards compatibility)",
			artifacts: []string{
				`/path/to/Dockerfile=image1`,
				`/path/to/Dockerfile2=image2`,
			},
			expectedPairs: []builderImagePair{
				{
					Builder:   docker.Docker{File: "/path/to/Dockerfile"},
					ImageName: "image1",
				},
				{
					Builder:   docker.Docker{File: "/path/to/Dockerfile2"},
					ImageName: "image2",
				},
			},
		},
		{
			description: "Valid",
			artifacts: []string{
				`{"builder":"Docker","payload":{"path":"/path/to/Dockerfile"},"image":"image1"}`,
				`{"builder":"Jib Gradle Plugin","payload":{"path":"/path/to/build.gradle"},"image":"image2"}`,
				`{"builder":"Jib Maven Plugin","payload":{"path":"/path/to/pom.xml","project":"project-name","image":"testImage"},"image":"image3"}`,
			},
			expectedPairs: []builderImagePair{
				{
					Builder:   docker.Docker{File: "/path/to/Dockerfile"},
					ImageName: "image1",
				},
				{
					Builder:   jib.Jib{BuilderName: "Jib Gradle Plugin", FilePath: "/path/to/build.gradle"},
					ImageName: "image2",
				},
				{
					Builder:   jib.Jib{BuilderName: "Jib Maven Plugin", FilePath: "/path/to/pom.xml", Project: "project-name", Image: "testImage"},
					ImageName: "image3",
				},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			pairs, err := processCliArtifacts(test.artifacts)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedPairs, pairs)
		})
	}
}
