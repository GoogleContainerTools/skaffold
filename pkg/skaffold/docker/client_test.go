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
	"os"
	"testing"

	testutil "github.com/GoogleCloudPlatform/skaffold/test"
)

func TestNewEnvClient(t *testing.T) {
	var tests = []struct {
		description string
		envs        map[string]string
		shouldErr   bool
	}{
		{
			description: "get client",
			envs: map[string]string{
				"DOCKER_HOST": "http://127.0.0.1:8080",
			},
		},
		{
			description: "invalid cert path",
			envs: map[string]string{
				"DOCKER_HOST":      "http://127.0.0.1:8080",
				"DOCKER_CERT_PATH": "invalid/cert/path",
			},
			shouldErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			unsetEnvs := setEnvs(t, test.envs)
			_, _, err := NewImageAPIClient()
			testutil.CheckError(t, test.shouldErr, err)
			unsetEnvs(t)
		})
	}

}

func setEnvs(t *testing.T, envs map[string]string) func(*testing.T) {
	prevEnvs := map[string]string{}
	for key, value := range envs {
		prevEnv := os.Getenv(key)
		prevEnvs[key] = prevEnv
		err := os.Setenv(key, value)
		if err != nil {
			t.Error(err)
		}
	}
	return func(t *testing.T) {
		for key, value := range prevEnvs {
			err := os.Setenv(key, value)
			if err != nil {
				t.Error(err)
			}
		}
	}
}
