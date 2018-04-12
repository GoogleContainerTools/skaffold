package image

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/containers/image/types"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChooseDigestFromManifestList(t *testing.T) {
	manifest, err := ioutil.ReadFile(filepath.Join("fixtures", "schema2list.json"))
	require.NoError(t, err)

	// Match found
	for arch, expected := range map[string]digest.Digest{
		"amd64": "sha256:030fcb92e1487b18c974784dcc110a93147c9fc402188370fbfd17efabffc6af",
		"s390x": "sha256:e5aa1b0a24620228b75382997a0977f609b3ca3a95533dafdef84c74cc8df642",
		// There are several "arm" images with different variants;
		// the current code returns the first match. NOTE: This is NOT an API promise.
		"arm": "sha256:9142d97ef280a7953cf1a85716de49a24cc1dd62776352afad67e635331ff77a",
	} {
		digest, err := chooseDigestFromManifestList(&types.SystemContext{
			ArchitectureChoice: arch,
			OSChoice:           "linux",
		}, manifest)
		require.NoError(t, err, arch)
		assert.Equal(t, expected, digest)
	}

	// Invalid manifest list
	_, err = chooseDigestFromManifestList(&types.SystemContext{
		ArchitectureChoice: "amd64", OSChoice: "linux",
	}, bytes.Join([][]byte{manifest, []byte("!INVALID")}, nil))
	assert.Error(t, err)

	// Not found
	_, err = chooseDigestFromManifestList(&types.SystemContext{OSChoice: "Unmatched"}, manifest)
	assert.Error(t, err)
}
