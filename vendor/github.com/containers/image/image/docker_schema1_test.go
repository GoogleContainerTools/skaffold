package image

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func manifestSchema1FromFixture(t *testing.T, fixture string) genericManifest {
	manifest, err := ioutil.ReadFile(filepath.Join("fixtures", fixture))
	require.NoError(t, err)

	m, err := manifestSchema1FromManifest(manifest)
	require.NoError(t, err)
	return m
}

func TestManifestSchema1ToOCIConfig(t *testing.T) {
	m := manifestSchema1FromFixture(t, "schema1-to-oci-config.json")
	configOCI, err := m.OCIConfig()
	require.NoError(t, err)
	assert.Equal(t, "/pause", configOCI.Config.Entrypoint[0])
}
