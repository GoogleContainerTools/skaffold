package docker

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/containers/image/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func dockerRefFromString(t *testing.T, s string) dockerReference {
	ref, err := ParseReference(s)
	require.NoError(t, err, s)
	dockerRef, ok := ref.(dockerReference)
	require.True(t, ok, s)
	return dockerRef
}

func TestConfiguredSignatureStorageBase(t *testing.T) {
	// Error reading configuration directory (/dev/null is not a directory)
	_, err := configuredSignatureStorageBase(&types.SystemContext{RegistriesDirPath: "/dev/null"},
		dockerRefFromString(t, "//busybox"), false)
	assert.Error(t, err)

	// No match found
	emptyDir, err := ioutil.TempDir("", "empty-dir")
	require.NoError(t, err)
	defer os.RemoveAll(emptyDir)
	base, err := configuredSignatureStorageBase(&types.SystemContext{RegistriesDirPath: emptyDir},
		dockerRefFromString(t, "//this/is/not/in/the:configuration"), false)
	assert.NoError(t, err)
	assert.Nil(t, base)

	// Invalid URL
	_, err = configuredSignatureStorageBase(&types.SystemContext{RegistriesDirPath: "fixtures/registries.d"},
		dockerRefFromString(t, "//localhost/invalid/url/test"), false)
	assert.Error(t, err)

	// Success
	base, err = configuredSignatureStorageBase(&types.SystemContext{RegistriesDirPath: "fixtures/registries.d"},
		dockerRefFromString(t, "//example.com/my/project"), false)
	assert.NoError(t, err)
	require.NotNil(t, base)
	assert.Equal(t, "https://sigstore.example.com/my/project", (*url.URL)(base).String())
}

func TestRegistriesDirPath(t *testing.T) {
	const nondefaultPath = "/this/is/not/the/default/registries.d"
	const variableReference = "$HOME"
	const rootPrefix = "/root/prefix"

	for _, c := range []struct {
		ctx      *types.SystemContext
		expected string
	}{
		// The common case
		{nil, systemRegistriesDirPath},
		// There is a context, but it does not override the path.
		{&types.SystemContext{}, systemRegistriesDirPath},
		// Path overridden
		{&types.SystemContext{RegistriesDirPath: nondefaultPath}, nondefaultPath},
		// Root overridden
		{
			&types.SystemContext{RootForImplicitAbsolutePaths: rootPrefix},
			filepath.Join(rootPrefix, systemRegistriesDirPath),
		},
		// Root and path overrides present simultaneously,
		{
			&types.SystemContext{
				RootForImplicitAbsolutePaths: rootPrefix,
				RegistriesDirPath:            nondefaultPath,
			},
			nondefaultPath,
		},
		// No environment expansion happens in the overridden paths
		{&types.SystemContext{RegistriesDirPath: variableReference}, variableReference},
	} {
		path := registriesDirPath(c.ctx)
		assert.Equal(t, c.expected, path)
	}
}

