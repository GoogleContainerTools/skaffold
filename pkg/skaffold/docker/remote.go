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
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// for testing
var (
	getRemoteImageImpl = getRemoteImage
	RemoteDigest       = getRemoteDigest
)

// Remote Operations

func (d *dockerAPI) InsecureRegistries() map[string]bool {
	return d.insecureRegistries
}

func (d *dockerAPI) RemoteDigest(identifier string) (string, error) {
	return RemoteDigest(identifier, d.insecureRegistries)
}

func (d *dockerAPI) AddRemoteTag(src, target string) error {
	logrus.Debugf("attempting to add tag %s to src %s", target, src)

	img, err := remoteImage(src, d.insecureRegistries)
	if err != nil {
		return err
	}

	targetRef, err := name.ParseReference(target, name.WeakValidation)
	if err != nil {
		return errors.Wrap(err, "getting target reference")
	}

	if IsInsecure(targetRef.Context().Registry.Name(), d.insecureRegistries) {
		targetRef, err = name.ParseReference(target, name.Insecure)
		if err != nil {
			logrus.Warnf("error getting insecure registry: %s\nremote references may not be retrieved", err.Error())
		}
	}

	return remote.Write(targetRef, img, remote.WithAuth(authenticators.For(targetRef)))
}

// RetrieveRemoteConfig retrieves the remote config file for an image
func (d *dockerAPI) RetrieveRemoteConfig(identifier string) (*v1.ConfigFile, error) {
	img, err := remoteImage(identifier, d.insecureRegistries)
	if err != nil {
		return nil, errors.Wrap(err, "getting image")
	}

	return img.ConfigFile()
}

// Push pushes the tarball image
func (d *dockerAPI) PushTar(tarPath, tag string) (string, error) {
	t, err := name.NewTag(tag, name.WeakValidation)
	if err != nil {
		return "", errors.Wrapf(err, "parsing tag %q", tag)
	}

	i, err := tarball.ImageFromPath(tarPath, nil)
	if err != nil {
		return "", errors.Wrapf(err, "reading image %q", tarPath)
	}

	if err := remote.Write(t, i, remote.WithAuth(authenticators.For(t))); err != nil {
		return "", errors.Wrapf(err, "writing image %q", t)
	}

	return getRemoteDigest(tag, d.insecureRegistries)
}

// Implementation

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

func remoteImage(identifier string, insecureRegistries map[string]bool, opts ...name.Option) (v1.Image, error) {
	ref, err := name.ParseReference(identifier, opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing reference [%s]", identifier)
	}

	if IsInsecure(ref.Context().Registry.Name(), insecureRegistries) {
		ref, err = name.ParseReference(identifier, name.Insecure)
		if err != nil {
			logrus.Warnf("error getting insecure registry: %s\nremote references may not be retrieved", err.Error())
		}
	}

	return getRemoteImageImpl(ref)
}

// IsInsecure tests if the registry is listed as an insecure registry; default is false
func IsInsecure(reg string, insecureRegistries map[string]bool) bool {
	return insecureRegistries[reg]
}

func getRemoteImage(ref name.Reference) (v1.Image, error) {
	return remote.Image(ref, remote.WithAuth(authenticators.For(ref)))
}
