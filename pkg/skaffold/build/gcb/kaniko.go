package gcb

import (
	"fmt"
	"sort"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
	"google.golang.org/api/cloudbuild/v1"
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

	steps := []*cloudbuild.BuildStep{
		{
			Name: b.KanikoImage,
			Args: kanikoArgs,
		},
	}
	return cloudbuild.Build{
		Steps: steps,
	}, nil
}

func (b *Builder) kanikoBuildArgs(artifact *latest.KanikoArtifact) ([]string, error) {
	buildArgs, err := docker.EvaluateBuildArgs(artifact.BuildArgs)
	if err != nil || buildArgs == nil {
		return nil, errors.Wrap(err, "unable to evaluate build args")
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
