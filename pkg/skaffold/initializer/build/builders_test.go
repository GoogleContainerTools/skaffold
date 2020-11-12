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
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/buildpacks"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/jib"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/prompt"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/warnings"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestResolveBuilderImages(t *testing.T) {
	tests := []struct {
		description            string
		buildConfigs           []InitBuilder
		images                 []string
		force                  bool
		shouldMakeChoice       bool
		shouldErr              bool
		expectedInfos          []ArtifactInfo
		expectedGeneratedInfos []GeneratedArtifactInfo
	}{
		{
			description:      "nothing to choose from",
			buildConfigs:     []InitBuilder{},
			images:           []string{},
			shouldMakeChoice: false,
			expectedInfos:    nil,
		},
		{
			description:      "don't prompt for single dockerfile and image",
			buildConfigs:     []InitBuilder{docker.ArtifactConfig{File: "Dockerfile1"}},
			images:           []string{"image1"},
			shouldMakeChoice: false,
			expectedInfos: []ArtifactInfo{
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
			expectedInfos: []ArtifactInfo{
				{
					Builder:   docker.ArtifactConfig{File: "Dockerfile1"},
					ImageName: "image1",
				},
				{
					Builder:   jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibGradle), File: "build.gradle"},
					ImageName: "image2",
				},
			},
			expectedGeneratedInfos: []GeneratedArtifactInfo{
				{
					ArtifactInfo: ArtifactInfo{
						Builder:   jib.ArtifactConfig{BuilderName: "Jib Maven Plugin", File: "pom.xml", Project: "project"},
						ImageName: "pom.xml-image",
					},
					ManifestPath: "deployment.yaml",
				},
			},
		},
		{
			description:      "successful force",
			buildConfigs:     []InitBuilder{jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibGradle), File: "build.gradle"}},
			images:           []string{"image1"},
			shouldMakeChoice: false,
			force:            true,
			expectedInfos: []ArtifactInfo{
				{
					Builder:   jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibGradle), File: "build.gradle"},
					ImageName: "image1",
				},
			},
		},
		{
			description:      "successful force - 1 image 2 builders",
			buildConfigs:     []InitBuilder{docker.ArtifactConfig{File: "Dockerfile1"}, jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibGradle), File: "build.gradle"}},
			images:           []string{"image1"},
			shouldMakeChoice: true,
			force:            true,
			expectedInfos: []ArtifactInfo{
				{
					Builder:   docker.ArtifactConfig{File: "Dockerfile1"},
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
		{
			description:  "one unresolved image",
			buildConfigs: []InitBuilder{docker.ArtifactConfig{File: "foo"}},
			images:       []string{},
			expectedGeneratedInfos: []GeneratedArtifactInfo{
				{
					ArtifactInfo: ArtifactInfo{
						Builder:   docker.ArtifactConfig{File: "foo"},
						ImageName: "foo-image",
					},
					ManifestPath: "deployment.yaml",
				},
			},
			shouldMakeChoice: false,
			force:            false,
			shouldErr:        false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			// Overrides prompt.BuildConfigFunc to choose first option rather than using the interactive menu
			t.Override(&prompt.BuildConfigFunc, func(image string, choices []string) (string, error) {
				if !test.shouldMakeChoice {
					t.FailNow()
				}
				return choices[0], nil
			})

			initializer := &defaultBuildInitializer{
				builders:         test.buildConfigs,
				force:            test.force,
				unresolvedImages: test.images,
			}
			err := initializer.resolveBuilderImages()
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedInfos, initializer.artifactInfos, cmp.AllowUnexported())
			t.CheckDeepEqual(test.expectedGeneratedInfos, initializer.generatedArtifactInfos, cmp.AllowUnexported())
		})
	}
}

