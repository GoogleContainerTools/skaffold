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
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/jib"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
	cloudbuild "google.golang.org/api/cloudbuild/v1"
)

func TestJibMavenBuildSpec(t *testing.T) {
	tests := []struct {
		description  string
		skipTests    bool
		expectedArgs []string
	}{
		{
			description:  "skip tests",
			skipTests:    true,
			expectedArgs: []string{"-c", "mvn -Duser.home=$$HOME -Djib.console=plain jib:_skaffold-fail-if-jib-out-of-date -Djib.requiredVersion=" + jib.MinimumJibMavenVersion + " --non-recursive -DskipTests=true prepare-package jib:build -Dimage=img"},
		},
		{
			description:  "do not skip tests",
			skipTests:    false,
			expectedArgs: []string{"-c", "mvn -Duser.home=$$HOME -Djib.console=plain jib:_skaffold-fail-if-jib-out-of-date -Djib.requiredVersion=" + jib.MinimumJibMavenVersion + " --non-recursive prepare-package jib:build -Dimage=img"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			artifact := &latest.Artifact{
				ArtifactType: latest.ArtifactType{
					JibMavenArtifact: &latest.JibMavenArtifact{},
				},
			}

			builder := newBuilder(latest.GoogleCloudBuild{
				MavenImage: "maven:3.6.0",
			})
			builder.skipTests = test.skipTests

			buildSpec, err := builder.buildSpec(artifact, "img", "bucket", "object")
			t.CheckNoError(err)

			expected := []*cloudbuild.BuildStep{{
				Entrypoint: "sh",
				Name:       "maven:3.6.0",
				Args:       test.expectedArgs,
			}}

			t.CheckDeepEqual(expected, buildSpec.Steps)
			t.CheckDeepEqual(0, len(buildSpec.Images))
		})
	}
}

func TestJibGradleBuildSpec(t *testing.T) {
	tests := []struct {
		description  string
		skipTests    bool
		expectedArgs []string
	}{
		{
			description:  "skip tests",
			skipTests:    true,
			expectedArgs: []string{"-c", "gradle -Duser.home=$$HOME -Djib.console=plain _skaffoldFailIfJibOutOfDate -Djib.requiredVersion=" + jib.MinimumJibGradleVersion + " :jib --image=img -x test"},
		},
		{
			description:  "do not skip tests",
			skipTests:    false,
			expectedArgs: []string{"-c", "gradle -Duser.home=$$HOME -Djib.console=plain _skaffoldFailIfJibOutOfDate -Djib.requiredVersion=" + jib.MinimumJibGradleVersion + " :jib --image=img"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			artifact := &latest.Artifact{
				ArtifactType: latest.ArtifactType{
					JibGradleArtifact: &latest.JibGradleArtifact{},
				},
			}

			builder := newBuilder(latest.GoogleCloudBuild{
				GradleImage: "gradle:5.1.1",
			})
			builder.skipTests = test.skipTests

			buildSpec, err := builder.buildSpec(artifact, "img", "bucket", "object")
			t.CheckNoError(err)

			expected := []*cloudbuild.BuildStep{{
				Entrypoint: "sh",
				Name:       "gradle:5.1.1",
				Args:       test.expectedArgs,
			}}

			t.CheckDeepEqual(expected, buildSpec.Steps)
			t.CheckDeepEqual(0, len(buildSpec.Images))
		})
	}
}
