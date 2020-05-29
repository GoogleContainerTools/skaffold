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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/misc"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

func (b *Builder) buildpackBuildSpec(artifact *latest.BuildpackArtifact, tag string) (cloudbuild.Build, error) {
	args := []string{"pack", "build", tag, "--builder", artifact.Builder}

	if artifact.ProjectDescriptor != constants.DefaultProjectDescriptor {
		args = append(args, "--descriptor", artifact.ProjectDescriptor)
	}

	if artifact.RunImage != "" {
		args = append(args, "--run-image", artifact.RunImage)
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
