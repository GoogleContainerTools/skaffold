/*
Copyright 2025 The Skaffold Authors

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

	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
)

// MockArtifactResolver mocks docker.ArtifactResolver interface.
type stubArtifactResolver struct {
	m map[string]string
}

// NewStubArtifactResolver returns a mock ArtifactResolver for testing.
func NewStubArtifactResolver(m map[string]string) *stubArtifactResolver {
	return &stubArtifactResolver{m}
}

func (r stubArtifactResolver) GetImageTag(imageName string) (string, bool) {
	val, found := r.m[imageName]
	return val, found
}

// simpleStubArtifactResolver is an implementation of docker.ArtifactResolver
// that returns the same value for any key
type simpleStubArtifactResolver struct{}

// GetImageTag is an implementation of docker.ArtifactResolver that
// always returns the same tag.
func (s *simpleStubArtifactResolver) GetImageTag(_ string) (string, bool) {
	return "image:latest", true
}

func NewSimpleStubArtifactResolver() ArtifactResolver {
	return &simpleStubArtifactResolver{}
}

// configStub is a mock implementation of the Config interface.
type configStub struct {
	runMode config.RunMode
	prune   bool
}

func (m configStub) GetKubeContext() string {
	return ""
}

func (m configStub) GlobalConfig() string {
	return ""
}

func (m configStub) MinikubeProfile() string {
	return ""
}

func (m configStub) GetInsecureRegistries() map[string]bool {
	return map[string]bool{}
}

func (m configStub) Mode() config.RunMode {
	return m.runMode
}

func (m configStub) Prune() bool {
	return m.prune
}

func (m configStub) ContainerDebugging() bool {
	return false
}

func NewConfigStub(mode config.RunMode, prune bool) Config {
	return &configStub{runMode: mode, prune: prune}
}

type fakeImageFetcher struct{}

func (f *fakeImageFetcher) fetch(_ context.Context, image string, _ Config) (*v1.ConfigFile, error) {
	switch image {
	case "ubuntu:14.04", "busybox", "nginx", "golang:1.9.2", "jboss/wildfly:14.0.1.Final", "gcr.io/distroless/base", "gcr.io/distroless/base:latest":
		return &v1.ConfigFile{}, nil
	case "golang:onbuild":
		return &v1.ConfigFile{
			Config: v1.Config{
				OnBuild: []string{
					"COPY . /onbuild",
				},
			},
		}, nil
	case "library/ruby:2.3.0":
		return nil, fmt.Errorf("retrieving image \"library/ruby:2.3.0\": unsupported MediaType: \"application/vnd.docker.distribution.manifest.v1+prettyjws\", see https://github.com/google/go-containerregistry/issues/377")
	}

	return nil, fmt.Errorf("no image found for %s", image)
}

type FetchImage func(context.Context, string, Config) (*v1.ConfigFile, error)

func NewFakeImageFetcher() FetchImage {
	fetcher := fakeImageFetcher{}
	return fetcher.fetch
}
