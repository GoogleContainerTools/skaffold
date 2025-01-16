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
