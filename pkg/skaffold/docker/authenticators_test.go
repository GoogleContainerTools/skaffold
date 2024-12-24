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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"

	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

// based on https://cloud.google.com/container-registry/docs/advanced-authentication#linux-macos
func TestResolve(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test doesn't work on windows")
	}

	tests := []struct {
		description           string
		dockerConfig          string
		registry              string
		gcloudOutput          string
		credentialsValues     map[string]string
		tokenURIRequestOutput string
		gcloudInPath          bool
		expectAnonymous       bool
	}{
		{
			description:  "Application Default Credentials configured and working",
			registry:     "gcr.io",
			dockerConfig: `{"credHelpers":{"anydomain.io": "gcloud"}}`,
			credentialsValues: map[string]string{
				"client_id":     "123456.apps.googleusercontent.com",
				"client_secret": "THE-SECRET",
				"refresh_token": "REFRESH-TOKEN",
				"type":          "authorized_user",
			},
			tokenURIRequestOutput: `{"access_token":"TOKEN","expires_in": 3599}`,
			expectAnonymous:       false,
		},
		{
			description:     "gcloud is configured and working",
			registry:        "gcr.io",
			dockerConfig:    `{"credHelpers":{"gcr.io": "gcloud"}}`,
			gcloudInPath:    true,
			gcloudOutput:    "#!/bin/sh\necho '{\"credential\":{\"access_token\":\"TOKEN\",\"token_expiry\":\"2999-01-01T08:48:55Z\"}}'",
			expectAnonymous: false,
		},
		{
			description:     "gcloud is configured but not found (anonymous)",
			registry:        "gcr.io",
			dockerConfig:    `{"credHelpers":{"gcr.io": "gcloud"}}`,
			gcloudInPath:    false,
			expectAnonymous: true,
		},
		{
			description:     "gcloud is configured but not working (anonymous)",
			registry:        "gcr.io",
			dockerConfig:    `{"credHelpers":{"gcr.io": "gcloud"}}`,
			gcloudInPath:    true,
			gcloudOutput:    `exit 1`,
			expectAnonymous: true,
		},
		{
			description:     "gcloud is not configured but working",
			registry:        "gcr.io",
			dockerConfig:    `{}`,
			gcloudInPath:    true,
			gcloudOutput:    "#!/bin/sh\necho '{\"credential\":{\"access_token\":\"TOKEN\",\"token_expiry\":\"2999-01-01T08:48:55Z\"}}'",
			expectAnonymous: false,
		},
		{
			description:     "gcloud is not configured and not working (anonymous)",
			registry:        "eu.gcr.io",
			dockerConfig:    `{}`,
			gcloudInPath:    true,
			gcloudOutput:    `exit 1`,
			expectAnonymous: true,
		},
		{
			description:     "anonymous",
			registry:        "docker",
			dockerConfig:    `{}`,
			expectAnonymous: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir().Write("config.json", test.dockerConfig)

			var path = tmpDir.Root()
			if test.gcloudInPath {
				path = tmpDir.Root() + ":" + os.Getenv("PATH")
				tmpDir.Write("gcloud", test.gcloudOutput)
			}

			var adc string
			if test.credentialsValues != nil {
				url := startTokenServer(t, test.tokenURIRequestOutput)
				credentialsFile := getCredentialsFile(t, test.credentialsValues, url)
				tmpDir.Write("credentials.json", credentialsFile)
				adc = tmpDir.Path("credentials.json")
			}

			t.SetEnvs(map[string]string{
				"DOCKER_CONFIG":                  tmpDir.Path("config.json"),
				"PATH":                           path,
				"HOME":                           tmpDir.Root(), // This is to prevent the go-containerregistry library from using ADCs that are already present on the computer.
				"GOOGLE_APPLICATION_CREDENTIALS": adc,
			})

			registry, err := name.NewRegistry(test.registry)
			t.CheckNoError(err)

			kc := &Keychain{configDir: tmpDir.Root()}
			authenticator, err := kc.Resolve(registry)
			t.CheckNotNil(authenticator)
			t.CheckNoError(err)

			authConfig, err := authenticator.Authorization()
			if test.expectAnonymous {
				t.CheckDeepEqual(&authn.AuthConfig{}, authConfig)
			} else {
				t.CheckDeepEqual("TOKEN", authConfig.Password)
			}
			t.CheckNoError(err)
		})
	}
}

func startTokenServer(t *testutil.T, reqOutput string) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(reqOutput))
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func getCredentialsFile(t *testutil.T, credValues map[string]string, tokenRefreshURL string) string {
	credValues["token_uri"] = tokenRefreshURL
	credFile, err := json.Marshal(credValues)
	if err != nil {
		t.Fatalf("error generating credential files: %v", err)
	}
	return string(credFile)
}
