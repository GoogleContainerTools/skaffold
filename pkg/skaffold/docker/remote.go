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
	"net/http"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	// for testing
	getInsecureRegistryImpl = getInsecureRegistry
	getRemoteImageImpl      = getRemoteImage
)

func RemoteDigest(identifier string, insecureRegistries map[string]bool) (string, error) {
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

	auth, err := authn.DefaultKeychain.Resolve(t.Registry)
	if err != nil {
		return "", errors.Wrapf(err, "getting creds for %q", t)
	}

	i, err := tarball.ImageFromPath(tarPath, nil)
	if err != nil {
		return "", errors.Wrapf(err, "reading image %q", tarPath)
	}

	if err := remote.Write(t, i, auth, http.DefaultTransport); err != nil {
		return "", errors.Wrapf(err, "writing image %q", t)
	}

	return RemoteDigest(tag, insecureRegistries)
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
	auth, err := authn.DefaultKeychain.Resolve(ref.Context().Registry)
	if err != nil {
		return nil, errors.Wrap(err, "getting default keychain auth")
	}

	return remote.Image(ref, remote.WithAuth(auth))
}
