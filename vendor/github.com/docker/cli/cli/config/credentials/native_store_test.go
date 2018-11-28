package credentials

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/docker/docker-credential-helpers/client"
	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

const (
	validServerAddress   = "https://index.docker.io/v1"
	validServerAddress2  = "https://example.com:5002"
	invalidServerAddress = "https://foobar.example.com"
	missingCredsAddress  = "https://missing.docker.io/v1"
)

var errCommandExited = errors.Errorf("exited 1")

// mockCommand simulates interactions between the docker client and a remote
// credentials helper.
// Unit tests inject this mocked command into the remote to control execution.
type mockCommand struct {
	arg   string
	input io.Reader
}

// Output returns responses from the remote credentials helper.
// It mocks those responses based in the input in the mock.
func (m *mockCommand) Output() ([]byte, error) {
	in, err := ioutil.ReadAll(m.input)
	if err != nil {
		return nil, err
	}
	inS := string(in)

	switch m.arg {
	case "erase":
		switch inS {
		case validServerAddress:
			return nil, nil
		default:
			return []byte("program failed"), errCommandExited
		}
	case "get":
		switch inS {
		case validServerAddress:
			return []byte(`{"Username": "foo", "Secret": "bar"}`), nil
		case validServerAddress2:
			return []byte(`{"Username": "<token>", "Secret": "abcd1234"}`), nil
		case missingCredsAddress:
			return []byte(credentials.NewErrCredentialsNotFound().Error()), errCommandExited
		case invalidServerAddress:
			return []byte("program failed"), errCommandExited
		}
	case "store":
		var c credentials.Credentials
		err := json.NewDecoder(strings.NewReader(inS)).Decode(&c)
		if err != nil {
			return []byte("program failed"), errCommandExited
		}
		switch c.ServerURL {
		case validServerAddress:
			return nil, nil
		default:
			return []byte("program failed"), errCommandExited
		}
	case "list":
		return []byte(fmt.Sprintf(`{"%s": "%s", "%s": "%s"}`, validServerAddress, "foo", validServerAddress2, "<token>")), nil
	}

	return []byte(fmt.Sprintf("unknown argument %q with %q", m.arg, inS)), errCommandExited
}

// Input sets the input to send to a remote credentials helper.
func (m *mockCommand) Input(in io.Reader) {
	m.input = in
}

func mockCommandFn(args ...string) client.Program {
	return &mockCommand{
		arg: args[0],
	}
}

func TestNativeStoreAddCredentials(t *testing.T) {
	f := newStore(make(map[string]types.AuthConfig))
	s := &nativeStore{
		programFunc: mockCommandFn,
		fileStore:   NewFileStore(f),
	}
	auth := types.AuthConfig{
		Username:      "foo",
		Password:      "bar",
		Email:         "foo@example.com",
		ServerAddress: validServerAddress,
	}
	err := s.Store(auth)
	assert.NilError(t, err)
	assert.Check(t, is.Len(f.GetAuthConfigs(), 1))

	actual, ok := f.GetAuthConfigs()[validServerAddress]
	assert.Check(t, ok)
	expected := types.AuthConfig{
		Email:         auth.Email,
		ServerAddress: auth.ServerAddress,
	}
	assert.Check(t, is.DeepEqual(expected, actual))
}

func TestNativeStoreAddInvalidCredentials(t *testing.T) {
	f := newStore(make(map[string]types.AuthConfig))
	s := &nativeStore{
		programFunc: mockCommandFn,
		fileStore:   NewFileStore(f),
	}
	err := s.Store(types.AuthConfig{
		Username:      "foo",
		Password:      "bar",
		Email:         "foo@example.com",
		ServerAddress: invalidServerAddress,
	})
	assert.ErrorContains(t, err, "program failed")
	assert.Check(t, is.Len(f.GetAuthConfigs(), 0))
}

func TestNativeStoreGet(t *testing.T) {
	f := newStore(map[string]types.AuthConfig{
		validServerAddress: {
			Email: "foo@example.com",
		},
	})
	s := &nativeStore{
		programFunc: mockCommandFn,
		fileStore:   NewFileStore(f),
	}
	actual, err := s.Get(validServerAddress)
	assert.NilError(t, err)

	expected := types.AuthConfig{
		Username: "foo",
		Password: "bar",
		Email:    "foo@example.com",
	}
	assert.Check(t, is.DeepEqual(expected, actual))
}

