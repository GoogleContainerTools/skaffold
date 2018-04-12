package transports

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStubTransport(t *testing.T) {
	const name = "whatever"

	s := NewStubTransport(name)
	assert.Equal(t, name, s.Name())
	_, err := s.ParseReference("this is rejected regardless of content")
	assert.Error(t, err)
	err = s.ValidatePolicyConfigurationScope("this is accepted regardless of content")
	assert.NoError(t, err)
}
