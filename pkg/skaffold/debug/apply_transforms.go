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

package debug

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/debug/types"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
)

var ConfigRetriever = func(ctx context.Context, image string, builds []graph.Artifact, registries map[string]bool) (ImageConfiguration, error) {
	if artifact := findArtifact(image, builds); artifact != nil {
		return RetrieveImageConfiguration(ctx, artifact, registries)
	}
	return ImageConfiguration{}, fmt.Errorf("no build artifact for %q", image)
}

// findArtifact finds the corresponding artifact for the given image.
// If `builds` is empty, then treat all `image` images as a build artifact.
func findArtifact(image string, builds []graph.Artifact) *graph.Artifact {
	if len(builds) == 0 {
		log.Entry(context.TODO()).Debugf("No build artifacts specified: using image as-is %q", image)
		return &graph.Artifact{ImageName: image, Tag: image}
	}
	for _, artifact := range builds {
		if image == artifact.ImageName || image == artifact.Tag {
			log.Entry(context.TODO()).Debugf("Found artifact for image %q", image)
			return &artifact
		}
	}
	return nil
}

// RetrieveImageConfiguration retrieves the image container configuration for
// the given build artifact
func RetrieveImageConfiguration(ctx context.Context, artifact *graph.Artifact, insecureRegistries map[string]bool) (ImageConfiguration, error) {
	// TODO: use the proper RunContext
	apiClient, err := docker.NewAPIClient(ctx, &runcontext.RunContext{
		InsecureRegistries: insecureRegistries,
	})
	if err != nil {
		return ImageConfiguration{}, fmt.Errorf("could not connect to local docker daemon: %w", err)
	}

	// the apiClient will go to the remote registry if local docker daemon is not available
	manifest, err := apiClient.ConfigFile(ctx, artifact.Tag)
	if err != nil {
		log.Entry(ctx).Debugf("Error retrieving image manifest for %v: %v", artifact.Tag, err)
		return ImageConfiguration{}, fmt.Errorf("retrieving image config for %q: %w", artifact.Tag, err)
	}

	config := manifest.Config
	log.Entry(ctx).Debugf("Retrieved local image configuration for %v: %v", artifact.Tag, config)
	// need to duplicate slices as apiClient caches requests
	return ImageConfiguration{
		Artifact:    artifact.ImageName,
		RuntimeType: types.ToRuntime(artifact.RuntimeType),
		Author:      manifest.Author,
		Env:         envAsMap(config.Env),
		Entrypoint:  dupArray(config.Entrypoint),
		Arguments:   dupArray(config.Cmd),
		Labels:      dupMap(config.Labels),
		WorkingDir:  config.WorkingDir,
	}, nil
}

// envAsMap turns an array of environment "NAME=value" strings into a map
func envAsMap(env []string) map[string]string {
	result := make(map[string]string)
	for _, pair := range env {
		s := strings.SplitN(pair, "=", 2)
		result[s[0]] = s[1]
	}
	return result
}

func dupArray(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	dup := make([]string, len(s))
	copy(dup, s)
	return dup
}

func dupMap(s map[string]string) map[string]string {
	if len(s) == 0 {
		return nil
	}
	dup := make(map[string]string, len(s))
	for k, v := range s {
		dup[k] = v
	}
	return dup
}