func TestNativeStoreGetIdentityToken(t *testing.T) {
	f := newStore(map[string]types.AuthConfig{
		validServerAddress2: {
			Email: "foo@example2.com",
		},
	})

	s := &nativeStore{
		programFunc: mockCommandFn,
		fileStore:   NewFileStore(f),
	}
	actual, err := s.Get(validServerAddress2)
	assert.NilError(t, err)

	expected := types.AuthConfig{
		IdentityToken: "abcd1234",
		Email:         "foo@example2.com",
	}
	assert.Check(t, is.DeepEqual(expected, actual))
}

func TestNativeStoreGetAll(t *testing.T) {
	f := newStore(map[string]types.AuthConfig{
		validServerAddress: {
			Email: "foo@example.com",
		},
	})

	s := &nativeStore{
		programFunc: mockCommandFn,
		fileStore:   NewFileStore(f),
	}
	as, err := s.GetAll()
	assert.NilError(t, err)
	assert.Check(t, is.Len(as, 2))

	if as[validServerAddress].Username != "foo" {
		t.Fatalf("expected username `foo` for %s, got %s", validServerAddress, as[validServerAddress].Username)
	}
	if as[validServerAddress].Password != "bar" {
		t.Fatalf("expected password `bar` for %s, got %s", validServerAddress, as[validServerAddress].Password)
	}
	if as[validServerAddress].IdentityToken != "" {
		t.Fatalf("expected identity to be empty for %s, got %s", validServerAddress, as[validServerAddress].IdentityToken)
	}
	if as[validServerAddress].Email != "foo@example.com" {
		t.Fatalf("expected email `foo@example.com` for %s, got %s", validServerAddress, as[validServerAddress].Email)
	}
	if as[validServerAddress2].Username != "" {
		t.Fatalf("expected username to be empty for %s, got %s", validServerAddress2, as[validServerAddress2].Username)
	}
	if as[validServerAddress2].Password != "" {
		t.Fatalf("expected password to be empty for %s, got %s", validServerAddress2, as[validServerAddress2].Password)
	}
	if as[validServerAddress2].IdentityToken != "abcd1234" {
		t.Fatalf("expected identity token `abcd1324` for %s, got %s", validServerAddress2, as[validServerAddress2].IdentityToken)
	}
	if as[validServerAddress2].Email != "" {
		t.Fatalf("expected no email for %s, got %s", validServerAddress2, as[validServerAddress2].Email)
	}
}

func TestNativeStoreGetMissingCredentials(t *testing.T) {
	f := newStore(map[string]types.AuthConfig{
		validServerAddress: {
			Email: "foo@example.com",
		},
	})

	s := &nativeStore{
		programFunc: mockCommandFn,
		fileStore:   NewFileStore(f),
	}
	_, err := s.Get(missingCredsAddress)
	assert.NilError(t, err)
}

func TestNativeStoreGetInvalidAddress(t *testing.T) {
	f := newStore(map[string]types.AuthConfig{
		validServerAddress: {
			Email: "foo@example.com",
		},
	})

	s := &nativeStore{
		programFunc: mockCommandFn,
		fileStore:   NewFileStore(f),
	}
	_, err := s.Get(invalidServerAddress)
	assert.ErrorContains(t, err, "program failed")
}

func TestNativeStoreErase(t *testing.T) {
	f := newStore(map[string]types.AuthConfig{
		validServerAddress: {
			Email: "foo@example.com",
		},
	})

	s := &nativeStore{
		programFunc: mockCommandFn,
		fileStore:   NewFileStore(f),
	}
	err := s.Erase(validServerAddress)
	assert.NilError(t, err)
	assert.Check(t, is.Len(f.GetAuthConfigs(), 0))
}

func TestNativeStoreEraseInvalidAddress(t *testing.T) {
	f := newStore(map[string]types.AuthConfig{
		validServerAddress: {
			Email: "foo@example.com",
		},
	})

	s := &nativeStore{
		programFunc: mockCommandFn,
		fileStore:   NewFileStore(f),
	}
	err := s.Erase(invalidServerAddress)
	assert.ErrorContains(t, err, "program failed")
}
