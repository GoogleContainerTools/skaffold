// Copyright 2018 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package authn

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
)

func TestConfigDir(t *testing.T) {
	clearEnv := func() {
		for _, e := range []string{"HOME", "DOCKER_CONFIG", "HOMEDRIVE", "HOMEPATH", "USERPROFILE"} {
			os.Unsetenv(e)
		}
	}

	for _, c := range []struct {
		desc             string
		env              map[string]string
		want             string
		wantErr          bool
		skipOnNonWindows bool
		skipOnWindows    bool
	}{{
		desc:    "no env set",
		env:     map[string]string{},
		wantErr: true,
	}, {
		desc: "DOCKER_CONFIG",
		env:  map[string]string{"DOCKER_CONFIG": filepath.FromSlash("/path/to/.docker")},
		want: filepath.FromSlash("/path/to/.docker"),
	}, {
		desc:          "HOME",
		skipOnWindows: true,
		env:           map[string]string{"HOME": filepath.FromSlash("/my/home")},
		want:          filepath.FromSlash("/my/home/.docker"),
	}, {
		desc:             "USERPROFILE",
		skipOnNonWindows: true,
		env:              map[string]string{"USERPROFILE": filepath.FromSlash("/user/profile")},
		want:             filepath.FromSlash("/user/profile/.docker"),
	}} {
		t.Run(c.desc, func(t *testing.T) {
			if c.skipOnNonWindows && runtime.GOOS != "windows" {
				t.Skip("Skipping on non-Windows")
			}
			if c.skipOnWindows && runtime.GOOS == "windows" {
				t.Skip("Skipping on Windows")
			}
			clearEnv()
			for k, v := range c.env {
				os.Setenv(k, v)
			}
			got, err := configDir()
			if err == nil && c.wantErr {
				t.Errorf("configDir() returned no error, got %q", got)
			} else if err != nil && !c.wantErr {
				t.Errorf("configDir(): %v", err)
			}

			if got != c.want {
				t.Errorf("configDir(); got %q, want %q", got, c.want)
			}
		})
	}
}

var (
	fresh           = 0
	testRegistry, _ = name.NewRegistry("test.io", name.WeakValidation)
)

// setupConfigDir sets up an isolated configDir() for this test.
func setupConfigDir(t *testing.T) string {
	tmpdir := os.Getenv("TEST_TMPDIR")
	if tmpdir == "" {
		var err error
		tmpdir, err = ioutil.TempDir("", "keychain_test")
		if err != nil {
			t.Fatalf("creating temp dir: %v", err)
		}
	}

	fresh = fresh + 1
	p := fmt.Sprintf("%s/%d", tmpdir, fresh)
	os.Setenv("DOCKER_CONFIG", p)
	if err := os.Mkdir(p, 0777); err != nil {
		t.Fatalf("mkdir %q: %v", p, err)
	}
	return p
}

func setupConfigFile(t *testing.T, content string) string {
	cd := setupConfigDir(t)
	p := filepath.Join(cd, "config.json")
	if err := ioutil.WriteFile(p, []byte(content), 0600); err != nil {
		t.Fatalf("write %q: %v", p, err)
	}

	// return the config dir so we can clean up
	return cd
}

func checkOutput(t *testing.T, want string) {
	auth, err := DefaultKeychain.Resolve(testRegistry)
	if err != nil {
		t.Errorf("Resolve() = %v", err)
	}

	got, err := auth.Authorization()
	if err != nil {
		t.Errorf("Authorization() = %v", err)
	}
	if got != want {
		t.Errorf("Authorization(); got %v, want %v", got, want)
	}
}

func checkAnonymousFallback(t *testing.T) {
	checkOutput(t, "")
}

func checkFooBarOutput(t *testing.T) {
	// base64(foo:bar)
	checkOutput(t, "Basic Zm9vOmJhcg==")
}

func checkHelper(t *testing.T) {
	auth, err := DefaultKeychain.Resolve(testRegistry)
	if err != nil {
		t.Errorf("Resolve() = %v", err)
	}

	help, ok := auth.(*helper)
	if !ok {
		t.Errorf("Resolve(); got %T, want *helper", auth)
	}
	if help.name != "test" {
		t.Errorf("Resolve().name; got %v, want \"test\"", help.name)
	}
	if help.domain != testRegistry {
		t.Errorf("Resolve().domain; got %v, want %v", help.domain, testRegistry)
	}
}

func TestNoConfig(t *testing.T) {
	cd := setupConfigDir(t)
	defer os.RemoveAll(filepath.Dir(cd))

	checkAnonymousFallback(t)
}

func TestVariousPaths(t *testing.T) {
	tests := []struct {
		content string
		check   func(*testing.T)
	}{{
		content: `}{`,
		check:   checkAnonymousFallback,
	}, {
		content: `{"credHelpers": {"https://test.io": "test"}}`,
		check:   checkHelper,
	}, {
		content: `{"credsStore": "test"}`,
		check:   checkHelper,
	}, {
		content: `{"auths": {"http://test.io/v2/": {"auth": "Zm9vOmJhcg=="}}}`,
		check:   checkFooBarOutput,
	}, {
		content: `{"auths": {"https://test.io/v1/": {"username": "foo", "password": "bar"}}}`,
		check:   checkFooBarOutput,
	}, {
		content: `{"auths": {"other.io": {"username": "asdf", "password": "fdsa"}}}`,
		check:   checkAnonymousFallback,
	}}

	for _, test := range tests {
		cd := setupConfigFile(t, test.content)
		// For some reason, these tempdirs don't get cleaned up.
		defer os.RemoveAll(filepath.Dir(cd))

		test.check(t)
	}
}
