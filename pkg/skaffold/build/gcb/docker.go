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

	cloudbuild "google.golang.org/api/cloudbuild/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// dockerBuildSpec lists the build steps required to build a docker image.
func (b *Builder) dockerBuildSpec(artifact *latest.DockerArtifact, tag string) (cloudbuild.Build, error) {
	args, err := b.dockerBuildArgs(artifact, tag)
	if err != nil {
		return cloudbuild.Build{}, err
	}

	steps := b.cacheFromSteps(artifact)
	steps = append(steps, &cloudbuild.BuildStep{
		Name: b.DockerImage,
		Args: args,
	})

	return cloudbuild.Build{
		Steps:  steps,
		Images: []string{tag},
	}, nil
}

// cacheFromSteps pulls images used by `--cache-from`.
func (b *Builder) cacheFromSteps(artifact *latest.DockerArtifact) []*cloudbuild.BuildStep {
	var steps []*cloudbuild.BuildStep

	for _, cacheFrom := range artifact.CacheFrom {
		steps = append(steps, &cloudbuild.BuildStep{
			Name:       b.DockerImage,
			Entrypoint: "sh",
			Args:       []string{"-c", fmt.Sprintf("docker pull %s || true", cacheFrom)},
		})
	}

	return steps
}

// dockerBuildArgs lists the arguments passed to `docker` to build a given image.
func (b *Builder) dockerBuildArgs(artifact *latest.DockerArtifact, tag string) ([]string, error) {
	ba, err := docker.GetBuildArgs(artifact)
	if err != nil {
		return nil, fmt.Errorf("getting docker build args: %w", err)
	}

	args := []string{"build", "--tag", tag, "-f", artifact.DockerfilePath}
	args = append(args, ba...)
	args = append(args, ".")

	return args, nil
}
