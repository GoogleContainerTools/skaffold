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
	"bufio"
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
)

var (
	decodeFromYaml = scheme.Codecs.UniversalDeserializer().Decode
	encodeAsYaml   = func(o runtime.Object) ([]byte, error) {
		s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
		var b bytes.Buffer
		w := bufio.NewWriter(&b)
		if err := s.Encode(o, w); err != nil {
			return nil, err
		}
		w.Flush()
		return b.Bytes(), nil
	}
)

// ApplyDebuggingTransforms applies language-platform-specific transforms to a list of manifests.
func ApplyDebuggingTransforms(l manifest.ManifestList, builds []graph.Artifact, registries manifest.Registries) (manifest.ManifestList, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	retriever := func(image string) (imageConfiguration, error) {
		if artifact := findArtifact(image, builds); artifact != nil {
			return retrieveImageConfiguration(ctx, artifact, registries.InsecureRegistries)
		}
		return imageConfiguration{}, fmt.Errorf("no build artifact for %q", image)
	}
	return applyDebuggingTransforms(l, retriever, registries.DebugHelpersRegistry)
}

func applyDebuggingTransforms(l manifest.ManifestList, retriever configurationRetriever, debugHelpersRegistry string) (manifest.ManifestList, error) {
	var updated manifest.ManifestList
	for _, manifest := range l {
		obj, _, err := decodeFromYaml(manifest, nil, nil)
		if err != nil {
			log.Entry(context.Background()).Debugf("Unable to interpret manifest for debugging: %v\n", err)
		} else if transformManifest(obj, retriever, debugHelpersRegistry) {
			manifest, err = encodeAsYaml(obj)
			if err != nil {
				return nil, fmt.Errorf("marshalling yaml: %w", err)
			}
			if logrus.IsLevelEnabled(logrus.DebugLevel) {
				log.Entry(context.Background()).Debugln("Applied debugging transform:\n", string(manifest))
			}
		}
		updated = append(updated, manifest)
	}

	return updated, nil
}

// findArtifact finds the corresponding artifact for the given image.
// If `builds` is empty, then treat all `image` images as a build artifact.
func findArtifact(image string, builds []graph.Artifact) *graph.Artifact {
	if len(builds) == 0 {
		log.Entry(context.Background()).Debugf("No build artifacts specified: using image as-is %q", image)
		return &graph.Artifact{ImageName: image, Tag: image}
	}
	for _, artifact := range builds {
		if image == artifact.ImageName || image == artifact.Tag {
			log.Entry(context.Background()).Debugf("Found artifact for image %q", image)
			return &artifact
		}
	}
	return nil
}

// retrieveImageConfiguration retrieves the image container configuration for
// the given build artifact
func retrieveImageConfiguration(ctx context.Context, artifact *graph.Artifact, insecureRegistries map[string]bool) (imageConfiguration, error) {
	// TODO: use the proper RunContext
	apiClient, err := docker.NewAPIClient(&runcontext.RunContext{
		InsecureRegistries: insecureRegistries,
	})
	if err != nil {
		return imageConfiguration{}, fmt.Errorf("could not connect to local docker daemon: %w", err)
	}

	// the apiClient will go to the remote registry if local docker daemon is not available
	manifest, err := apiClient.ConfigFile(ctx, artifact.Tag)
	if err != nil {
		log.Entry(ctx).Debugf("Error retrieving image manifest for %v: %v", artifact.Tag, err)
		return imageConfiguration{}, fmt.Errorf("retrieving image config for %q: %w", artifact.Tag, err)
	}

	config := manifest.Config
	log.Entry(ctx).Debugf("Retrieved local image configuration for %v: %v", artifact.Tag, config)
	// need to duplicate slices as apiClient caches requests
	return imageConfiguration{
		artifact:   artifact.ImageName,
		env:        envAsMap(config.Env),
		entrypoint: dupArray(config.Entrypoint),
		arguments:  dupArray(config.Cmd),
		labels:     dupMap(config.Labels),
		workingDir: config.WorkingDir,
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
