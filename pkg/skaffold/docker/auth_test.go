/*
Copyright 2018 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package docker

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/docker/cli/cli/config"
	"github.com/docker/docker/api/types"
)

const dockerCfg = `{
	"auths": {
			"https://appengine.gcr.io": {},
			"https://asia.gcr.io": {},
			"https://b.gcr.io": {},
			"https://beta.gcr.io": {},
			"https://bucket.gcr.io": {},
			"https://eu.gcr.io": {},
			"https://gcr.io": {},
			"https://gcr.kubernetes.io": {},
			"https://us.gcr.io": {}
	},
	"credsStore": "gcr",
	"credHelpers": {
			"appengine.gcr.io": "gcr",
			"asia.gcr.io": "gcr",
			"eu.gcr.io": "gcr",
			"gcr.io": "gcr",
			"gcr.kubernetes.io": "gcr",
			"us.gcr.io": "gcr"
	}
}`

func TestLoad(t *testing.T) {
	tempDir, cleanup := testutil.TempDir(t)
	defer cleanup()

	defer func(d string) { configDir = d }(configDir)
	configDir = tempDir

	ioutil.WriteFile(filepath.Join(configDir, config.ConfigFileName), []byte(dockerCfg), 0650)

	_, err := load()
	if err != nil {
		t.Errorf("Couldn't load docker config: %s", err)
	}
}

func TestLoadNotARealPath(t *testing.T) {
	defer func(d string) { configDir = d }(configDir)
	configDir = "not a real path"

	cf, err := load()
	if err == nil {
		t.Errorf("Expected error loading from bad path, but got none: %+v", cf)
	}
}

type testAuthHelper struct {
	getAuthConfigErr     error
	getAllAuthConfigsErr error
}

var gcrAuthConfig = types.AuthConfig{
	Username:      "bob",
	Password:      "saget",
	ServerAddress: "https://gcr.io",
}

func (t testAuthHelper) GetAuthConfig(string) (types.AuthConfig, error) {
	return gcrAuthConfig, t.getAuthConfigErr
}

func (t testAuthHelper) GetAllAuthConfigs() (map[string]types.AuthConfig, error) {
	return map[string]types.AuthConfig{
		"gcr.io": gcrAuthConfig,
	}, t.getAllAuthConfigsErr
}

func TestGetEncodedRegistryAuth(t *testing.T) {
	var tests = []struct {
		description string
		image       string
		authType    AuthConfigHelper
		expected    string
		shouldErr   bool
	}{
		{
			description: "encode successful",
			image:       "gcr.io/skaffold",
			authType:    testAuthHelper{},
			expected:    "eyJ1c2VybmFtZSI6ImJvYiIsInBhc3N3b3JkIjoic2FnZXQiLCJzZXJ2ZXJhZGRyZXNzIjoiaHR0cHM6Ly9nY3IuaW8ifQ==",
		},
		{
			description: "bad registry name",
			image:       ".",
			authType:    testAuthHelper{},
			shouldErr:   true,
		},
		{
			description: "bad registry name",
			image:       "gcr.io/skaffold",
			authType:    testAuthHelper{getAuthConfigErr: fmt.Errorf("")},
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			defer func(h AuthConfigHelper) { DefaultAuthHelper = h }(DefaultAuthHelper)
			DefaultAuthHelper = test.authType

			out, err := encodedRegistryAuth(context.Background(), nil, test.authType, test.image)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, out)
		})
	}
}