func TestLoadAndMergeConfig(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "merge-config")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// No registries.d exists
	config, err := loadAndMergeConfig(filepath.Join(tmpDir, "thisdoesnotexist"))
	require.NoError(t, err)
	assert.Equal(t, &registryConfiguration{Docker: map[string]registryNamespace{}}, config)

	// Empty registries.d directory
	emptyDir := filepath.Join(tmpDir, "empty")
	err = os.Mkdir(emptyDir, 0755)
	require.NoError(t, err)
	config, err = loadAndMergeConfig(emptyDir)
	require.NoError(t, err)
	assert.Equal(t, &registryConfiguration{Docker: map[string]registryNamespace{}}, config)

	// Unreadable registries.d directory
	unreadableDir := filepath.Join(tmpDir, "unreadable")
	err = os.Mkdir(unreadableDir, 0000)
	require.NoError(t, err)
	config, err = loadAndMergeConfig(unreadableDir)
	assert.Error(t, err)

	// An unreadable file in a registries.d directory
	unreadableFileDir := filepath.Join(tmpDir, "unreadableFile")
	err = os.Mkdir(unreadableFileDir, 0755)
	require.NoError(t, err)
	err = ioutil.WriteFile(filepath.Join(unreadableFileDir, "0.yaml"), []byte("{}"), 0644)
	require.NoError(t, err)
	err = ioutil.WriteFile(filepath.Join(unreadableFileDir, "1.yaml"), nil, 0000)
	require.NoError(t, err)
	config, err = loadAndMergeConfig(unreadableFileDir)
	assert.Error(t, err)

	// Invalid YAML
	invalidYAMLDir := filepath.Join(tmpDir, "invalidYAML")
	err = os.Mkdir(invalidYAMLDir, 0755)
	require.NoError(t, err)
	err = ioutil.WriteFile(filepath.Join(invalidYAMLDir, "0.yaml"), []byte("}"), 0644)
	require.NoError(t, err)
	config, err = loadAndMergeConfig(invalidYAMLDir)
	assert.Error(t, err)

	// Duplicate DefaultDocker
	duplicateDefault := filepath.Join(tmpDir, "duplicateDefault")
	err = os.Mkdir(duplicateDefault, 0755)
	require.NoError(t, err)
	err = ioutil.WriteFile(filepath.Join(duplicateDefault, "0.yaml"),
		[]byte("default-docker:\n sigstore: file:////tmp/something"), 0644)
	require.NoError(t, err)
	err = ioutil.WriteFile(filepath.Join(duplicateDefault, "1.yaml"),
		[]byte("default-docker:\n sigstore: file:////tmp/different"), 0644)
	require.NoError(t, err)
	config, err = loadAndMergeConfig(duplicateDefault)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "0.yaml")
	assert.Contains(t, err.Error(), "1.yaml")

	// Duplicate DefaultDocker
	duplicateNS := filepath.Join(tmpDir, "duplicateNS")
	err = os.Mkdir(duplicateNS, 0755)
	require.NoError(t, err)
	err = ioutil.WriteFile(filepath.Join(duplicateNS, "0.yaml"),
		[]byte("docker:\n example.com:\n  sigstore: file:////tmp/something"), 0644)
	require.NoError(t, err)
	err = ioutil.WriteFile(filepath.Join(duplicateNS, "1.yaml"),
		[]byte("docker:\n example.com:\n  sigstore: file:////tmp/different"), 0644)
	require.NoError(t, err)
	config, err = loadAndMergeConfig(duplicateNS)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "0.yaml")
	assert.Contains(t, err.Error(), "1.yaml")

	// A fully worked example, including an empty-dictionary file and a non-.yaml file
	config, err = loadAndMergeConfig("fixtures/registries.d")
	require.NoError(t, err)
	assert.Equal(t, &registryConfiguration{
		DefaultDocker: &registryNamespace{SigStore: "file:///mnt/companywide/signatures/for/other/repositories"},
		Docker: map[string]registryNamespace{
			"example.com":                    {SigStore: "https://sigstore.example.com"},
			"registry.test.example.com":      {SigStore: "http://registry.test.example.com/sigstore"},
			"registry.test.example.com:8888": {SigStore: "http://registry.test.example.com:8889/sigstore", SigStoreStaging: "https://registry.test.example.com:8889/sigstore/specialAPIserverWhichDoesntExist"},
			"localhost":                      {SigStore: "file:///home/mitr/mydevelopment1"},
			"localhost:8080":                 {SigStore: "file:///home/mitr/mydevelopment2"},
			"localhost/invalid/url/test":     {SigStore: ":emptyscheme"},
			"docker.io/contoso":              {SigStore: "https://sigstore.contoso.com/fordocker"},
			"docker.io/centos":               {SigStore: "https://sigstore.centos.org/"},
			"docker.io/centos/mybetaprooduct": {
				SigStore:        "http://localhost:9999/mybetaWIP/sigstore",
				SigStoreStaging: "file:///srv/mybetaWIP/sigstore",
			},
			"docker.io/centos/mybetaproduct:latest": {SigStore: "https://sigstore.centos.org/"},
		},
	}, config)
}

