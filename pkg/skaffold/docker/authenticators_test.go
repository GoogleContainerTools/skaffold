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
	"os"
	"runtime"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestResolve(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test doesn't work on windows")
	}

	tests := []struct {
		description     string
		dockerConfig    string
		registry        string
		gcloudOutput    string
		gcloudInPath    bool
		expectAnonymous bool
	}{
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

			var path string
			if test.gcloudInPath {
				path = tmpDir.Root() + ":" + os.Getenv("PATH")
				tmpDir.Write("gcloud", test.gcloudOutput)
			} else {
				path = tmpDir.Root()
			}

			t.SetEnvs(map[string]string{
				"DOCKER_CONFIG": tmpDir.Path("config.json"),
				"PATH":          path,
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
				t.CheckDeepEqual("TOKEN", authConfig.RegistryToken)
			}
			t.CheckNoError(err)
		})
	}
}
