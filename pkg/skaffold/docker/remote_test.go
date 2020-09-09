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
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestIsInsecure(t *testing.T) {
	tests := []struct {
		description        string
		image              string
		insecureRegistries map[string]bool
		result             bool
	}{
		{"nil registries", "localhost:5000/img", nil, false},
		{"unlisted registry", "other.tld/img", map[string]bool{"registry.tld": true}, false},
		{"listed insecure", "registry.tld/img", map[string]bool{"registry.tld": true}, true},
		{"listed secure", "registry.tld/img", map[string]bool{"registry.tld": false}, false},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			ref, err := name.ParseReference(test.image)
			t.CheckNoError(err)

			result := IsInsecure(ref, test.insecureRegistries)

			t.CheckDeepEqual(test.result, result)
		})
	}
}

func TestRemoteImage(t *testing.T) {
	tests := []struct {
		description        string
		image              string
		insecureRegistries map[string]bool
		expectedScheme     string
		shouldErr          bool
	}{
		{
			description:    "secure",
			image:          "gcr.io/secure/image",
			expectedScheme: "https",
		},
		{
			description: "insecure",
			image:       "my.insecure.registry/image",
			insecureRegistries: map[string]bool{
				"my.insecure.registry": true,
			},
			expectedScheme: "http",
		},
		{
			description: "invalid",
			image:       "invalid image",
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&remoteImage, func(ref name.Reference, options ...remote.Option) (v1.Image, error) {
				return &fakeImage{
					Reference: ref,
				}, nil
			})

			img, err := getRemoteImage(test.image, test.insecureRegistries)

			t.CheckError(test.shouldErr, err)
			if !test.shouldErr {
				t.CheckDeepEqual(test.expectedScheme, img.(*fakeImage).Reference.Context().Registry.Scheme())
			}
		})
	}
}

func TestRemoteDigest(t *testing.T) {
	tests := []struct {
		description    string
		image          string
		hash           v1.Hash
		shouldErr      bool
		expectedDigest string
	}{
		{
			description:    "OCI v1 image",
			image:          "image",
			expectedDigest: "sha256:abacab",
		},
		{
			description:    "OCI image index",
			image:          "index",
			expectedDigest: "sha256:cdefcdef",
		},
		{
			description: "image not found",
			image:       "notfound",
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&remoteIndex, func(ref name.Reference, options ...remote.Option) (v1.ImageIndex, error) {
				if ref.Name() != "index.docker.io/library/index:latest" {
					return nil, fmt.Errorf("not found: %s", ref.Name())
				}

				return &fakeImageIndex{
					Hash: v1.Hash{Algorithm: "sha256", Hex: "cdefcdef"},
				}, nil
			})
			t.Override(&remoteImage, func(ref name.Reference, options ...remote.Option) (v1.Image, error) {
				if ref.Name() != "index.docker.io/library/image:latest" {
					return nil, fmt.Errorf("not found: %s", ref.Name())
				}

				return &fakeImage{
					Hash: v1.Hash{Algorithm: "sha256", Hex: "abacab"},
				}, nil
			})

			digest, err := RemoteDigest(test.image, nil)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedDigest, digest)
		})
	}
}

type fakeImage struct {
	v1.Image
	Reference name.Reference
	Hash      v1.Hash
}

func (i *fakeImage) Digest() (v1.Hash, error) {
	return i.Hash, nil
}

type fakeImageIndex struct {
	Hash v1.Hash
}

func (i *fakeImageIndex) Digest() (v1.Hash, error) {
	return i.Hash, nil
}

func (i *fakeImageIndex) MediaType() (types.MediaType, error)       { return "", nil }
func (i *fakeImageIndex) Size() (int64, error)                      { return 0, nil }
func (i *fakeImageIndex) IndexManifest() (*v1.IndexManifest, error) { return nil, nil }
func (i *fakeImageIndex) RawManifest() ([]byte, error)              { return nil, nil }
func (i *fakeImageIndex) Image(v1.Hash) (v1.Image, error)           { return nil, nil }
func (i *fakeImageIndex) ImageIndex(v1.Hash) (v1.ImageIndex, error) { return nil, nil }
