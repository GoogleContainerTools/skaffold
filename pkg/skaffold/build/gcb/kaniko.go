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
	v1 "k8s.io/api/core/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/kaniko"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/misc"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
)

func (b *Builder) kanikoBuildSpec(a *latestV1.Artifact, tag string) (cloudbuild.Build, error) {
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

	env, err := misc.EvaluateEnv(envFromVars(k.Env))
	if err != nil {
		return cloudbuild.Build{}, fmt.Errorf("unable to evaluate env variables: %w", err)
	}

	return cloudbuild.Build{
		Steps: []*cloudbuild.BuildStep{{
			Name: b.KanikoImage,
			Args: kanikoArgs,
			Env:  env,
		}},
	}, nil
}

func envFromVars(env []v1.EnvVar) []string {
	s := make([]string, 0, len(env))
	for _, envVar := range env {
		s = append(s, envVar.Name+"="+envVar.Value)
	}
	return s
}
