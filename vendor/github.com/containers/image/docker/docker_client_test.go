package docker

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/containers/image/pkg/docker/config"
	"github.com/containers/image/types"
	"github.com/containers/storage/pkg/homedir"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestDockerCertDir(t *testing.T) {
	const nondefaultFullPath = "/this/is/not/the/default/full/path"
	const nondefaultPerHostDir = "/this/is/not/the/default/certs.d"
	const variableReference = "$HOME"
	const rootPrefix = "/root/prefix"
	const registryHostPort = "thishostdefinitelydoesnotexist:5000"

	systemPerHostResult := filepath.Join(systemPerHostCertDirPaths[len(systemPerHostCertDirPaths)-1], registryHostPort)
	for _, c := range []struct {
		ctx      *types.SystemContext
		expected string
	}{
		// The common case
		{nil, systemPerHostResult},
		// There is a context, but it does not override the path.
		{&types.SystemContext{}, systemPerHostResult},
		// Full path overridden
		{&types.SystemContext{DockerCertPath: nondefaultFullPath}, nondefaultFullPath},
		// Per-host path overridden
		{
			&types.SystemContext{DockerPerHostCertDirPath: nondefaultPerHostDir},
			filepath.Join(nondefaultPerHostDir, registryHostPort),
		},
		// Both overridden
		{
			&types.SystemContext{
				DockerCertPath:           nondefaultFullPath,
				DockerPerHostCertDirPath: nondefaultPerHostDir,
			},
			nondefaultFullPath,
		},
		// Root overridden
		{
			&types.SystemContext{RootForImplicitAbsolutePaths: rootPrefix},
			filepath.Join(rootPrefix, systemPerHostResult),
		},
		// Root and path overrides present simultaneously,
		{
			&types.SystemContext{
				DockerCertPath:               nondefaultFullPath,
				RootForImplicitAbsolutePaths: rootPrefix,
			},
			nondefaultFullPath,
		},
		{
			&types.SystemContext{
				DockerPerHostCertDirPath:     nondefaultPerHostDir,
				RootForImplicitAbsolutePaths: rootPrefix,
			},
			filepath.Join(nondefaultPerHostDir, registryHostPort),
		},
		// … and everything at once
		{
			&types.SystemContext{
				DockerCertPath:               nondefaultFullPath,
				DockerPerHostCertDirPath:     nondefaultPerHostDir,
				RootForImplicitAbsolutePaths: rootPrefix,
			},
			nondefaultFullPath,
		},
		// No environment expansion happens in the overridden paths
		{&types.SystemContext{DockerCertPath: variableReference}, variableReference},
		{
			&types.SystemContext{DockerPerHostCertDirPath: variableReference},
			filepath.Join(variableReference, registryHostPort),
		},
	} {
		path, err := dockerCertDir(c.ctx, registryHostPort)
		require.Equal(t, nil, err)
		assert.Equal(t, c.expected, path)
	}
}

