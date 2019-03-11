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
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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

func remoteImage(identifier string, insecureRegistries map[string]bool) (v1.Image, error) {
	ref, err := name.ParseReference(identifier, name.WeakValidation)
	if err != nil {
		return nil, errors.Wrap(err, "parsing initial ref")
	}

	reg := ref.Context().Registry
	if _, ok := insecureRegistries[reg.Name()]; ok {
		reg, err = name.NewInsecureRegistry(reg.Name(), name.StrictValidation)
		if err != nil {
			logrus.Warnf("error getting insecure registry: %s\nremote references may not be retrieved", err.Error())
		}
	}

	auth, err := authn.DefaultKeychain.Resolve(reg)
	if err != nil {
		return nil, errors.Wrap(err, "getting default keychain auth")
	}

	return remote.Image(ref, remote.WithAuth(auth))
}
