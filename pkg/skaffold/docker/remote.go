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

	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
)

// for testing
var (
	RemoteDigest = getRemoteDigest
	remoteImage  = remote.Image
	remoteIndex  = remote.Index
)

func AddRemoteTag(src, target string, insecureRegistries map[string]bool) error {
	logrus.Debugf("attempting to add tag %s to src %s", target, src)
	img, err := getRemoteImage(src, insecureRegistries)
	if err != nil {
		return fmt.Errorf("getting image: %w", err)
	}

	targetRef, err := parseReference(target, insecureRegistries, name.WeakValidation)
	if err != nil {
		return err
	}

	return remote.Write(targetRef, img, remote.WithAuthFromKeychain(primaryKeychain))
}

func getRemoteDigest(identifier string, insecureRegistries map[string]bool) (string, error) {
	idx, err := getRemoteIndex(identifier, insecureRegistries)
	if err == nil {
		return digest(idx)
	}

	img, err := getRemoteImage(identifier, insecureRegistries)
	if err != nil {
		return "", fmt.Errorf("getting image: %w", err)
	}

	return digest(img)
}

// RetrieveRemoteConfig retrieves the remote config file for an image
func RetrieveRemoteConfig(identifier string, insecureRegistries map[string]bool) (*v1.ConfigFile, error) {
	img, err := getRemoteImage(identifier, insecureRegistries)
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

	if err := remote.Write(t, i, remote.WithAuthFromKeychain(primaryKeychain)); err != nil {
		return "", fmt.Errorf("%s %q: %w", sErrors.PushImageErrPrefix, t, err)
	}

	return getRemoteDigest(tag, insecureRegistries)
}

func getRemoteImage(identifier string, insecureRegistries map[string]bool) (v1.Image, error) {
	ref, err := parseReference(identifier, insecureRegistries)
	if err != nil {
		return nil, err
	}

	return remoteImage(ref, remote.WithAuthFromKeychain(primaryKeychain))
}

func getRemoteIndex(identifier string, insecureRegistries map[string]bool) (v1.ImageIndex, error) {
	ref, err := parseReference(identifier, insecureRegistries)
	if err != nil {
		return nil, err
	}

	return remoteIndex(ref, remote.WithAuthFromKeychain(primaryKeychain))
}

// IsInsecure tests if an image is pulled from an insecure registry; default is false
func IsInsecure(ref name.Reference, insecureRegistries map[string]bool) bool {
	return insecureRegistries[ref.Context().Registry.Name()]
}

func parseReference(s string, insecureRegistries map[string]bool, opts ...name.Option) (name.Reference, error) {
	ref, err := name.ParseReference(s, opts...)
	if err != nil {
		return nil, fmt.Errorf("parsing reference %q: %w", s, err)
	}

	if IsInsecure(ref, insecureRegistries) {
		ref, err = name.ParseReference(s, name.Insecure)
		if err != nil {
			logrus.Warnf("error getting insecure registry: %s\nremote references may not be retrieved", err.Error())
		}
	}

	return ref, nil
}

type digester interface {
	Digest() (v1.Hash, error)
}

func digest(d digester) (string, error) {
	h, err := d.Digest()
	if err != nil {
		return "", fmt.Errorf("getting digest: %w", err)
	}

	return h.String(), nil
}