func TestGetAuth(t *testing.T) {
	origXDG := os.Getenv("XDG_RUNTIME_DIR")
	tmpDir1, err := ioutil.TempDir("", "test_docker_client_get_auth")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("using temporary XDG_RUNTIME_DIR directory: %q", tmpDir1)
	// override XDG_RUNTIME_DIR
	os.Setenv("XDG_RUNTIME_DIR", tmpDir1)
	defer func() {
		err := os.RemoveAll(tmpDir1)
		if err != nil {
			t.Logf("failed to cleanup temporary home directory %q: %v", tmpDir1, err)
		}
		os.Setenv("XDG_RUNTIME_DIR", origXDG)
	}()

	origHomeDir := homedir.Get()
	tmpDir2, err := ioutil.TempDir("", "test_docker_client_get_auth")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("using temporary home directory: %q", tmpDir2)
	//override homedir
	os.Setenv(homedir.Key(), tmpDir2)
	defer func() {
		err := os.RemoveAll(tmpDir2)
		if err != nil {
			t.Logf("failed to cleanup temporary home directory %q: %v", tmpDir2, err)
		}
		os.Setenv(homedir.Key(), origHomeDir)
	}()

	configDir1 := filepath.Join(tmpDir1, "containers")
	if err := os.MkdirAll(configDir1, 0700); err != nil {
		t.Fatal(err)
	}
	configDir2 := filepath.Join(tmpDir2, ".docker")
	if err := os.MkdirAll(configDir2, 0700); err != nil {
		t.Fatal(err)
	}
	configPaths := [2]string{filepath.Join(configDir1, "auth.json"), filepath.Join(configDir2, "config.json")}

	for _, configPath := range configPaths {
		for _, tc := range []struct {
			name             string
			hostname         string
			authConfig       testAuthConfig
			expectedUsername string
			expectedPassword string
			expectedError    error
			ctx              *types.SystemContext
		}{
			{
				name:       "empty hostname",
				authConfig: makeTestAuthConfig(testAuthConfigDataMap{"localhost:5000": testAuthConfigData{"bob", "password"}}),
			},
			{
				name:     "no auth config",
				hostname: "index.docker.io",
			},
			{
				name:             "match one",
				hostname:         "example.org",
				authConfig:       makeTestAuthConfig(testAuthConfigDataMap{"example.org": testAuthConfigData{"joe", "mypass"}}),
				expectedUsername: "joe",
				expectedPassword: "mypass",
			},
			{
				name:       "match none",
				hostname:   "registry.example.org",
				authConfig: makeTestAuthConfig(testAuthConfigDataMap{"example.org": testAuthConfigData{"joe", "mypass"}}),
			},
			{
				name:     "match docker.io",
				hostname: "docker.io",
				authConfig: makeTestAuthConfig(testAuthConfigDataMap{
					"example.org":     testAuthConfigData{"example", "org"},
					"index.docker.io": testAuthConfigData{"index", "docker.io"},
					"docker.io":       testAuthConfigData{"docker", "io"},
				}),
				expectedUsername: "docker",
				expectedPassword: "io",
			},
			{
				name:     "match docker.io normalized",
				hostname: "docker.io",
				authConfig: makeTestAuthConfig(testAuthConfigDataMap{
					"example.org":                testAuthConfigData{"bob", "pw"},
					"https://index.docker.io/v1": testAuthConfigData{"alice", "wp"},
				}),
				expectedUsername: "alice",
				expectedPassword: "wp",
			},
			{
				name:     "normalize registry",
				hostname: "https://docker.io/v1",
				authConfig: makeTestAuthConfig(testAuthConfigDataMap{
					"docker.io":      testAuthConfigData{"user", "pw"},
					"localhost:5000": testAuthConfigData{"joe", "pass"},
				}),
				expectedUsername: "user",
				expectedPassword: "pw",
			},
			{
				name:     "match localhost",
				hostname: "http://localhost",
				authConfig: makeTestAuthConfig(testAuthConfigDataMap{
					"docker.io":   testAuthConfigData{"user", "pw"},
					"localhost":   testAuthConfigData{"joe", "pass"},
					"example.com": testAuthConfigData{"alice", "pwd"},
				}),
				expectedUsername: "joe",
				expectedPassword: "pass",
			},
			{
				name:     "match ip",
				hostname: "10.10.3.56:5000",
				authConfig: makeTestAuthConfig(testAuthConfigDataMap{
					"10.10.30.45":     testAuthConfigData{"user", "pw"},
					"localhost":       testAuthConfigData{"joe", "pass"},
					"10.10.3.56":      testAuthConfigData{"alice", "pwd"},
					"10.10.3.56:5000": testAuthConfigData{"me", "mine"},
				}),
				expectedUsername: "me",
				expectedPassword: "mine",
			},
			{
				name:     "match port",
				hostname: "https://localhost:5000",
				authConfig: makeTestAuthConfig(testAuthConfigDataMap{
					"https://127.0.0.1:5000": testAuthConfigData{"user", "pw"},
					"http://localhost":       testAuthConfigData{"joe", "pass"},
					"https://localhost:5001": testAuthConfigData{"alice", "pwd"},
					"localhost:5000":         testAuthConfigData{"me", "mine"},
				}),
				expectedUsername: "me",
				expectedPassword: "mine",
			},
			{
				name:     "use system context",
				hostname: "example.org",
				authConfig: makeTestAuthConfig(testAuthConfigDataMap{
					"example.org": testAuthConfigData{"user", "pw"},
				}),
				expectedUsername: "foo",
				expectedPassword: "bar",
				ctx: &types.SystemContext{
					DockerAuthConfig: &types.DockerAuthConfig{
						Username: "foo",
						Password: "bar",
					},
				},
			},
		} {
			contents, err := json.MarshalIndent(&tc.authConfig, "", "  ")
			if err != nil {
				t.Errorf("[%s] failed to marshal authConfig: %v", tc.name, err)
				continue
			}
			if err := ioutil.WriteFile(configPath, contents, 0640); err != nil {
				t.Errorf("[%s] failed to write file %q: %v", tc.name, configPath, err)
				continue
			}

			var ctx *types.SystemContext
			if tc.ctx != nil {
				ctx = tc.ctx
			}
			username, password, err := config.GetAuthentication(ctx, tc.hostname)
			if err == nil && tc.expectedError != nil {
				t.Errorf("[%s] got unexpected non error and username=%q, password=%q", tc.name, username, password)
				continue
			}
			if err != nil && tc.expectedError == nil {
				t.Errorf("[%s] got unexpected error: %#+v", tc.name, err)
				continue
			}
			if !reflect.DeepEqual(err, tc.expectedError) {
				t.Errorf("[%s] got unexpected error: %#+v != %#+v", tc.name, err, tc.expectedError)
				continue
			}

			if username != tc.expectedUsername {
				t.Errorf("[%s] got unexpected user name: %q != %q", tc.name, username, tc.expectedUsername)
			}
			if password != tc.expectedPassword {
				t.Errorf("[%s] got unexpected user name: %q != %q", tc.name, password, tc.expectedPassword)
			}
		}
		os.RemoveAll(configPath)
	}
}

