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

package gcb

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"google.golang.org/api/cloudbuild/v1"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build/jib"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestMain(m *testing.M) {
	// these tests don't actually require a JVM
	jib.JVMFound = func(context.Context) bool { return true }
	os.Exit(m.Run())
}

func TestJibMavenBuildSpec(t *testing.T) {
	tests := []struct {
		description  string
		skipTests    bool
		baseImage    string
		platforms    platform.Matcher
		expectedArgs []string
	}{
		{
			description:  "skip tests",
			skipTests:    true,
			expectedArgs: []string{"-c", "mvn -Duser.home=$$HOME --batch-mode jib:_skaffold-fail-if-jib-out-of-date -Djib.requiredVersion=" + jib.MinimumJibMavenVersion + " --non-recursive -DskipTests=true prepare-package jib:build -Dimage=img"},
		},
		{
			description:  "do not skip tests",
			skipTests:    false,
			expectedArgs: []string{"-c", "mvn -Duser.home=$$HOME --batch-mode jib:_skaffold-fail-if-jib-out-of-date -Djib.requiredVersion=" + jib.MinimumJibMavenVersion + " --non-recursive prepare-package jib:build -Dimage=img"},
		},
		{
			description:  "custom base image",
			baseImage:    "busybox",
			skipTests:    false,
			expectedArgs: []string{"-c", "mvn -Duser.home=$$HOME --batch-mode jib:_skaffold-fail-if-jib-out-of-date -Djib.requiredVersion=" + jib.MinimumJibMavenVersion + " --non-recursive prepare-package jib:build -Djib.from.image=busybox -Dimage=img"},
		},
		{
			description:  "custom base image from required artifacts",
			baseImage:    "img2",
			skipTests:    false,
			expectedArgs: []string{"-c", "mvn -Duser.home=$$HOME --batch-mode jib:_skaffold-fail-if-jib-out-of-date -Djib.requiredVersion=" + jib.MinimumJibMavenVersion + " --non-recursive prepare-package jib:build -Djib.from.image=img2:tag -Dimage=img"},
		},
		{
			description:  "cross platform",
			platforms:    platform.Matcher{Platforms: []v1.Platform{{Architecture: "arm", OS: "freebsd"}}},
			expectedArgs: []string{"-c", "mvn -Duser.home=$$HOME --batch-mode jib:_skaffold-fail-if-jib-out-of-date -Djib.requiredVersion=" + jib.MinimumJibMavenVersionForCrossPlatform + " --non-recursive prepare-package jib:build -Djib.from.platforms=freebsd/arm -Dimage=img"},
		},
		{
			description:  "multi platform",
			platforms:    platform.Matcher{Platforms: []v1.Platform{{Architecture: "arm", OS: "freebsd"}, {Architecture: "arm64", OS: "linux"}}},
			expectedArgs: []string{"-c", "mvn -Duser.home=$$HOME --batch-mode jib:_skaffold-fail-if-jib-out-of-date -Djib.requiredVersion=" + jib.MinimumJibMavenVersionForCrossPlatform + " --non-recursive prepare-package jib:build -Djib.from.platforms=freebsd/arm,linux/arm64 -Dimage=img"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			artifact := &latest.Artifact{
				ArtifactType: latest.ArtifactType{
					JibArtifact: &latest.JibArtifact{Type: string(jib.JibMaven), BaseImage: test.baseImage},
				},
				Dependencies: []*latest.ArtifactDependency{
					{ImageName: "img2", Alias: "img2"},
					{ImageName: "img3", Alias: "img3"},
				},
			}

			store := mockArtifactStore{
				"img2": "img2:tag",
				"img3": "img3:tag",
			}
			builder := NewBuilder(&mockBuilderContext{artifactStore: store}, &latest.GoogleCloudBuild{
				MavenImage: "maven:3.6.0",
			})
			builder.skipTests = test.skipTests

			buildSpec, err := builder.buildSpec(context.Background(), artifact, "img", test.platforms, "bucket", "object")
			t.CheckNoError(err)

			expected := []*cloudbuild.BuildStep{{
				Entrypoint: "sh",
				Name:       "maven:3.6.0",
				Args:       test.expectedArgs,
			}}

			t.CheckDeepEqual(expected, buildSpec.Steps)
			t.CheckEmpty(buildSpec.Images)
		})
	}
}

