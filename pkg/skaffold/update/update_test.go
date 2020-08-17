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
	"fmt"
	"testing"

	"github.com/blang/semver"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestCheckVersions(t *testing.T) {
	tests := []struct {
		description       string
		checkVersionsFunc func() (semver.Version, semver.Version, error)
		expected          []string
		enabled           bool
		configCheck       bool
		shouldError       bool
	}{
		{
			description: "globally disabled - disabled in config -> disabled",
			enabled:     false,
			configCheck: false,
			expected:    []string{"", ""},
		},
		{
			description: "globally enabled - disabled in config -> disabled",
			enabled:     true,
			configCheck: false,
			expected:    []string{"", ""},
		},
		{
			description: "globally disabled - enabled in config -> disabled",
			enabled:     false,
			configCheck: true,
			expected:    []string{"", ""},
		},
		{
			description:       "globally enabled - enabled in config - latest version -> disabled",
			enabled:           true,
			configCheck:       true,
			checkVersionsFunc: currentEqualsLatest,
			expected:          []string{"", ""},
		},
		{
			description:       "globally enabled - enabled in config - older version -> enabled",
			enabled:           true,
			configCheck:       true,
			checkVersionsFunc: latestGreaterThanCurrent,
			expected: []string{
				"There is a new version (1.1.0) of Skaffold available. Download it from:\n  https://github.com/GoogleContainerTools/skaffold/releases/tag/v1.1.0\n",
				"Your Skaffold version might be too old. Download the latest version (1.1.0) from:\n  https://github.com/GoogleContainerTools/skaffold/releases/tag/v1.1.0\n",
			},
		},
		{
			description:       "globally enabled - enabled in config - version check failed -> enabled",
			enabled:           true,
			configCheck:       true,
			checkVersionsFunc: errorGettingVersions,
			shouldError:       true,
			expected:          []string{"", ""},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&EnableCheck, test.enabled)
			t.Override(&isConfigUpdateCheckEnabled, func(string) bool { return test.configCheck })
			t.Override(&GetLatestAndCurrentVersion, test.checkVersionsFunc)

			msg, err := CheckVersion("foo")
			t.CheckErrorAndDeepEqual(test.shouldError, err, test.expected[0], msg)

			msg, err = CheckVersionOnError("foo")
			t.CheckErrorAndDeepEqual(test.shouldError, err, test.expected[1], msg)
		})
	}
}

func latestGreaterThanCurrent() (semver.Version, semver.Version, error) {
	return semver.Version{
			Major: 1,
			Minor: 1,
			Patch: 0,
		}, semver.Version{
			Major: 1,
			Minor: 0,
			Patch: 0,
		}, nil
}

func currentEqualsLatest() (semver.Version, semver.Version, error) {
	return semver.Version{
			Major: 1,
			Minor: 0,
			Patch: 0,
		}, semver.Version{
			Major: 1,
			Minor: 0,
			Patch: 0,
		}, nil
}

func errorGettingVersions() (semver.Version, semver.Version, error) {
	return semver.Version{}, semver.Version{}, fmt.Errorf("error getting versions")
}
