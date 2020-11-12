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

	"google.golang.org/api/cloudbuild/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/kaniko"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

func (b *Builder) kanikoBuildSpec(a *latest.Artifact, tag string) (cloudbuild.Build, error) {
	k := a.KanikoArtifact
	requiredImages := docker.ResolveDependencyImages(a.Dependencies, b.artifactStore, true)
	// add required artifacts as build args
	buildArgs, err := docker.EvalBuildArgs(b.cfg.Mode(), a.Workspace, k.DockerfilePath, k.BuildArgs, requiredImages)
	if err != nil {
		return cloudbuild.Build{}, fmt.Errorf("unable to evaluate build args: %w", err)
	}
	k.BuildArgs = buildArgs
	kanikoArgs, err := kaniko.Args(k, tag, "")
	if err != nil {
		return cloudbuild.Build{}, err
	}

	return cloudbuild.Build{
		Steps: []*cloudbuild.BuildStep{{
			Name: b.KanikoImage,
			Args: kanikoArgs,
		}},
	}, nil
}