func TestAutoSelectBuilders(t *testing.T) {
	tests := []struct {
		description              string
		builderConfigs           []InitBuilder
		images                   []string
		expectedInfos            []ArtifactInfo
		expectedBuildersLeft     []InitBuilder
		expectedUnresolvedImages []string
	}{
		{
			description: "no automatic matches",
			builderConfigs: []InitBuilder{
				docker.ArtifactConfig{File: "Dockerfile"},
				jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibGradle), File: "build.gradle"},
				jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibMaven), File: "pom.xml", Image: "not a k8s image"},
			},
			images:        []string{"image1", "image2"},
			expectedInfos: nil,
			expectedBuildersLeft: []InitBuilder{
				docker.ArtifactConfig{File: "Dockerfile"},
				jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibGradle), File: "build.gradle"},
				jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibMaven), File: "pom.xml", Image: "not a k8s image"},
			},
			expectedUnresolvedImages: []string{"image1", "image2"},
		},
		{
			description: "automatic jib matches",
			builderConfigs: []InitBuilder{
				docker.ArtifactConfig{File: "Dockerfile"},
				jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibGradle), File: "build.gradle", Image: "image1"},
				jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibMaven), File: "pom.xml", Image: "image2"},
			},
			images: []string{"image1", "image2", "image3"},
			expectedInfos: []ArtifactInfo{
				{
					Builder:   jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibGradle), File: "build.gradle", Image: "image1"},
					ImageName: "image1",
				},
				{
					Builder:   jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibMaven), File: "pom.xml", Image: "image2"},
					ImageName: "image2",
				},
			},
			expectedBuildersLeft:     []InitBuilder{docker.ArtifactConfig{File: "Dockerfile"}},
			expectedUnresolvedImages: []string{"image3"},
		},
		{
			description: "multiple matches for one image",
			builderConfigs: []InitBuilder{
				jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibGradle), File: "build.gradle", Image: "image1"},
				jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibMaven), File: "pom.xml", Image: "image1"},
			},
			images:        []string{"image1", "image2"},
			expectedInfos: nil,
			expectedBuildersLeft: []InitBuilder{
				jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibGradle), File: "build.gradle", Image: "image1"},
				jib.ArtifactConfig{BuilderName: jib.PluginName(jib.JibMaven), File: "pom.xml", Image: "image1"},
			},
			expectedUnresolvedImages: []string{"image1", "image2"},
		},
		{
			description:              "show unique image names",
			builderConfigs:           nil,
			images:                   []string{"image1", "image1"},
			expectedInfos:            nil,
			expectedBuildersLeft:     nil,
			expectedUnresolvedImages: []string{"image1"},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			pairs, builderConfigs, unresolvedImages := matchBuildersToImages(test.builderConfigs, test.images)

			t.CheckDeepEqual(test.expectedInfos, pairs)
			t.CheckDeepEqual(test.expectedBuildersLeft, builderConfigs)
			t.CheckDeepEqual(test.expectedUnresolvedImages, unresolvedImages)
		})
	}
}

func TestProcessCliArtifacts(t *testing.T) {
	tests := []struct {
		description        string
		artifacts          []string
		shouldErr          bool
		expectedInfos      []ArtifactInfo
		expectedWorkspaces []string
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
			expectedInfos: []ArtifactInfo{
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
				`{"builder":"Docker","payload":{"path":"/path/to/Dockerfile"},"image":"image1", "context": "path/to/docker/workspace"}`,
				`{"builder":"Jib Gradle Plugin","payload":{"path":"/path/to/build.gradle"},"image":"image2", "context":"path/to/jib/workspace"}`,
				`{"builder":"Jib Maven Plugin","payload":{"path":"/path/to/pom.xml","project":"project-name","image":"testImage"},"image":"image3"}`,
				`{"builder":"Buildpacks","payload":{"path":"/path/to/package.json"},"image":"image4"}`,
			},
			expectedInfos: []ArtifactInfo{
				{
					Builder:   docker.ArtifactConfig{File: "/path/to/Dockerfile"},
					ImageName: "image1",
					Workspace: "path/to/docker/workspace",
				},
				{
					Builder:   jib.ArtifactConfig{BuilderName: "Jib Gradle Plugin", File: "/path/to/build.gradle"},
					ImageName: "image2",
					Workspace: "path/to/jib/workspace",
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

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedInfos, pairs)
		})
	}
}

func TestStripImageTags(t *testing.T) {
	tests := []struct {
		description      string
		taggedImages     []string
		expectedImages   []string
		expectedWarnings []string
	}{
		{
			description:      "empty",
			taggedImages:     nil,
			expectedImages:   nil,
			expectedWarnings: nil,
		},
		{
			description: "tags are removed",
			taggedImages: []string{
				"gcr.io/testproject/testimage:latest",
				"testdockerhublib/bla:v1.0",
				"registrywithport:5000/image:v2.3",
			},
			expectedImages: []string{
				"gcr.io/testproject/testimage",
				"testdockerhublib/bla",
				"registrywithport:5000/image",
			},
			expectedWarnings: nil,
		},
		{
			description: "invalid image names are skipped with warning",
			taggedImages: []string{
				"gcr.io/testproject/testimage:latest",
				"{{ REPOSITORY }}/{{IMAGE}}",
			},
			expectedImages: []string{
				"gcr.io/testproject/testimage",
			},
			expectedWarnings: []string{
				"Couldn't parse image [{{ REPOSITORY }}/{{IMAGE}}]: invalid reference format",
			},
		},
		{
			description: "images with digest are ignored",
			taggedImages: []string{
				"gcr.io/testregistry/testimage@sha256:16a019b0fa168b31fbecb3f909f55a5342e39f346cae919b7ff0b22f40029876",
			},
			expectedImages: nil,
			expectedWarnings: []string{
				"Ignoring image referenced by digest: [gcr.io/testregistry/testimage@sha256:16a019b0fa168b31fbecb3f909f55a5342e39f346cae919b7ff0b22f40029876]",
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			fakeWarner := &warnings.Collect{}
			t.Override(&warnings.Printf, fakeWarner.Warnf)

			images := tag.StripTags(test.taggedImages)

			t.CheckDeepEqual(test.expectedImages, images)
			t.CheckDeepEqual(test.expectedWarnings, fakeWarner.Warnings)
		})
	}
}