func TestRegistryConfigurationSignaureTopLevel(t *testing.T) {
	config := registryConfiguration{
		DefaultDocker: &registryNamespace{SigStore: "=default", SigStoreStaging: "=default+w"},
		Docker:        map[string]registryNamespace{},
	}
	for _, ns := range []string{
		"localhost",
		"localhost:5000",
		"example.com",
		"example.com/ns1",
		"example.com/ns1/ns2",
		"example.com/ns1/ns2/repo",
		"example.com/ns1/ns2/repo:notlatest",
	} {
		config.Docker[ns] = registryNamespace{SigStore: ns, SigStoreStaging: ns + "+w"}
	}

	for _, c := range []struct{ input, expected string }{
		{"example.com/ns1/ns2/repo:notlatest", "example.com/ns1/ns2/repo:notlatest"},
		{"example.com/ns1/ns2/repo:unmatched", "example.com/ns1/ns2/repo"},
		{"example.com/ns1/ns2/notrepo:notlatest", "example.com/ns1/ns2"},
		{"example.com/ns1/notns2/repo:notlatest", "example.com/ns1"},
		{"example.com/notns1/ns2/repo:notlatest", "example.com"},
		{"unknown.example.com/busybox", "=default"},
		{"localhost:5000/busybox", "localhost:5000"},
		{"localhost/busybox", "localhost"},
		{"localhost:9999/busybox", "=default"},
	} {
		dr := dockerRefFromString(t, "//"+c.input)

		res := config.signatureTopLevel(dr, false)
		assert.Equal(t, c.expected, res, c.input)
		res = config.signatureTopLevel(dr, true) // test that forWriting is correctly propagated
		assert.Equal(t, c.expected+"+w", res, c.input)
	}

	config = registryConfiguration{
		Docker: map[string]registryNamespace{
			"unmatched": {SigStore: "a", SigStoreStaging: "b"},
		},
	}
	dr := dockerRefFromString(t, "//thisisnotmatched")
	res := config.signatureTopLevel(dr, false)
	assert.Equal(t, "", res)
	res = config.signatureTopLevel(dr, true)
	assert.Equal(t, "", res)
}

func TestRegistryNamespaceSignatureTopLevel(t *testing.T) {
	for _, c := range []struct {
		ns         registryNamespace
		forWriting bool
		expected   string
	}{
		{registryNamespace{SigStoreStaging: "a", SigStore: "b"}, true, "a"},
		{registryNamespace{SigStoreStaging: "a", SigStore: "b"}, false, "b"},
		{registryNamespace{SigStore: "b"}, true, "b"},
		{registryNamespace{SigStore: "b"}, false, "b"},
		{registryNamespace{SigStoreStaging: "a"}, true, "a"},
		{registryNamespace{SigStoreStaging: "a"}, false, ""},
		{registryNamespace{}, true, ""},
		{registryNamespace{}, false, ""},
	} {
		res := c.ns.signatureTopLevel(c.forWriting)
		assert.Equal(t, c.expected, res, fmt.Sprintf("%#v %v", c.ns, c.forWriting))
	}
}

func TestSignatureStorageBaseSignatureStorageURL(t *testing.T) {
	const mdInput = "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	const mdMapped = "sha256=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	assert.True(t, signatureStorageURL(nil, mdInput, 0) == nil)
	for _, c := range []struct {
		base     string
		index    int
		expected string
	}{
		{"file:///tmp", 0, "file:///tmp@" + mdMapped + "/signature-1"},
		{"file:///tmp", 1, "file:///tmp@" + mdMapped + "/signature-2"},
		{"https://localhost:5555/root", 0, "https://localhost:5555/root@" + mdMapped + "/signature-1"},
		{"https://localhost:5555/root", 1, "https://localhost:5555/root@" + mdMapped + "/signature-2"},
		{"http://localhost:5555/root", 0, "http://localhost:5555/root@" + mdMapped + "/signature-1"},
		{"http://localhost:5555/root", 1, "http://localhost:5555/root@" + mdMapped + "/signature-2"},
	} {
		url, err := url.Parse(c.base)
		require.NoError(t, err)
		expectedURL, err := url.Parse(c.expected)
		require.NoError(t, err)
		res := signatureStorageURL(url, mdInput, c.index)
		assert.Equal(t, expectedURL, res, c.expected)
	}
}
