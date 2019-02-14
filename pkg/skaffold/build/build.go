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

package build

import (
	"context"
	"io"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	config "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Artifact is the result corresponding to each successful build.
type Artifact struct {
	ImageName string
	Tag       string
	Config    ConfigurationRetriever
}

// Builder is an interface to the Build API of Skaffold.
// It must build and make the resulting image accesible to the cluster.
// This could include pushing to a authorized repository or loading the nodes with the image.
// If artifacts is supplied, the builder should only rebuild those artifacts.
type Builder interface {
	Labels() map[string]string

	Build(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]Artifact, error)
}

// ConfigurationRetriever is a function signature for a function that returns an OCI image configuration.
// There are two ConfigurationRetrievers available:
//
// - RegistryConfigurationRetriever for querying a registry based on the provided image reference
//
// - DockerConfigurationRetriever for querying the local docker daemon's cache
type ConfigurationRetriever func(ctx context.Context) (config.Config, error)

// RegistryConfigurationRetriever returns a function to retrieve an image configuration from a registry
func RegistryConfigurationRetriever(image string) ConfigurationRetriever {
	return func(ctx context.Context) (config.Config, error) {
		logrus.Debugf("Retrieving image configuration for %v", image)
		ref, err := parseReference(image)
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
}

func parseReference(image string) (name.Reference, error) {
	ref, err := name.ParseReference(image, name.WeakValidation)
	if err == nil {
		return ref, nil
	}
	if parts := strings.Split(image, "@"); len(parts) == 2 {
		// workaround https://github.com/google/go-containerregistry/issues/351 and strip tag
		if tagged, err := name.NewTag(parts[0], name.WeakValidation); err == nil {
			image = tagged.Repository.Name() + "@" + parts[1]
			return name.ParseReference(image, name.WeakValidation)
		}
	}
	return nil, err
}

// DockerConfigurationRetriever returns a function to retrieve an image configuration from a local docker daemon
func DockerConfigurationRetriever(localDocker docker.LocalDaemon, image string) ConfigurationRetriever {
	return func(ctx context.Context) (config.Config, error) {
		manifest, err := localDocker.ConfigFile(ctx, image)
		if err != nil {
			logrus.Debugf("Error retrieving local image manifest for %v: %v", image, err)
			return config.Config{}, errors.Wrapf(err, "retrieving image config for %q", image)
		}
		logrus.Debugf("Retrieved local image configuration for %v: %v", image, manifest.Config)
		return manifest.Config, nil
	}
}
