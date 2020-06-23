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
	"sort"

	"google.golang.org/api/cloudbuild/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

func (b *Builder) kanikoBuildSpec(artifact *latest.KanikoArtifact, tag string) (cloudbuild.Build, error) {
	buildArgs, err := b.kanikoBuildArgs(artifact)
	if err != nil {
		return cloudbuild.Build{}, err
	}

	kanikoArgs := []string{
		"--destination", tag,
		"--dockerfile", artifact.DockerfilePath,
	}
	kanikoArgs = append(kanikoArgs, buildArgs...)

	if artifact.Cache != nil {
		kanikoArgs = append(kanikoArgs, "--cache")

		if artifact.Cache.Repo != "" {
			kanikoArgs = append(kanikoArgs, "--cache-repo", artifact.Cache.Repo)
		}
	}

	if artifact.Reproducible {
		kanikoArgs = append(kanikoArgs, "--reproducible")
	}

	if artifact.Target != "" {
		kanikoArgs = append(kanikoArgs, "--target", artifact.Target)
	}

	return cloudbuild.Build{
		Steps: []*cloudbuild.BuildStep{{
			Name: b.KanikoImage,
			Args: kanikoArgs,
		}},
	}, nil
}

func (b *Builder) kanikoBuildArgs(artifact *latest.KanikoArtifact) ([]string, error) {
	buildArgs, err := docker.EvaluateBuildArgs(artifact.BuildArgs)
	if err != nil {
		return nil, fmt.Errorf("unable to evaluate build args: %w", err)
	}

	var keys []string
	for k := range buildArgs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buildArgFlags []string
	for _, k := range keys {
		v := buildArgs[k]
		if v == nil {
			buildArgFlags = append(buildArgFlags, "--build-arg", k)
		} else {
			buildArgFlags = append(buildArgFlags, "--build-arg", fmt.Sprintf("%s=%s", k, *v))
		}
	}

	return buildArgFlags, nil
}
