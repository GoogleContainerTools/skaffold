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

package apiversion

import (
	"testing"

	"github.com/blang/semver"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		description string
		version     string
		want        semver.Version
		shouldErr   bool
	}{
		{
			description: "full",
			version:     "skaffold/v7alpha3",
			want: semver.Version{
				Major: 7,
				Pre: []semver.PRVersion{
					{
						VersionStr: "alpha",
					},
					{
						VersionNum: 3,
						IsNum:      true,
					},
				},
			},
		},
		{
			description: "ga",
			version:     "skaffold/v4",
			want: semver.Version{
				Major: 4,
			},
		},
		{
			description: "incorrect",
			version:     "apps/v1",
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			got, err := Parse(test.version)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.want, got)
		})
	}
}
