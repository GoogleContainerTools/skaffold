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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
	cloudbuild "google.golang.org/api/cloudbuild/v1"
)

func (b *Builder) dockerBuildSteps(artifact *latest.DockerArtifact, tags []string) ([]*cloudbuild.BuildStep, error) {
	var steps []*cloudbuild.BuildStep

	for _, cacheFrom := range artifact.CacheFrom {
		steps = append(steps, &cloudbuild.BuildStep{
			Name:       b.DockerImage,
			Entrypoint: "sh",
			Args:       []string{"-c", fmt.Sprintf("docker pull %s || true", cacheFrom)},
		})
	}

	args := []string{"build"}
	for _, t := range tags {
		args = append(args, []string{"--tag", t}...)
	}
	args = append(args, []string{"-f", artifact.DockerfilePath}...)
	ba, err := docker.GetBuildArgs(artifact)
	if err != nil {
		return nil, errors.Wrap(err, "getting docker build args")
	}
	args = append(args, ba...)
	args = append(args, ".")

	return append(steps, &cloudbuild.BuildStep{
		Name: b.DockerImage,
		Args: args,
	}), nil
}
