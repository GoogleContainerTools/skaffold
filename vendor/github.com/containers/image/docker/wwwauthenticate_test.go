package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// This is just a smoke test for the common expected header formats,
// by no means comprehensive.
func TestParseValueAndParams(t *testing.T) {
	for _, c := range []struct {
		input  string
		scope  string
		params map[string]string
	}{
		{
			`Bearer realm="https://auth.docker.io/token",service="registry.docker.io",scope="repository:library/busybox:pull"`,
			"bearer",
			map[string]string{
				"realm":   "https://auth.docker.io/token",
				"service": "registry.docker.io",
				"scope":   "repository:library/busybox:pull",
			},
		},
		{
			`Bearer realm="https://auth.docker.io/token",service="registry.docker.io",scope="repository:library/busybox:pull,push"`,
			"bearer",
			map[string]string{
				"realm":   "https://auth.docker.io/token",
				"service": "registry.docker.io",
				"scope":   "repository:library/busybox:pull,push",
			},
		},
		{
			`Bearer realm="http://127.0.0.1:5000/openshift/token"`,
			"bearer",
			map[string]string{"realm": "http://127.0.0.1:5000/openshift/token"},
		},
	} {
		scope, params := parseValueAndParams(c.input)
		assert.Equal(t, c.scope, scope, c.input)
		assert.Equal(t, c.params, params, c.input)
	}
}
