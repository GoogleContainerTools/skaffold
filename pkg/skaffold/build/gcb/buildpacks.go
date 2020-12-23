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
	"fmt"

	"github.com/sirupsen/logrus"
	cloudbuild "google.golang.org/api/cloudbuild/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/misc"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

func (b *Builder) buildpackBuildSpec(artifact *latest.BuildpackArtifact, tag string, deps []*latest.ArtifactDependency) (cloudbuild.Build, error) {
	args := []string{"pack", "build", tag, "--builder", fromRequiredArtifacts(artifact.Builder, b.artifactStore, deps)}

	if artifact.ProjectDescriptor != constants.DefaultProjectDescriptor {
		args = append(args, "--descriptor", artifact.ProjectDescriptor)
	}

	if artifact.RunImage != "" {
		args = append(args, "--run-image", fromRequiredArtifacts(artifact.RunImage, b.artifactStore, deps))
	}

	for _, buildpack := range artifact.Buildpacks {
		args = append(args, "--buildpack", buildpack)
	}

	if artifact.TrustBuilder {
		args = append(args, "--trust-builder")
	}

	env, err := misc.EvaluateEnv(artifact.Env)
	if err != nil {
		return cloudbuild.Build{}, fmt.Errorf("unable to evaluate env variables: %w", err)
	}

	for _, kv := range env {
		args = append(args, "--env", kv)
	}

	return cloudbuild.Build{
		Steps: []*cloudbuild.BuildStep{{
			Name: b.PackImage,
			Args: args,
		}},
		Images: []string{tag},
	}, nil
}

// fromRequiredArtifacts replaces the provided image name with image from the required artifacts if matched.
func fromRequiredArtifacts(imageName string, r docker.ArtifactResolver, deps []*latest.ArtifactDependency) string {
	for _, d := range deps {
		if imageName == d.Alias {
			image, found := r.GetImageTag(d.ImageName)
			if !found {
				logrus.Fatalf("failed to resolve build result for required artifact %q", d.ImageName)
			}
			return image
		}
	}
	return imageName
}
