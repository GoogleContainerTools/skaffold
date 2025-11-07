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
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	specs "github.com/opencontainers/image-spec/specs-go/v1"

	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

// for testing
var (
	RemoteDigest = getRemoteDigest
	remoteImage  = remote.Image
	remoteIndex  = remote.Index
)

func AddRemoteTag(src, target string, cfg Config, platforms []specs.Platform) error {
	log.Entry(context.TODO()).Debugf("attempting to add tag %s to src %s", target, src)

	targetRef, err := parseReference(target, cfg, name.WeakValidation)
	if err != nil {
		return err
	}

	if len(platforms) > 1 {
		index, err := getRemoteIndex(src, cfg)
		if err != nil {
			return fmt.Errorf("getting image index: %w", err)
		}
		return remote.WriteIndex(targetRef, index, remote.WithAuthFromKeychain(primaryKeychain))
	}

	var pl v1.Platform
	if len(platforms) == 1 {
		pl = util.ConvertToV1Platform(platforms[0])
	}
	img, err := getRemoteImage(src, cfg, pl)
	if err != nil {
		return fmt.Errorf("getting image: %w", err)
	}

	return remote.Write(targetRef, img, remote.WithAuthFromKeychain(primaryKeychain))
}

func getRemoteDigest(identifier string, cfg Config, platforms []specs.Platform) (string, error) {
	idx, err := getRemoteIndex(identifier, cfg)
	if err == nil {
		return digest(idx)
	}
	if len(platforms) > 1 {
		return "", fmt.Errorf("cannot fetch remote index for multiple platform image %q: %w", identifier, err)
	}
	var pl v1.Platform
	if len(platforms) == 1 {
		pl = util.ConvertToV1Platform(platforms[0])
	}
	img, err := getRemoteImage(identifier, cfg, pl)
	if err != nil {
		return "", fmt.Errorf("getting image: %w", err)
	}

	return digest(img)
}

// RetrieveRemoteConfig retrieves the remote config file for an image
func RetrieveRemoteConfig(identifier string, cfg Config, platform v1.Platform) (*v1.ConfigFile, error) {
	img, err := getRemoteImage(identifier, cfg, platform)
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

func getRemoteImage(identifier string, cfg Config, platform v1.Platform) (v1.Image, error) {
	ref, err := parseReference(identifier, cfg)
	if err != nil {
		return nil, err
	}
	options := []remote.Option{
		remote.WithAuthFromKeychain(primaryKeychain),
	}
	if IsInsecure(ref, cfg.GetInsecureRegistries()) {
		options = append(options, insecureTransportOption())
	}
	if platform.String() != "" {
		options = append(options, remote.WithPlatform(platform))
	}

	return remoteImage(ref, options...)
}

func getRemoteIndex(identifier string, cfg Config) (v1.ImageIndex, error) {
	ref, err := parseReference(identifier, cfg)
	if err != nil {
		return nil, err
	}

	options := []remote.Option{
		remote.WithAuthFromKeychain(primaryKeychain),
	}
	if IsInsecure(ref, cfg.GetInsecureRegistries()) {
		options = append(options, insecureTransportOption())
	}
	return remoteIndex(ref, options...)
}

// IsInsecure tests if an image is pulled from an insecure registry; default is false
func IsInsecure(ref name.Reference, insecureRegistries map[string]bool) bool {
	return insecureRegistries[ref.Context().Registry.Name()]
}

// insecureTransportOption allows untrusted certificates.
func insecureTransportOption() remote.Option {
	transport := remote.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true, //nolint: gosec
	}
	return remote.WithTransport(transport)
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
