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
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/pkg/errors"
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
		return errors.Wrap(err, "getting image")
	}

	targetRef, err := name.ParseReference(target, name.WeakValidation)
	if err != nil {
		return errors.Wrap(err, "getting target reference")
	}

	return remote.Write(targetRef, img, remote.WithAuthFromKeychain(authn.DefaultKeychain))
}

func getRemoteDigest(identifier string, insecureRegistries map[string]bool) (string, error) {
	img, err := remoteImage(identifier, insecureRegistries)
	if err != nil {
		return "", errors.Wrap(err, "getting image")
	}

	h, err := img.Digest()
	if err != nil {
		return "", errors.Wrap(err, "getting digest")
	}

	return h.String(), nil
}

// RetrieveRemoteConfig retrieves the remote config file for an image
func RetrieveRemoteConfig(identifier string, insecureRegistries map[string]bool) (*v1.ConfigFile, error) {
	img, err := remoteImage(identifier, insecureRegistries)
	if err != nil {
		return nil, errors.Wrap(err, "getting image")
	}

	return img.ConfigFile()
}

// Push pushes the tarball image
func Push(tarPath, tag string, insecureRegistries map[string]bool) (string, error) {
	t, err := name.NewTag(tag, name.WeakValidation)
	if err != nil {
		return "", errors.Wrapf(err, "parsing tag %q", tag)
	}

	i, err := tarball.ImageFromPath(tarPath, nil)
	if err != nil {
		return "", errors.Wrapf(err, "reading image %q", tarPath)
	}

	if err := remote.Write(t, i, remote.WithAuthFromKeychain(authn.DefaultKeychain)); err != nil {
		return "", errors.Wrapf(err, "writing image %q", t)
	}

	return getRemoteDigest(tag, insecureRegistries)
}

func remoteImage(identifier string, insecureRegistries map[string]bool) (v1.Image, error) {
	ref, err := name.ParseReference(identifier)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing reference [%s]", identifier)
	}

	if isInsecure(ref.Context().Registry.Name(), insecureRegistries) {
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
		return nil, errors.Wrapf(err, "parsing reference [%s]", identifier)
	}
	return ref, nil
}

func isInsecure(ref string, insecureRegistries map[string]bool) bool {
	_, ok := insecureRegistries[ref]
	return ok
}

func getRemoteImage(ref name.Reference) (v1.Image, error) {
	return remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
}
