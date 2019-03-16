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

package debugging

import (
	"bufio"
	"bytes"
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/runtime"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	config "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

var (
	decodeFromYaml = scheme.Codecs.UniversalDeserializer().Decode
	encodeAsYaml   = func(o runtime.Object) ([]byte, error) {
		s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
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
func ApplyDebuggingTransforms(l kubectl.ManifestList, builds []build.Artifact) (kubectl.ManifestList, error) {
	var updated kubectl.ManifestList

	retriever := func(image string) (imageConfiguration, error) {
		if artifact := findArtifact(image, builds); artifact != nil {
			return retrieveImageConfiguration(image, artifact)
		}
		return imageConfiguration{}, errors.Errorf("no build artifact for [%q]", image)
	}

	for _, manifest := range l {
		obj, _, err := decodeFromYaml(manifest, nil, nil)
		if err != nil {
			return nil, errors.Wrap(err, "reading kubernetes YAML")
		}

		if transformManifest(obj, retriever) {
			manifest, err = encodeAsYaml(obj)
			if err != nil {
				return nil, errors.Wrap(err, "marshalling yaml")
			}
			if logrus.IsLevelEnabled(logrus.DebugLevel) {
				logrus.Debugln("Applied debugging transform:\n", string(manifest))
			}
		}
		updated = append(updated, manifest)
	}

	return updated, nil
}

// findArtifact finds the corresponding artifact for the given image
func findArtifact(image string, builds []build.Artifact) *build.Artifact {
	for _, artifact := range builds {
		if image == artifact.ImageName || image == artifact.Tag {
			logrus.Debugf("Found artifact for image [%s]", image)
			return &artifact
		}
	}
	return nil
}

// retrieveImageConfiguration retrieves the image container configuration for
// the given build artifact
func retrieveImageConfiguration(image string, artifact *build.Artifact) (imageConfiguration, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var config config.Config
	var err error
	if artifact.Location == build.ToLocalDocker {
		config, err = retrieveDockerConfiguration(ctx, artifact.Tag)
	} else {
		config, err = retrieveRegistryConfiguration(artifact.Tag)
	}
	if err != nil {
		return imageConfiguration{}, errors.Wrapf(err, "unable to retrieve image configuration [%q]", image)
	}

	return imageConfiguration{
		env:        envAsMap(config.Env),
		entrypoint: config.Entrypoint,
		arguments:  config.Cmd,
		labels:     config.Labels,
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

// retrieveRegistryConfiguration retrieves an image configuration from a registry
func retrieveRegistryConfiguration(image string) (config.Config, error) {
	logrus.Debugf("Retrieving image configuration for %v", image)
	ref, err := name.ParseReference(image, name.WeakValidation)
	if err != nil {
		logrus.Debugf("Error parsing image %v: %v", image, err)
		return config.Config{}, errors.Wrapf(err, "parsing image %q", image)
	}

	auth, err := authn.DefaultKeychain.Resolve(ref.Context().Registry)
	if err != nil {
		return config.Config{}, errors.Wrap(err, "getting default keychain auth")
	}

	remoteImage, err := remote.Image(ref, remote.WithAuth(auth))
	if err != nil {
		logrus.Debugf("Error retrieving remote image details %v: %v", image, err)
		return config.Config{}, errors.Wrapf(err, "retrieving image %q", ref)
	}

	manifest, err := remoteImage.ConfigFile()
	if err != nil {
		logrus.Debugf("Error retrieving remote image manifest %v: %v", image, err)
		return config.Config{}, errors.Wrapf(err, "retrieving image config for %q", ref)
	}
	return manifest.Config, nil
}

// retrieveDockerConfiguration retrieves an image configuration from a local docker daemon
func retrieveDockerConfiguration(ctx context.Context, image string) (config.Config, error) {
	localDocker, err := docker.NewAPIClient()
	if err != nil {
		return config.Config{}, errors.Wrap(err, "could not connect to local docker daemon")
	}
	manifest, err := localDocker.ConfigFile(ctx, image)
	if err != nil {
		logrus.Debugf("Error retrieving local image manifest for %v: %v", image, err)
		return config.Config{}, errors.Wrapf(err, "retrieving image config for %q", image)
	}
	logrus.Debugf("Retrieved local image configuration for %v: %v", image, manifest.Config)
	return manifest.Config, nil
}
