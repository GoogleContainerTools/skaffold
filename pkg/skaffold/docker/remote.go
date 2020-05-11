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

package docker

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/sirupsen/logrus"
)

// for testing
var (
	getInsecureRegistryImpl = getInsecureRegistry
	getRemoteImageImpl      = getRemoteImage
	RemoteDigest            = getRemoteDigest
)

func AddRemoteTag(src, target string, insecureRegistries map[string]bool) error {
	logrus.Debugf("attempting to add tag %s to src %s", target, src)
	img, err := remoteImage(src, insecureRegistries)
	if err != nil {
		return fmt.Errorf("getting image: %w", err)
	}

	targetRef, err := name.ParseReference(target, name.WeakValidation)
	if err != nil {
		return fmt.Errorf("getting target reference: %w", err)
	}

	if IsInsecure(targetRef.Context().Registry.Name(), insecureRegistries) {
		targetRef, err = getInsecureRegistryImpl(target)
		if err != nil {
			logrus.Warnf("error getting insecure registry: %s\nremote references may not be retrieved", err.Error())
		}
	}

	return remote.Write(targetRef, img, remote.WithAuthFromKeychain(masterKeychain))
}

func getRemoteDigest(identifier string, insecureRegistries map[string]bool) (string, error) {
	img, err := remoteImage(identifier, insecureRegistries)
	if err != nil {
		return "", fmt.Errorf("getting image: %w", err)
	}

	h, err := img.Digest()
	if err != nil {
		return "", fmt.Errorf("getting digest: %w", err)
	}

	return h.String(), nil
}

// RetrieveRemoteConfig retrieves the remote config file for an image
func RetrieveRemoteConfig(identifier string, insecureRegistries map[string]bool) (*v1.ConfigFile, error) {
	img, err := remoteImage(identifier, insecureRegistries)
	if err != nil {
		return nil, err
	}

	return img.ConfigFile()
}

// Push pushes the tarball image
func Push(tarPath, tag string, insecureRegistries map[string]bool) (string, error) {
	t, err := name.NewTag(tag, name.WeakValidation)
	if err != nil {
		return "", fmt.Errorf("parsing tag %q: %w", tag, err)
	}

	i, err := tarball.ImageFromPath(tarPath, nil)
	if err != nil {
		return "", fmt.Errorf("reading image %q: %w", tarPath, err)
	}

	if err := remote.Write(t, i, remote.WithAuthFromKeychain(masterKeychain)); err != nil {
		return "", fmt.Errorf("writing image %q: %w", t, err)
	}

	return getRemoteDigest(tag, insecureRegistries)
}

func remoteImage(identifier string, insecureRegistries map[string]bool) (v1.Image, error) {
	ref, err := name.ParseReference(identifier)
	if err != nil {
		return nil, fmt.Errorf("parsing reference [%s]: %w", identifier, err)
	}

	if IsInsecure(ref.Context().Registry.Name(), insecureRegistries) {
		ref, err = getInsecureRegistryImpl(identifier)
		if err != nil {
			logrus.Warnf("error getting insecure registry: %s\nremote references may not be retrieved", err.Error())
		}
	}

	return getRemoteImageImpl(ref)
}

func getInsecureRegistry(identifier string) (name.Reference, error) {
	ref, err := name.ParseReference(identifier, name.Insecure)
	if err != nil {
		return nil, fmt.Errorf("parsing reference %q: %w", identifier, err)
	}
	return ref, nil
}

// IsInsecure tests if the registry is listed as an insecure registry; default is false
func IsInsecure(reg string, insecureRegistries map[string]bool) bool {
	return insecureRegistries[reg]
}

func getRemoteImage(ref name.Reference) (v1.Image, error) {
	return remote.Image(ref, remote.WithAuthFromKeychain(masterKeychain))
}
