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
	"errors"
	"fmt"
	"os"

	cloudbuild "google.golang.org/api/cloudbuild/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util/stringslice"
)

// dockerBuildSpec lists the build steps required to build a docker image.
func (b *Builder) dockerBuildSpec(a *latest.Artifact, tag string, platforms platform.Matcher) (cloudbuild.Build, error) {
	a = adjustCacheFrom(a, tag)

	args, err := b.dockerBuildArgs(a, tag, a.Dependencies, platforms)
	if err != nil {
		return cloudbuild.Build{}, err
	}
	steps := b.cacheFromSteps(a.DockerArtifact, platforms)
	buildStep := &cloudbuild.BuildStep{
		Name: b.DockerImage,
		Args: args,
	}
	if platforms.IsNotEmpty() {
		// cross-platform build requires buildkit enabled
		buildStep.Env = append(buildStep.Env, "DOCKER_BUILDKIT=1")
	}
	steps = append(steps, buildStep)

	return cloudbuild.Build{
		Steps:  steps,
		Images: []string{tag},
	}, nil
}

// cacheFromSteps pulls images used by `--cache-from`.
func (b *Builder) cacheFromSteps(artifact *latest.DockerArtifact, platforms platform.Matcher) []*cloudbuild.BuildStep {
	var steps []*cloudbuild.BuildStep
	argFmt := "docker pull %s || true"
	if platforms.IsNotEmpty() {
		argFmt = "docker pull --platform " + platforms.String() + " %s || true"
	}
	for _, cacheFrom := range artifact.CacheFrom {
		steps = append(steps, &cloudbuild.BuildStep{
			Name:       b.DockerImage,
			Entrypoint: "sh",
			Args:       []string{"-c", fmt.Sprintf(argFmt, cacheFrom)},
		})
	}

	return steps
}

// dockerBuildArgs lists the arguments passed to `docker` to build a given image.
func (b *Builder) dockerBuildArgs(a *latest.Artifact, tag string, deps []*latest.ArtifactDependency, platforms platform.Matcher) ([]string, error) {
	d := a.DockerArtifact
	// TODO(nkubala): remove when buildkit is supported in GCB (#4773)
	if len(d.Secrets) > 0 || d.SSH != "" {
		return nil, errors.New("docker build options, secrets and ssh, are not currently supported in GCB builds")
	}
	requiredImages := docker.ResolveDependencyImages(deps, b.artifactStore, true)
	buildArgs, err := docker.EvalBuildArgs(b.cfg.Mode(), a.Workspace, d.DockerfilePath, d.BuildArgs, requiredImages)
	if err != nil {
		return nil, fmt.Errorf("unable to evaluate build args: %w", err)
	}

	ba, err := docker.ToCLIBuildArgs(d, buildArgs)
	if err != nil {
		return nil, fmt.Errorf("getting docker build args: %w", err)
	}

	args := []string{"build", "--tag", tag, "-f", d.DockerfilePath}
	if platforms.IsNotEmpty() {
		args = append(args, "--platform", platforms.String())
	}
	args = append(args, ba...)
	args = append(args, ".")

	return args, nil
}

// adjustCacheFrom returns  an artifact where any cache references from the artifactImage is changed to the tagged built image name instead.
func adjustCacheFrom(a *latest.Artifact, artifactTag string) *latest.Artifact {
	if os.Getenv("SKAFFOLD_DISABLE_GCB_CACHE_ADJUSTMENT") != "" {
		// allow this behaviour to be disabled
		return a
	}

	if !stringslice.Contains(a.DockerArtifact.CacheFrom, a.ImageName) {
		return a
	}

	cf := make([]string, 0, len(a.DockerArtifact.CacheFrom))
	for _, image := range a.DockerArtifact.CacheFrom {
		if image == a.ImageName {
			cf = append(cf, artifactTag)
		} else {
			cf = append(cf, image)
		}
	}
	copy := *a
	copy.DockerArtifact.CacheFrom = cf
	return &copy
}
