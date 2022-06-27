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
	"context"
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	specs "github.com/opencontainers/image-spec/specs-go/v1"

	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// for testing
var (
	RemoteDigest = getRemoteDigest
	remoteImage  = remote.Image
	remoteIndex  = remote.Index
)

func AddRemoteTag(src, target string, cfg Config, platforms []specs.Platform) error {
	log.Entry(context.TODO()).Debugf("attempting to add tag %s to src %s", target, src)
	img, err := getRemoteImage(src, cfg, platforms)
	if err != nil {
		return fmt.Errorf("getting image: %w", err)
	}

	targetRef, err := parseReference(target, cfg, name.WeakValidation)
	if err != nil {
		return err
	}

	return remote.Write(targetRef, img, remote.WithAuthFromKeychain(primaryKeychain))
}

func getRemoteDigest(identifier string, cfg Config, platforms []specs.Platform) (string, error) {
	idx, err := getRemoteIndex(identifier, cfg)
	if err == nil {
		return digest(idx)
	}

	img, err := getRemoteImage(identifier, cfg, platforms)
	if err != nil {
		return "", fmt.Errorf("getting image: %w", err)
	}

	return digest(img)
}

// RetrieveRemoteConfig retrieves the remote config file for an image
func RetrieveRemoteConfig(identifier string, cfg Config, platforms []specs.Platform) (*v1.ConfigFile, error) {
	img, err := getRemoteImage(identifier, cfg, platforms)
	if err != nil {
		return nil, err
	}

	return img.ConfigFile()
}

// Push pushes the tarball image
func Push(tarPath, tag string, cfg Config, platforms []specs.Platform) (string, error) {
	t, err := name.NewTag(tag, name.WeakValidation)
	if err != nil {
		return "", fmt.Errorf("parsing tag %q: %w", tag, err)
	}

	i, err := tarball.ImageFromPath(tarPath, nil)
	if err != nil {
		return "", fmt.Errorf("reading image %q: %w", tarPath, err)
	}

	if err := remote.Write(t, i, remote.WithAuthFromKeychain(primaryKeychain)); err != nil {
		return "", fmt.Errorf("%s %q: %w", sErrors.PushImageErr, t, err)
	}

	return getRemoteDigest(tag, cfg, platforms)
}

func getRemoteImage(identifier string, cfg Config, platforms []specs.Platform) (v1.Image, error) {
	// for multi-arch images, we can only get the image for one platform at a time.
	// If the platform is not specified then it'll fetch the image for the host architecture.
	// To avoid discrepencies, we fail image fetch for multi-platform images. These should instead be accessed with the `remoteIndex` method.
	if len(platforms) > 1 {
		return nil, fmt.Errorf("cannot fetch remote image %q for multiple platforms %q", identifier, platforms)
	}
	ref, err := parseReference(identifier, cfg)
	if err != nil {
		return nil, err
	}
	options := []remote.Option{
		remote.WithAuthFromKeychain(primaryKeychain),
	}
	if len(platforms) == 1 {
		options = append(options, remote.WithPlatform(util.ConvertToV1Platform(platforms[0])))
	}

	return remoteImage(ref, options...)
}

func getRemoteIndex(identifier string, cfg Config) (v1.ImageIndex, error) {
	ref, err := parseReference(identifier, cfg)
	if err != nil {
		return nil, err
	}

	return remoteIndex(ref, remote.WithAuthFromKeychain(primaryKeychain))
}

// IsInsecure tests if an image is pulled from an insecure registry; default is false
func IsInsecure(ref name.Reference, insecureRegistries map[string]bool) bool {
	return insecureRegistries[ref.Context().Registry.Name()]
}

func parseReference(s string, cfg Config, opts ...name.Option) (name.Reference, error) {
	ref, err := name.ParseReference(s, opts...)
	if err != nil {
		return nil, fmt.Errorf("parsing reference %q: %w", s, err)
	}

	if IsInsecure(ref, cfg.GetInsecureRegistries()) {
		ref, err = name.ParseReference(s, name.Insecure)
		if err != nil {
			log.Entry(context.TODO()).Warnf("error getting insecure registry: %s\nremote references may not be retrieved", err.Error())
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
		return "", remoteDigestGetErr(err)
	}

	return h.String(), nil
}
