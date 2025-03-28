package testmocks

import (
	"github.com/buildpacks/imgutil"
	"github.com/buildpacks/imgutil/fakes"
)

type MockImage struct {
	*fakes.Image
	EntrypointCall struct {
		callCount int
		Received  struct{}
		Returns   struct {
			StringArr []string
			Error     error
		}
		Stub func() ([]string, error)
	}
}

func NewImage(name, topLayerSha string, identifier imgutil.Identifier) *MockImage {
	return &MockImage{
		Image: fakes.NewImage(name, topLayerSha, identifier),
	}
}

func (m *MockImage) Entrypoint() ([]string, error) {
	if m.EntrypointCall.Stub != nil {
		return m.EntrypointCall.Stub()
	}
	return m.EntrypointCall.Returns.StringArr, m.EntrypointCall.Returns.Error
}
