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

// MockArtifactResolver mocks docker.ArtifactResolver interface.
type mockArtifactResolver struct {
	m map[string]string
}

// NewMockArtifactResolver returns a mock ArtifactResolver for testing.
func NewMockArtifactResolver(m map[string]string) *mockArtifactResolver {
	return &mockArtifactResolver{m}
}

func (r mockArtifactResolver) GetImageTag(imageName string) (string, bool) {
	val, found := r.m[imageName]
	return val, found
}

// simpleMockArtifactResolver is an implementation of docker.ArtifactResolver
// that returns the same value for any key
type simpleMockArtifactResolver struct{}

// GetImageTag is an implementation of docker.ArtifactResolver that
// always returns the same tag.
func (s *simpleMockArtifactResolver) GetImageTag(_ string) (string, bool) {
	return "image:latest", true
}

func NewSimpleMockArtifactResolver() ArtifactResolver {
	return &simpleMockArtifactResolver{}
}
