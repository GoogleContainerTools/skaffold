/*
Copyright 2019 The Skaffold Authors

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
	"runtime"
	"testing"

	"github.com/docker/docker/api/types"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

type testAuthHelper struct {
	getAuthConfigErr     error
	getAllAuthConfigsErr error
}

var gcrAuthConfig = types.AuthConfig{
	Username:      "bob",
	Password:      "saget",
	ServerAddress: "https://gcr.io",
}

var allAuthConfig = map[string]types.AuthConfig{
	"gcr.io": gcrAuthConfig,
}

func (t testAuthHelper) GetAuthConfig(string) (types.AuthConfig, error) {
	return gcrAuthConfig, t.getAuthConfigErr
}
func (t testAuthHelper) GetAllAuthConfigs(context.Context) (map[string]types.AuthConfig, error) {
	return allAuthConfig, t.getAllAuthConfigsErr
}

func TestGetAllAuthConfigs(t *testing.T) {
	testutil.Run(t, "auto-configure gcr.io", func(t *testutil.T) {
		if runtime.GOOS == "windows" {
			t.Skip("test doesn't work on windows")
		}

		tmpDir := t.NewTempDir().
			Write("config.json", `{"credHelpers":{"my.registry":"helper"}}`).
			Write("docker-credential-gcloud", `#!/bin/sh
read server
echo "{\"Username\":\"<token>\",\"Secret\":\"TOKEN_$server\"}"`).
			Write("docker-credential-helper", `#!/bin/sh
read server
echo "{\"Username\":\"<token>\",\"Secret\":\"TOKEN_$server\"}"`)
		t.Override(&configDir, tmpDir.Root())
		t.SetEnvs(map[string]string{"PATH": tmpDir.Root()})

		auth, err := DefaultAuthHelper.GetAllAuthConfigs(context.Background())

		t.CheckNoError(err)
		t.CheckDeepEqual(map[string]types.AuthConfig{
			"asia.gcr.io":        {IdentityToken: "TOKEN_asia.gcr.io"},
			"eu.gcr.io":          {IdentityToken: "TOKEN_eu.gcr.io"},
			"gcr.io":             {IdentityToken: "TOKEN_gcr.io"},
			"my.registry":        {IdentityToken: "TOKEN_my.registry"},
			"marketplace.gcr.io": {IdentityToken: "TOKEN_marketplace.gcr.io"},
			"staging-k8s.gcr.io": {IdentityToken: "TOKEN_staging-k8s.gcr.io"},
			"us.gcr.io":          {IdentityToken: "TOKEN_us.gcr.io"},
		}, auth)
	})

	testutil.Run(t, "invalid config.json", func(t *testutil.T) {
		tmpDir := t.NewTempDir().Write("config.json", "invalid json")
		t.Override(&configDir, tmpDir.Root())

		auth, err := DefaultAuthHelper.GetAllAuthConfigs(context.Background())

		t.CheckError(true, err)
		t.CheckEmpty(auth)
	})
}

func TestGetEncodedRegistryAuth(t *testing.T) {
	tests := []struct {
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
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&DefaultAuthHelper, test.authType)

			l := &localDaemon{}
			out, err := l.encodedRegistryAuth(context.Background(), test.authType, test.image)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, out)
		})
	}
}