func TestGetAuthFromLegacyFile(t *testing.T) {
	origHomeDir := homedir.Get()
	tmpDir, err := ioutil.TempDir("", "test_docker_client_get_auth")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("using temporary home directory: %q", tmpDir)
	// override homedir
	os.Setenv(homedir.Key(), tmpDir)
	defer func() {
		err := os.RemoveAll(tmpDir)
		if err != nil {
			t.Logf("failed to cleanup temporary home directory %q: %v", tmpDir, err)
		}
		os.Setenv(homedir.Key(), origHomeDir)
	}()

	configPath := filepath.Join(tmpDir, ".dockercfg")

	for _, tc := range []struct {
		name             string
		hostname         string
		authConfig       testAuthConfig
		expectedUsername string
		expectedPassword string
		expectedError    error
	}{
		{
			name:     "normalize registry",
			hostname: "https://docker.io/v1",
			authConfig: makeTestAuthConfig(testAuthConfigDataMap{
				"docker.io":      testAuthConfigData{"user", "pw"},
				"localhost:5000": testAuthConfigData{"joe", "pass"},
			}),
			expectedUsername: "user",
			expectedPassword: "pw",
		},
		{
			name:     "ignore schema and path",
			hostname: "http://index.docker.io/v1",
			authConfig: makeTestAuthConfig(testAuthConfigDataMap{
				"docker.io/v2":         testAuthConfigData{"user", "pw"},
				"https://localhost/v1": testAuthConfigData{"joe", "pwd"},
			}),
			expectedUsername: "user",
			expectedPassword: "pw",
		},
	} {
		contents, err := json.MarshalIndent(&tc.authConfig.Auths, "", "  ")
		if err != nil {
			t.Errorf("[%s] failed to marshal authConfig: %v", tc.name, err)
			continue
		}
		if err := ioutil.WriteFile(configPath, contents, 0640); err != nil {
			t.Errorf("[%s] failed to write file %q: %v", tc.name, configPath, err)
			continue
		}

		username, password, err := config.GetAuthentication(nil, tc.hostname)
		if err == nil && tc.expectedError != nil {
			t.Errorf("[%s] got unexpected non error and username=%q, password=%q", tc.name, username, password)
			continue
		}
		if err != nil && tc.expectedError == nil {
			t.Errorf("[%s] got unexpected error: %#+v", tc.name, err)
			continue
		}
		if !reflect.DeepEqual(err, tc.expectedError) {
			t.Errorf("[%s] got unexpected error: %#+v != %#+v", tc.name, err, tc.expectedError)
			continue
		}

		if username != tc.expectedUsername {
			t.Errorf("[%s] got unexpected user name: %q != %q", tc.name, username, tc.expectedUsername)
		}
		if password != tc.expectedPassword {
			t.Errorf("[%s] got unexpected user name: %q != %q", tc.name, password, tc.expectedPassword)
		}
	}
}

