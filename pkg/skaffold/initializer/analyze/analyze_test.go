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

package analyze

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	initconfig "github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/config"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/jib"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

type builder struct {
	name string
	path string
}

func TestAnalyze(t *testing.T) {
	emptyFile := ""
	largeFile := ""
	for i := 1; i < 1000; i++ {
		largeFile = fmt.Sprintf("%s0", largeFile)
	}
	validK8sManifest := "apiVersion: v1\nkind: Service\nmetadata:\n  name: test\n"

	tests := []struct {
		description       string
		filesWithContents map[string]string
		expectedConfigs   []string
		expectedBuilders  []builder
		config            initconfig.Config
		shouldErr         bool
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
			config: initconfig.Config{
				Force:                false,
				EnableBuildpacksInit: false,
				EnableJibInit:        false,
			},
			expectedConfigs: []string{
				"k8pod.yml",
				"config/test.yaml",
			},
			expectedBuilders: []builder{
				{name: "Docker", path: "Dockerfile"},
				{name: "Docker", path: "deploy/Dockerfile"},
				{name: "Docker", path: "deploy/Dockerfile.dev"},
				{name: "Docker", path: "deploy/dev.Dockerfile"},
				{name: "Docker", path: "deploy/test.dockerfile"},
			},
			shouldErr: false,
		},
		{
			description: "--skip-build should return no builders in analysis",
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
			config: initconfig.Config{
				Force:                false,
				EnableBuildpacksInit: false,
				EnableJibInit:        false,
				SkipBuild:            true,
			},
			expectedConfigs: []string{
				"k8pod.yml",
				"config/test.yaml",
			},
			expectedBuilders: nil,
			shouldErr:        false,
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
			config: initconfig.Config{
				Force:                false,
				EnableBuildpacksInit: true,
				EnableJibInit:        true,
				EnableJibGradleInit:  true,
			},
			expectedConfigs: []string{
				"k8pod.yml",
				"config/test.yaml",
			},
			expectedBuilders: []builder{
				{name: "Docker", path: "Dockerfile"},
				{name: "Docker", path: "deploy/Dockerfile"},
				{name: "Jib Gradle Plugin", path: "gradle/build.gradle"},
				{name: "Buildpacks", path: "gradle/build.gradle"},
				{name: "Jib Maven Plugin", path: "maven/pom.xml"},
				{name: "Buildpacks", path: "maven/pom.xml"},
				{name: "Buildpacks", path: "node/package.json"},
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
				"gradle/subproject/Dockerfile":   emptyFile,
				"maven/asubproject/pom.xml":      emptyFile,
				"maven/asubproject/Dockerfile":   emptyFile,
				"maven/pom.xml":                  emptyFile,
			},
			config: initconfig.Config{
				Force:                false,
				EnableBuildpacksInit: false,
				EnableJibInit:        true,
				EnableJibGradleInit:  true,
			},
			expectedConfigs: []string{
				"k8pod.yml",
				"config/test.yaml",
			},
			expectedBuilders: []builder{
				{name: "Jib Gradle Plugin", path: "gradle/build.gradle"},
				{name: "Docker", path: "gradle/subproject/Dockerfile"},
				{name: "Jib Maven Plugin", path: "maven/pom.xml"},
				{name: "Docker", path: "maven/asubproject/Dockerfile"},
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
			config: initconfig.Config{
				Force:                false,
				EnableBuildpacksInit: false,
				EnableJibInit:        true,
				EnableJibGradleInit:  true,
			},
			expectedConfigs: []string{
				"k8pod.yml",
				"not-ignored-config/test.yaml",
			},
			expectedBuilders: []builder{
				{name: "Docker", path: "Dockerfile"},
				{name: "Jib Gradle Plugin", path: "build.gradle"},
				{name: "Jib Maven Plugin", path: "pom.xml"},
			},
			shouldErr: false,
		},
		{
			description: "should skip jib gradle",
			filesWithContents: map[string]string{
				"build.gradle": emptyFile,
				"pom.xml":      emptyFile,
			},
			config: initconfig.Config{
				Force:                false,
				EnableBuildpacksInit: false,
				EnableJibInit:        true,
				EnableJibGradleInit:  false,
			},
			expectedConfigs: nil,
			expectedBuilders: []builder{
				{name: "Jib Maven Plugin", path: "pom.xml"},
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
			config: initconfig.Config{
				Force:                false,
				EnableBuildpacksInit: false,
				EnableJibInit:        true,
			},
			expectedConfigs: []string{
				"k8pod.yml",
			},
			expectedBuilders: []builder{
				{name: "Docker", path: "Dockerfile"},
			},
			shouldErr: false,
		},
		{
			description: "should skip vendor folder",
			filesWithContents: map[string]string{
				"Dockerfile":            emptyFile,
				"vendor/Dockerfile":     emptyFile,
				"vendor/pom.xml":        emptyFile,
				"vendor/package.json":   emptyFile,
				"sub/vendor/Dockerfile": emptyFile,
			},
			config: initconfig.Config{
				Force:                false,
				EnableBuildpacksInit: true,
				EnableJibInit:        true,
			},
			expectedBuilders: []builder{
				{name: "Docker", path: "Dockerfile"},
			},
			shouldErr: false,
		},
		{
			description: "should skip node_modules folder",
			filesWithContents: map[string]string{
				"Dockerfile":                  emptyFile,
				"node_modules/Dockerfile":     emptyFile,
				"node_modules/pom.xml":        emptyFile,
				"node_modules/package.json":   emptyFile,
				"sub/node_modules/Dockerfile": emptyFile,
			},
			config: initconfig.Config{
				Force:                false,
				EnableBuildpacksInit: true,
				EnableJibInit:        true,
			},
			expectedBuilders: []builder{
				{name: "Docker", path: "Dockerfile"},
			},
			shouldErr: false,
		},
		{
			description: "should skip large files",
			filesWithContents: map[string]string{
				"k8pod.yml":               validK8sManifest,
				"README":                  emptyFile,
				"Dockerfile":              emptyFile,
				"largeFileDir/Dockerfile": largeFile,
			},
			config: initconfig.Config{
				Force:                false,
				EnableBuildpacksInit: false,
				EnableJibInit:        true,
				MaxFileSize:          100,
			},
			expectedConfigs: []string{
				"k8pod.yml",
			},
			expectedBuilders: []builder{
				{name: "Docker", path: "Dockerfile"},
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
			config: initconfig.Config{
				Force:                true,
				EnableBuildpacksInit: false,
				EnableJibInit:        true,
			},
			expectedConfigs: []string{
				"k8pod.yml",
				"config/test.yaml",
			},
			expectedBuilders: []builder{
				{name: "Docker", path: "Dockerfile"},
				{name: "Docker", path: "deploy/Dockerfile"},
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
			config: initconfig.Config{
				Force:                false,
				EnableBuildpacksInit: false,
				EnableJibInit:        true,
				Opts: config.SkaffoldOptions{
					ConfigurationFile: "skaffold.yaml",
				},
			},
			expectedConfigs:  nil,
			expectedBuilders: nil,
			shouldErr:        true,
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
			config: initconfig.Config{
				Force:                false,
				EnableBuildpacksInit: false,
				EnableJibInit:        true,
				Opts: config.SkaffoldOptions{
					ConfigurationFile: "skaffold.yaml",
				},
			},
			expectedConfigs:  nil,
			expectedBuilders: nil,
			shouldErr:        true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.NewTempDir().WriteFiles(test.filesWithContents).Chdir()

			t.Override(&docker.Validate, fakeValidateDockerfile)
			t.Override(&jib.Validate, fakeValidateJibConfig)

			a := NewAnalyzer(test.config)
			err := a.Analyze(".")

			t.CheckError(test.shouldErr, err)
			if test.shouldErr {
				return
			}

			t.CheckDeepEqual(test.expectedConfigs, a.Manifests())

			if len(test.expectedBuilders) != len(a.Builders()) {
				t.Fatalf("expected %d builders, got %d: %v",
					len(test.expectedBuilders),
					len(a.Builders()), a.Builders())
			}
			for i := range a.Builders() {
				t.CheckDeepEqual(test.expectedBuilders[i].name, a.Builders()[i].Name())
				t.CheckDeepEqual(test.expectedBuilders[i].path, a.Builders()[i].Path())
			}
		})
	}
}

func fakeValidateDockerfile(path string) bool {
	return strings.Contains(strings.ToLower(path), "dockerfile")
}

func fakeValidateJibConfig(path string, enableGradle bool) []jib.ArtifactConfig {
	if strings.HasSuffix(path, "build.gradle") && enableGradle {
		return []jib.ArtifactConfig{{BuilderName: jib.PluginName(jib.JibGradle), File: path}}
	}
	if strings.HasSuffix(path, "pom.xml") {
		return []jib.ArtifactConfig{{BuilderName: jib.PluginName(jib.JibMaven), File: path}}
	}
	return nil
}
