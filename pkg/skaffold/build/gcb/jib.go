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
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/jib"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	cloudbuild "google.golang.org/api/cloudbuild/v1"
)

func (b *Builder) jibMavenBuildSpec(artifact *latest.JibMavenArtifact, tag string) cloudbuild.Build {
	return cloudbuild.Build{
		Steps: []*cloudbuild.BuildStep{{
			Name:       b.MavenImage,
			Entrypoint: "sh",
			Args:       fixHome("mvn", jib.GenerateMavenArgs("build", tag, artifact, b.skipTests)),
		}},
	}
}

func (b *Builder) jibGradleBuildSpec(artifact *latest.JibGradleArtifact, tag string) cloudbuild.Build {
	return cloudbuild.Build{
		Steps: []*cloudbuild.BuildStep{{
			Name:       b.GradleImage,
			Entrypoint: "sh",
			Args:       fixHome("gradle", jib.GenerateGradleArgs("jib", tag, artifact, b.skipTests)),
		}},
	}
}

func fixHome(command string, args []string) []string {
	return []string{"-c", command + " -Duser.home=$$HOME " + strings.Join(args, " ")}
}