func TestGetAuthPreferNewConfig(t *testing.T) {
	origHomeDir := homedir.Get()
	tmpDir, err := ioutil.TempDir("", "test_docker_client_get_auth")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("using temporary home directory: %q", tmpDir)
	// override homedir
	os.Setenv(homedir.Key(), tmpDir)
	defer func() {
		err := os.RemoveAll(tmpDir)
		if err != nil {
			t.Logf("failed to cleanup temporary home directory %q: %v", tmpDir, err)
		}
		os.Setenv(homedir.Key(), origHomeDir)
	}()

	configDir := filepath.Join(tmpDir, ".docker")
	if err := os.Mkdir(configDir, 0750); err != nil {
		t.Fatal(err)
	}

	for _, data := range []struct {
		path string
		ac   interface{}
	}{
		{
			filepath.Join(configDir, "config.json"),
			makeTestAuthConfig(testAuthConfigDataMap{
				"https://index.docker.io/v1/": testAuthConfigData{"alice", "pass"},
			}),
		},
		{
			filepath.Join(tmpDir, ".dockercfg"),
			makeTestAuthConfig(testAuthConfigDataMap{
				"https://index.docker.io/v1/": testAuthConfigData{"bob", "pw"},
			}).Auths,
		},
	} {
		contents, err := json.MarshalIndent(&data.ac, "", "  ")
		if err != nil {
			t.Fatalf("failed to marshal authConfig: %v", err)
		}
		if err := ioutil.WriteFile(data.path, contents, 0640); err != nil {
			t.Fatalf("failed to write file %q: %v", data.path, err)
		}
	}

	username, password, err := config.GetAuthentication(nil, "index.docker.io")
	if err != nil {
		t.Fatalf("got unexpected error: %#+v", err)
	}

	if username != "alice" {
		t.Fatalf("got unexpected user name: %q != %q", username, "alice")
	}
	if password != "pass" {
		t.Fatalf("got unexpected user name: %q != %q", password, "pass")
	}
}

func TestGetAuthFailsOnBadInput(t *testing.T) {
	origXDG := os.Getenv("XDG_RUNTIME_DIR")
	tmpDir1, err := ioutil.TempDir("", "test_docker_client_get_auth")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("using temporary XDG_RUNTIME_DIR directory: %q", tmpDir1)
	// override homedir
	os.Setenv("XDG_RUNTIME_DIR", tmpDir1)
	defer func() {
		err := os.RemoveAll(tmpDir1)
		if err != nil {
			t.Logf("failed to cleanup temporary home directory %q: %v", tmpDir1, err)
		}
		os.Setenv("XDG_RUNTIME_DIR", origXDG)
	}()

	origHomeDir := homedir.Get()
	tmpDir2, err := ioutil.TempDir("", "test_docker_client_get_auth")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("using temporary home directory: %q", tmpDir2)
	// override homedir
	os.Setenv(homedir.Key(), tmpDir2)
	defer func() {
		err := os.RemoveAll(tmpDir2)
		if err != nil {
			t.Logf("failed to cleanup temporary home directory %q: %v", tmpDir2, err)
		}
		os.Setenv(homedir.Key(), origHomeDir)
	}()

	configDir := filepath.Join(tmpDir1, "containers")
	if err := os.Mkdir(configDir, 0750); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "auth.json")

	// no config file present
	username, password, err := config.GetAuthentication(nil, "index.docker.io")
	if err != nil {
		t.Fatalf("got unexpected error: %#+v", err)
	}
	if len(username) > 0 || len(password) > 0 {
		t.Fatalf("got unexpected not empty username/password: %q/%q", username, password)
	}

	if err := ioutil.WriteFile(configPath, []byte("Json rocks! Unless it doesn't."), 0640); err != nil {
		t.Fatalf("failed to write file %q: %v", configPath, err)
	}
	username, password, err = config.GetAuthentication(nil, "index.docker.io")
	if err == nil {
		t.Fatalf("got unexpected non-error: username=%q, password=%q", username, password)
	}
	if _, ok := errors.Cause(err).(*json.SyntaxError); !ok {
		t.Fatalf("expected JSON syntax error, not: %#+v", err)
	}

	// remove the invalid config file
	os.RemoveAll(configPath)
	// no config file present
	username, password, err = config.GetAuthentication(nil, "index.docker.io")
	if err != nil {
		t.Fatalf("got unexpected error: %#+v", err)
	}
	if len(username) > 0 || len(password) > 0 {
		t.Fatalf("got unexpected not empty username/password: %q/%q", username, password)
	}

	configPath = filepath.Join(tmpDir2, ".dockercfg")
	if err := ioutil.WriteFile(configPath, []byte("I'm certainly not a json string."), 0640); err != nil {
		t.Fatalf("failed to write file %q: %v", configPath, err)
	}
	username, password, err = config.GetAuthentication(nil, "index.docker.io")
	if err == nil {
		t.Fatalf("got unexpected non-error: username=%q, password=%q", username, password)
	}
	if _, ok := errors.Cause(err).(*json.SyntaxError); !ok {
		t.Fatalf("expected JSON syntax error, not: %#+v", err)
	}
}

