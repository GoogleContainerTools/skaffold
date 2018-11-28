package credentials

import (
	"testing"

	"github.com/docker/docker/api/types"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

type fakeStore struct {
	configs map[string]types.AuthConfig
}

func (f *fakeStore) Save() error {
	return nil
}

func (f *fakeStore) GetAuthConfigs() map[string]types.AuthConfig {
	return f.configs
}

func (f *fakeStore) GetFilename() string {
	return "/tmp/docker-fakestore"
}

func newStore(auths map[string]types.AuthConfig) store {
	return &fakeStore{configs: auths}
}

func TestFileStoreAddCredentials(t *testing.T) {
	f := newStore(make(map[string]types.AuthConfig))

	s := NewFileStore(f)
	auth := types.AuthConfig{
		Auth:          "super_secret_token",
		Email:         "foo@example.com",
		ServerAddress: "https://example.com",
	}
	err := s.Store(auth)
	assert.NilError(t, err)
	assert.Check(t, is.Len(f.GetAuthConfigs(), 1))

	actual, ok := f.GetAuthConfigs()["https://example.com"]
	assert.Check(t, ok)
	assert.Check(t, is.DeepEqual(auth, actual))
}

func TestFileStoreGet(t *testing.T) {
	f := newStore(map[string]types.AuthConfig{
		"https://example.com": {
			Auth:          "super_secret_token",
			Email:         "foo@example.com",
			ServerAddress: "https://example.com",
		},
	})

	s := NewFileStore(f)
	a, err := s.Get("https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	if a.Auth != "super_secret_token" {
		t.Fatalf("expected auth `super_secret_token`, got %s", a.Auth)
	}
	if a.Email != "foo@example.com" {
		t.Fatalf("expected email `foo@example.com`, got %s", a.Email)
	}
}

func TestFileStoreGetAll(t *testing.T) {
	s1 := "https://example.com"
	s2 := "https://example2.com"
	f := newStore(map[string]types.AuthConfig{
		s1: {
			Auth:          "super_secret_token",
			Email:         "foo@example.com",
			ServerAddress: "https://example.com",
		},
		s2: {
			Auth:          "super_secret_token2",
			Email:         "foo@example2.com",
			ServerAddress: "https://example2.com",
		},
	})

	s := NewFileStore(f)
	as, err := s.GetAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(as) != 2 {
		t.Fatalf("wanted 2, got %d", len(as))
	}
	if as[s1].Auth != "super_secret_token" {
		t.Fatalf("expected auth `super_secret_token`, got %s", as[s1].Auth)
	}
	if as[s1].Email != "foo@example.com" {
		t.Fatalf("expected email `foo@example.com`, got %s", as[s1].Email)
	}
	if as[s2].Auth != "super_secret_token2" {
		t.Fatalf("expected auth `super_secret_token2`, got %s", as[s2].Auth)
	}
	if as[s2].Email != "foo@example2.com" {
		t.Fatalf("expected email `foo@example2.com`, got %s", as[s2].Email)
	}
}

func TestFileStoreErase(t *testing.T) {
	f := newStore(map[string]types.AuthConfig{
		"https://example.com": {
			Auth:          "super_secret_token",
			Email:         "foo@example.com",
			ServerAddress: "https://example.com",
		},
	})

	s := NewFileStore(f)
	err := s.Erase("https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	// file store never returns errors, check that the auth config is empty
	a, err := s.Get("https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	if a.Auth != "" {
		t.Fatalf("expected empty auth token, got %s", a.Auth)
	}
	if a.Email != "" {
		t.Fatalf("expected empty email, got %s", a.Email)
	}
}
