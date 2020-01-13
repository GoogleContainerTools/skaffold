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
	"context"
	"strings"
	"testing"

	"github.com/pkg/errors"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/buildpacks"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/jib"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
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
			pairs:       []builderImagePair{{jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibGradle), Image: "image1", File: "build.gradle", Project: "project"}, "image1"}},
			builders:    []InitBuilder{docker.ArtifactConfig{File: "Dockerfile"}},
			images:      []string{"image2"},
			expected:    `{"builders":[{"name":"Jib Gradle Plugin","payload":{"image":"image1","path":"build.gradle","project":"project"}},{"name":"Docker","payload":{"path":"Dockerfile"}}],"images":[{"name":"image1","foundMatch":true},{"name":"image2","foundMatch":false}]}`,
		},
		{
			description: "builders and images with no pairs",
			builders:    []InitBuilder{jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibGradle), File: "build.gradle", Project: "project"}, docker.ArtifactConfig{File: "Dockerfile"}},
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
			builders:    []InitBuilder{docker.ArtifactConfig{File: "Dockerfile1"}, docker.ArtifactConfig{File: "Dockerfile2"}},
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
	validK8sManifest := "apiVersion: v1\nkind: Service\nmetadata:\n  name: test\n"

	tests := []struct {
		description         string
		filesWithContents   map[string]string
		expectedConfigs     []string
		expectedPaths       []string
		force               bool
		enableJibInit       bool
		enableBuildpackInit bool
		shouldErr           bool
	}{
		{
			description: "should return correct k8 configs and build files (backwards compatibility)",
			filesWithContents: map[string]string{
				"config/test.yaml":       validK8sManifest,
				"config/invalid.yaml":    emptyFile,
				"k8pod.yml":              validK8sManifest,
				"README":                 emptyFile,
				"deploy/Dockerfile":      emptyFile,
				"deploy/Dockerfile.dev":  emptyFile,
				"deploy/dev.Dockerfile":  emptyFile,
				"deploy/test.dockerfile": emptyFile,
				"gradle/build.gradle":    emptyFile,
				"maven/pom.xml":          emptyFile,
				"Dockerfile":             emptyFile,
			},
			force: false,
			expectedConfigs: []string{
				"k8pod.yml",
				"config/test.yaml",
			},
			expectedPaths: []string{
				"Dockerfile",
				"deploy/Dockerfile",
				"deploy/Dockerfile.dev",
				"deploy/dev.Dockerfile",
				"deploy/test.dockerfile",
			},
			shouldErr: false,
		},
		{
			description: "should return correct k8 configs and build files",
			filesWithContents: map[string]string{
				"config/test.yaml":    validK8sManifest,
				"config/invalid.yaml": emptyFile,
				"k8pod.yml":           validK8sManifest,
				"README":              emptyFile,
				"deploy/Dockerfile":   emptyFile,
				"gradle/build.gradle": emptyFile,
				"maven/pom.xml":       emptyFile,
				"Dockerfile":          emptyFile,
				"node/package.json":   emptyFile,
			},
			force:               false,
			enableJibInit:       true,
			enableBuildpackInit: true,
			expectedConfigs: []string{
				"k8pod.yml",
				"config/test.yaml",
			},
			expectedPaths: []string{
				"Dockerfile",
				"deploy/Dockerfile",
				"gradle/build.gradle",
				"maven/pom.xml",
				"node/package.json",
			},
			shouldErr: false,
		},
		{
			description: "skip validating nested jib configs",
			filesWithContents: map[string]string{
				"config/test.yaml":               validK8sManifest,
				"k8pod.yml":                      validK8sManifest,
				"gradle/build.gradle":            emptyFile,
				"gradle/subproject/build.gradle": emptyFile,
				"maven/asubproject/pom.xml":      emptyFile,
				"maven/pom.xml":                  emptyFile,
			},
			force:         false,
			enableJibInit: true,
			expectedConfigs: []string{
				"k8pod.yml",
				"config/test.yaml",
			},
			expectedPaths: []string{
				"gradle/build.gradle",
				"maven/pom.xml",
			},
			shouldErr: false,
		},
		{
			description: "multiple builders in same directory",
			filesWithContents: map[string]string{
				"build.gradle":                 emptyFile,
				"ignored-builder/build.gradle": emptyFile,
				"not-ignored-config/test.yaml": validK8sManifest,
				"Dockerfile":                   emptyFile,
				"k8pod.yml":                    validK8sManifest,
				"pom.xml":                      emptyFile,
			},
			force:         false,
			enableJibInit: true,
			expectedConfigs: []string{
				"k8pod.yml",
				"not-ignored-config/test.yaml",
			},
			expectedPaths: []string{
				"Dockerfile",
				"build.gradle",
				"pom.xml",
			},
			shouldErr: false,
		},
		{
			description: "should skip hidden dir",
			filesWithContents: map[string]string{
				".hidden/test.yaml":  validK8sManifest,
				"k8pod.yml":          validK8sManifest,
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
				"config/test.yaml":  validK8sManifest,
				"k8pod.yml":         validK8sManifest,
				"README":            emptyFile,
				"deploy/Dockerfile": emptyFile,
				"Dockerfile":        emptyFile,
			},
			force:         true,
			enableJibInit: true,
			expectedConfigs: []string{
				"k8pod.yml",
				"config/test.yaml",
			},
			expectedPaths: []string{
				"Dockerfile",
				"deploy/Dockerfile",
			},
			shouldErr: false,
		},
		{
			description: "should error when skaffold.config present and force = false",
			filesWithContents: map[string]string{
				"config/test.yaml":  validK8sManifest,
				"k8pod.yml":         validK8sManifest,
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
		{
			description: "should error when skaffold.config present with jib config",
			filesWithContents: map[string]string{
				"config/test.yaml": validK8sManifest,
				"k8pod.yml":        validK8sManifest,
				"README":           emptyFile,
				"pom.xml":          emptyFile,
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
			tmpDir := t.NewTempDir().WriteFiles(test.filesWithContents)

			t.Override(&docker.Validate, fakeValidateDockerfile)
			t.Override(&jib.Validate, fakeValidateJibConfig)

			potentialConfigs, builders, err := walk(tmpDir.Root(), test.force, test.enableJibInit, test.enableBuildpackInit)

			t.CheckError(test.shouldErr, err)
			if test.shouldErr {
				return
			}

			t.CheckDeepEqual(tmpDir.Paths(test.expectedConfigs...), potentialConfigs)
			t.CheckDeepEqual(len(test.expectedPaths), len(builders))
			for i := range builders {
				t.CheckDeepEqual(tmpDir.Path(test.expectedPaths[i]), builders[i].Path())
			}
		})
	}
}

func fakeValidateDockerfile(path string) bool {
	return strings.Contains(strings.ToLower(path), "dockerfile")
}

func fakeValidateJibConfig(path string) []jib.ArtifactConfig {
	if strings.HasSuffix(path, "build.gradle") {
		return []jib.ArtifactConfig{{BuilderName: jib.PluginName(jib.JibGradle), File: path}}
	}
	if strings.HasSuffix(path, "pom.xml") {
		return []jib.ArtifactConfig{{BuilderName: jib.PluginName(jib.JibMaven), File: path}}
	}
	return nil
}

func TestResolveBuilderImages(t *testing.T) {
	tests := []struct {
		description      string
		buildConfigs     []InitBuilder
		images           []string
		force            bool
		shouldMakeChoice bool
		shouldErr        bool
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
			buildConfigs:     []InitBuilder{docker.ArtifactConfig{File: "Dockerfile1"}},
			images:           []string{"image1"},
			shouldMakeChoice: false,
			expectedPairs: []builderImagePair{
				{
					Builder:   docker.ArtifactConfig{File: "Dockerfile1"},
					ImageName: "image1",
				},
			},
		},
		{
			description:      "prompt for multiple builders and images",
			buildConfigs:     []InitBuilder{docker.ArtifactConfig{File: "Dockerfile1"}, jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibGradle), File: "build.gradle"}, jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibMaven), Project: "project", File: "pom.xml"}},
			images:           []string{"image1", "image2"},
			shouldMakeChoice: true,
			expectedPairs: []builderImagePair{
				{
					Builder:   docker.ArtifactConfig{File: "Dockerfile1"},
					ImageName: "image1",
				},
				{
					Builder:   jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibGradle), File: "build.gradle"},
					ImageName: "image2",
				},
			},
		},
		{
			description:      "successful force",
			buildConfigs:     []InitBuilder{jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibGradle), File: "build.gradle"}},
			images:           []string{"image1"},
			shouldMakeChoice: false,
			force:            true,
			expectedPairs: []builderImagePair{
				{
					Builder:   jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibGradle), File: "build.gradle"},
					ImageName: "image1",
				},
			},
		},
		{
			description:      "error with ambiguous force",
			buildConfigs:     []InitBuilder{docker.ArtifactConfig{File: "Dockerfile1"}, jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibGradle), File: "build.gradle"}},
			images:           []string{"image1", "image2"},
			shouldMakeChoice: false,
			force:            true,
			shouldErr:        true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			// Overrides promptUserForBuildConfig to choose first option rather than using the interactive menu
			t.Override(&promptUserForBuildConfigFunc, func(image string, choices []string) (string, error) {
				if !test.shouldMakeChoice {
					t.FailNow()
				}
				return choices[0], nil
			})

			pairs, err := resolveBuilderImages(test.buildConfigs, test.images, test.force)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedPairs, pairs)
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
				docker.ArtifactConfig{File: "Dockerfile"},
				jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibGradle), File: "build.gradle"},
				jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibMaven), File: "pom.xml", Image: "not a k8s image"},
			},
			images:        []string{"image1", "image2"},
			expectedPairs: nil,
			expectedBuildersLeft: []InitBuilder{
				docker.ArtifactConfig{File: "Dockerfile"},
				jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibGradle), File: "build.gradle"},
				jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibMaven), File: "pom.xml", Image: "not a k8s image"},
			},
			expectedFilteredImages: []string{"image1", "image2"},
		},
		{
			description: "automatic jib matches",
			builderConfigs: []InitBuilder{
				docker.ArtifactConfig{File: "Dockerfile"},
				jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibGradle), File: "build.gradle", Image: "image1"},
				jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibMaven), File: "pom.xml", Image: "image2"},
			},
			images: []string{"image1", "image2", "image3"},
			expectedPairs: []builderImagePair{
				{
					jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibGradle), File: "build.gradle", Image: "image1"},
					"image1",
				},
				{
					jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibMaven), File: "pom.xml", Image: "image2"},
					"image2",
				},
			},
			expectedBuildersLeft:   []InitBuilder{docker.ArtifactConfig{File: "Dockerfile"}},
			expectedFilteredImages: []string{"image3"},
		},
		{
			description: "multiple matches for one image",
			builderConfigs: []InitBuilder{
				jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibGradle), File: "build.gradle", Image: "image1"},
				jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibMaven), File: "pom.xml", Image: "image1"},
			},
			images:        []string{"image1", "image2"},
			expectedPairs: nil,
			expectedBuildersLeft: []InitBuilder{
				jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibGradle), File: "build.gradle", Image: "image1"},
				jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibMaven), File: "pom.xml", Image: "image1"},
			},
			expectedFilteredImages: []string{"image1", "image2"},
		},
		{
			description:            "show unique image names",
			builderConfigs:         nil,
			images:                 []string{"image1", "image1"},
			expectedPairs:          nil,
			expectedBuildersLeft:   nil,
			expectedFilteredImages: []string{"image1"},
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
					Builder:   docker.ArtifactConfig{File: "/path/to/Dockerfile"},
					ImageName: "image1",
				},
				{
					Builder:   docker.ArtifactConfig{File: "/path/to/Dockerfile2"},
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
				`{"builder":"Buildpacks","payload":{"path":"/path/to/package.json"},"image":"image4"}`,
			},
			expectedPairs: []builderImagePair{
				{
					Builder:   docker.ArtifactConfig{File: "/path/to/Dockerfile"},
					ImageName: "image1",
				},
				{
					Builder:   jib.ArtifactConfig{BuilderName: "Jib Gradle Plugin", File: "/path/to/build.gradle"},
					ImageName: "image2",
				},
				{
					Builder:   jib.ArtifactConfig{BuilderName: "Jib Maven Plugin", File: "/path/to/pom.xml", Project: "project-name", Image: "testImage"},
					ImageName: "image3",
				},
				{
					Builder:   buildpacks.ArtifactConfig{File: "/path/to/package.json"},
					ImageName: "image4",
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

func Test_canonicalizeName(t *testing.T) {
	const length253 = "aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaaaaaaaa-aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaaaaaaaa-aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaa"
	tests := []struct {
		in, out string
	}{
		{
			in:  "abc def",
			out: "abc-def",
		},
		{
			in:  "abc    def",
			out: "abc-def",
		},
		{
			in:  "abc...def",
			out: "abc...def",
		},
		{
			in:  "abc---def",
			out: "abc---def",
		},
		{
			in:  "aBc DeF",
			out: "abc-def",
		},
		{
			in:  length253 + "XXXXXXX",
			out: length253,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.in, func(t *testutil.T) {
			actual := canonicalizeName(test.in)

			t.CheckDeepEqual(test.out, actual)
		})
	}
}

func TestRunKompose(t *testing.T) {
	tests := []struct {
		description   string
		composeFile   string
		commands      util.Command
		expectedError string
	}{
		{
			description: "success",
			composeFile: "docker-compose.yaml",
			commands:    testutil.CmdRunOut("kompose convert -f docker-compose.yaml", ""),
		},
		{
			description:   "not found",
			composeFile:   "not-found.yaml",
			expectedError: "(no such file or directory|cannot find the file specified)",
		},
		{
			description:   "failure",
			composeFile:   "docker-compose.yaml",
			commands:      testutil.CmdRunOutErr("kompose convert -f docker-compose.yaml", "", errors.New("BUG")),
			expectedError: "BUG",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.NewTempDir().Touch("docker-compose.yaml").Chdir()
			t.Override(&util.DefaultExecCommand, test.commands)

			err := runKompose(context.Background(), test.composeFile)

			if test.expectedError != "" {
				t.CheckMatches(test.expectedError, err.Error())
			}
		})
	}
}

func TestArtifacts(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		artifacts := artifacts([]builderImagePair{
			{
				ImageName: "image1",
				Builder: docker.ArtifactConfig{
					File: "Dockerfile",
				},
			},
			{
				ImageName: "image2",
				Builder: docker.ArtifactConfig{
					File: "front/Dockerfile2",
				},
			},
			{
				ImageName: "image3",
				Builder: buildpacks.ArtifactConfig{
					File: "package.json",
				},
			},
		})

		expected := []*latest.Artifact{
			{
				ImageName:    "image1",
				ArtifactType: latest.ArtifactType{},
			},
			{
				ImageName: "image2",
				Workspace: "front",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						DockerfilePath: "Dockerfile2",
					},
				},
			},
			{
				ImageName: "image3",
				ArtifactType: latest.ArtifactType{
					BuildpackArtifact: &latest.BuildpackArtifact{
						Builder: "heroku/buildpacks",
					},
				},
			},
		}

		t.CheckDeepEqual(expected, artifacts)
	})
}