func TestNewBearerTokenFromJsonBlob(t *testing.T) {
	expected := &bearerToken{Token: "IAmAToken", ExpiresIn: 100, IssuedAt: time.Unix(1514800802, 0)}
	tokenBlob := []byte(`{"token":"IAmAToken","expires_in":100,"issued_at":"2018-01-01T10:00:02+00:00"}`)
	token, err := newBearerTokenFromJSONBlob(tokenBlob)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertBearerTokensEqual(t, expected, token)
}

func TestNewBearerAccessTokenFromJsonBlob(t *testing.T) {
	expected := &bearerToken{Token: "IAmAToken", ExpiresIn: 100, IssuedAt: time.Unix(1514800802, 0)}
	tokenBlob := []byte(`{"access_token":"IAmAToken","expires_in":100,"issued_at":"2018-01-01T10:00:02+00:00"}`)
	token, err := newBearerTokenFromJSONBlob(tokenBlob)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertBearerTokensEqual(t, expected, token)
}

func TestNewBearerTokenFromInvalidJsonBlob(t *testing.T) {
	tokenBlob := []byte("IAmNotJson")
	_, err := newBearerTokenFromJSONBlob(tokenBlob)
	if err == nil {
		t.Fatalf("unexpected an error unmarshalling JSON")
	}
}

func TestNewBearerTokenSmallExpiryFromJsonBlob(t *testing.T) {
	expected := &bearerToken{Token: "IAmAToken", ExpiresIn: 60, IssuedAt: time.Unix(1514800802, 0)}
	tokenBlob := []byte(`{"token":"IAmAToken","expires_in":1,"issued_at":"2018-01-01T10:00:02+00:00"}`)
	token, err := newBearerTokenFromJSONBlob(tokenBlob)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertBearerTokensEqual(t, expected, token)
}

func TestNewBearerTokenIssuedAtZeroFromJsonBlob(t *testing.T) {
	zeroTime := time.Time{}.Format(time.RFC3339)
	now := time.Now()
	tokenBlob := []byte(fmt.Sprintf(`{"token":"IAmAToken","expires_in":100,"issued_at":"%s"}`, zeroTime))
	token, err := newBearerTokenFromJSONBlob(tokenBlob)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if token.IssuedAt.Before(now) {
		t.Fatalf("expected [%s] not to be before [%s]", token.IssuedAt, now)
	}

}

type testAuthConfigData struct {
	username string
	password string
}

type testAuthConfigDataMap map[string]testAuthConfigData

type testAuthConfigEntry struct {
	Auth string `json:"auth,omitempty"`
}

type testAuthConfig struct {
	Auths map[string]testAuthConfigEntry `json:"auths"`
}

// encodeAuth creates an auth value from given authConfig data to be stored in auth config file.
// Inspired by github.com/docker/docker/cliconfig/config.go v1.10.3.
func encodeAuth(authConfig *testAuthConfigData) string {
	authStr := authConfig.username + ":" + authConfig.password
	msg := []byte(authStr)
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(msg)))
	base64.StdEncoding.Encode(encoded, msg)
	return string(encoded)
}

func makeTestAuthConfig(authConfigData map[string]testAuthConfigData) testAuthConfig {
	ac := testAuthConfig{
		Auths: make(map[string]testAuthConfigEntry),
	}
	for host, data := range authConfigData {
		ac.Auths[host] = testAuthConfigEntry{
			Auth: encodeAuth(&data),
		}
	}
	return ac
}

func assertBearerTokensEqual(t *testing.T, expected, subject *bearerToken) {
	if expected.Token != subject.Token {
		t.Fatalf("expected [%s] to equal [%s], it did not", subject.Token, expected.Token)
	}
	if expected.ExpiresIn != subject.ExpiresIn {
		t.Fatalf("expected [%d] to equal [%d], it did not", subject.ExpiresIn, expected.ExpiresIn)
	}
	if !expected.IssuedAt.Equal(subject.IssuedAt) {
		t.Fatalf("expected [%s] to equal [%s], it did not", subject.IssuedAt, expected.IssuedAt)
	}
}
