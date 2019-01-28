/*
Copyright 2018 The Skaffold Authors

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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
	cloudbuild "google.golang.org/api/cloudbuild/v1"
)

func TestJibMavenBuildSteps(t *testing.T) {
	artifact := &latest.JibMavenArtifact{}

	builder := Builder{
		GoogleCloudBuild: &latest.GoogleCloudBuild{
			MavenImage: "maven:3.6.0",
		},
	}
	steps := builder.jibMavenBuildSteps("img", artifact)

	expected := []*cloudbuild.BuildStep{{
		Name: "maven:3.6.0",
		Args: []string{"--non-recursive", "prepare-package", "jib:dockerBuild", "-Dimage=img"},
	}}

	testutil.CheckDeepEqual(t, expected, steps)
}

func TestJibGradleBuildSteps(t *testing.T) {
	artifact := &latest.JibGradleArtifact{}

	builder := Builder{
		GoogleCloudBuild: &latest.GoogleCloudBuild{
			GradleImage: "gradle:5.1.1",
		},
	}
	steps := builder.jibGradleBuildSteps("img", artifact)

	expected := []*cloudbuild.BuildStep{{
		Name: "gradle:5.1.1",
		Args: []string{":jibDockerBuild", "--image=img"},
	}}

	testutil.CheckDeepEqual(t, expected, steps)
}
