// +build !windows

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

package gcp

import (
	"testing"

	"github.com/docker/cli/cli/config/configfile"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestAutoConfigureGCRCredentialHelper(t *testing.T) {
	tests := []struct {
		description  string
		helperInPath bool
		config       *configfile.ConfigFile
		expected     *configfile.ConfigFile
	}{
		{
			description:  "add to nil map",
			helperInPath: true,
			config:       &configfile.ConfigFile{},
			expected: &configfile.ConfigFile{
				CredentialHelpers: map[string]string{
					"gcr.io":             "gcloud",
					"us.gcr.io":          "gcloud",
					"eu.gcr.io":          "gcloud",
					"asia.gcr.io":        "gcloud",
					"staging-k8s.gcr.io": "gcloud",
					"marketplace.gcr.io": "gcloud",
				},
			},
		},
		{
			description:  "add to empty map",
			helperInPath: true,
			config: &configfile.ConfigFile{
				CredentialHelpers: map[string]string{},
			},
			expected: &configfile.ConfigFile{
				CredentialHelpers: map[string]string{
					"gcr.io":             "gcloud",
					"us.gcr.io":          "gcloud",
					"eu.gcr.io":          "gcloud",
					"asia.gcr.io":        "gcloud",
					"staging-k8s.gcr.io": "gcloud",
					"marketplace.gcr.io": "gcloud",
				},
			},
		},
		{
			description: "leave existing helper",
			config: &configfile.ConfigFile{
				CredentialHelpers: map[string]string{
					"gcr.io":             "existing",
					"us.gcr.io":          "existing",
					"eu.gcr.io":          "existing",
					"asia.gcr.io":        "existing",
					"staging-k8s.gcr.io": "existing",
					"marketplace.gcr.io": "existing",
				},
			},
			expected: &configfile.ConfigFile{
				CredentialHelpers: map[string]string{
					"gcr.io":             "existing",
					"us.gcr.io":          "existing",
					"eu.gcr.io":          "existing",
					"asia.gcr.io":        "existing",
					"staging-k8s.gcr.io": "existing",
					"marketplace.gcr.io": "existing",
				},
			},
		},
		{
			description:  "ignore if gcloud is not in PATH",
			helperInPath: false,
			config:       &configfile.ConfigFile{},
			expected:     &configfile.ConfigFile{},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir()
			t.SetEnvs(map[string]string{"PATH": tmpDir.Root()})

			if test.helperInPath {
				tmpDir.Write("docker-credential-gcloud", "")
			}

			AutoConfigureGCRCredentialHelper(test.config)

			t.CheckDeepEqual(test.expected, test.config)
		})
	}
}
