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

package update

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/blang/semver"
)

func TestIsUpdateCheckEnabled(t *testing.T) {
	var tests = []struct {
		description       string
		updateCheckEnvVar string
		expected          bool
	}{
		{
			description:       "env is empty",
			updateCheckEnvVar: "",
			expected:          true,
		},
		{
			description:       "env is true",
			updateCheckEnvVar: "true",
			expected:          true,
		},
		{
			description:       "env is false",
			updateCheckEnvVar: "false",
			expected:          false,
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			reset := testutil.SetEnvs(t, map[string]string{
				"SKAFFOLD_UPDATE_CHECK": test.updateCheckEnvVar,
			})
			defer reset(t)

			testutil.CheckDeepEqual(t, test.expected, IsUpdateCheckEnabled())
		})
	}
}

func TestGetLatestAndCurrentVersion(t *testing.T) {
	var tests = []struct {
		description     string
		status          int
		body            string
		expectedLatest  semver.Version
		expectedCurrent semver.Version
		shouldErr       bool
	}{
		{
			description:     "status is not 200",
			status:          http.StatusNotFound,
			body:            "v1.2.3",
			expectedLatest:  semver.Version{Major: 0, Minor: 0, Patch: 0},
			expectedCurrent: semver.Version{Major: 0, Minor: 0, Patch: 0},
			shouldErr:       true,
		},
		{
			description:     "latest version is invalid",
			status:          http.StatusOK,
			body:            ".2.3",
			expectedLatest:  semver.Version{Major: 0, Minor: 0, Patch: 0},
			expectedCurrent: semver.Version{Major: 0, Minor: 0, Patch: 0},
			shouldErr:       true,
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.status)
				w.Write([]byte(test.body))
			})

			server := httptest.NewServer(handler)
			defer server.Close()

			gotLatest, gotCurrent, err := GetLatestAndCurrentVersion(server.URL)

			testutil.CheckError(t, test.shouldErr, err)
			testutil.CheckDeepEqual(t, test.expectedLatest, gotLatest)
			testutil.CheckDeepEqual(t, test.expectedCurrent, gotCurrent)
		})
	}
}