func TestJibGradleBuildSpec(t *testing.T) {
	tests := []struct {
		description  string
		skipTests    bool
		platforms    platform.Matcher
		expectedArgs []string
	}{
		{
			description:  "skip tests",
			skipTests:    true,
			expectedArgs: []string{"-c", "gradle -Duser.home=$$HOME --console=plain _skaffoldFailIfJibOutOfDate -Djib.requiredVersion=" + jib.MinimumJibGradleVersion + " :jib -x test --image=img"},
		},
		{
			description:  "do not skip tests",
			skipTests:    false,
			expectedArgs: []string{"-c", "gradle -Duser.home=$$HOME --console=plain _skaffoldFailIfJibOutOfDate -Djib.requiredVersion=" + jib.MinimumJibGradleVersion + " :jib --image=img"},
		},
		{
			description:  "cross platform",
			platforms:    platform.Matcher{Platforms: []v1.Platform{{Architecture: "arm", OS: "freebsd"}}},
			expectedArgs: []string{"-c", "gradle -Duser.home=$$HOME --console=plain _skaffoldFailIfJibOutOfDate -Djib.requiredVersion=" + jib.MinimumJibGradleVersionForCrossPlatform + " :jib -Djib.from.platforms=freebsd/arm --image=img"},
		},
		{
			description:  "multi platform",
			platforms:    platform.Matcher{Platforms: []v1.Platform{{Architecture: "arm", OS: "freebsd"}, {Architecture: "arm64", OS: "linux"}}},
			expectedArgs: []string{"-c", "gradle -Duser.home=$$HOME --console=plain _skaffoldFailIfJibOutOfDate -Djib.requiredVersion=" + jib.MinimumJibGradleVersionForCrossPlatform + " :jib -Djib.from.platforms=freebsd/arm,linux/arm64 --image=img"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			artifact := &latest.Artifact{
				ArtifactType: latest.ArtifactType{
					JibArtifact: &latest.JibArtifact{Type: string(jib.JibGradle)},
				},
			}

			builder := NewBuilder(&mockBuilderContext{}, &latest.GoogleCloudBuild{
				GradleImage: "gradle:5.1.1",
			})
			builder.skipTests = test.skipTests

			buildSpec, err := builder.buildSpec(context.Background(), artifact, "img", test.platforms, "bucket", "object")
			t.CheckNoError(err)

			expected := []*cloudbuild.BuildStep{{
				Entrypoint: "sh",
				Name:       "gradle:5.1.1",
				Args:       test.expectedArgs,
			}}

			t.CheckDeepEqual(expected, buildSpec.Steps)
			t.CheckEmpty(buildSpec.Images)
		})
	}
}

func TestJibAddWorkspaceToDependencies(t *testing.T) {
	tests := []struct {
		description       string
		workspacePaths    []string
		dependencies      []string
		expectedWorkspace []string
	}{
		{
			description:       "basic test",
			workspacePaths:    []string{"a/b/file", "c/file", "file"},
			dependencies:      []string{"dependencyA", "dependencyB"},
			expectedWorkspace: []string{"", "/a", "/a/b", "/a/b/file", "/c", "/c/file", "/file"},
		},
		{
			description:       "ignore target with pom",
			workspacePaths:    []string{"pom.xml", "target/fileA", "target/fileB", "watchedFile"},
			dependencies:      []string{"dependencyA", "dependencyB"},
			expectedWorkspace: []string{"", "/pom.xml", "/watchedFile"},
		},
		{
			description:       "don't ignore target without pom",
			workspacePaths:    []string{"target/fileA", "target/fileB", "watchedFile"},
			dependencies:      []string{"dependencyA", "dependencyB"},
			expectedWorkspace: []string{"", "/target", "/target/fileA", "/target/fileB", "/watchedFile"},
		},
		{
			description:       "ignore build with build.gradle",
			workspacePaths:    []string{"build.gradle", "build/fileA", "build/fileB", "watchedFile"},
			dependencies:      []string{"dependencyA", "dependencyB"},
			expectedWorkspace: []string{"", "/build.gradle", "/watchedFile"},
		},
		{
			description:       "don't ignore build without build.gradle",
			workspacePaths:    []string{"build/fileA", "build/fileB", "watchedFile"},
			dependencies:      []string{"dependencyA", "dependencyB"},
			expectedWorkspace: []string{"", "/build", "/build/fileA", "/build/fileB", "/watchedFile"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir()
			for _, f := range test.workspacePaths {
				tmpDir.Write(filepath.FromSlash(f), "")
			}

			for i := range test.expectedWorkspace {
				test.expectedWorkspace[i] = tmpDir.Root() + filepath.FromSlash(test.expectedWorkspace[i])
			}
			expectedDependencies := append(test.dependencies, test.expectedWorkspace...)

			actualDepedencies, err := jibAddWorkspaceToDependencies(tmpDir.Root(), test.dependencies)

			t.CheckNoError(err)
			t.CheckDeepEqual(expectedDependencies, actualDepedencies)
		})
	}
}
