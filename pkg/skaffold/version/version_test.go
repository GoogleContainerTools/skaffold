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

package version

import (
	"testing"

	"github.com/blang/semver"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		description string
		in          string
		out         semver.Version
		shouldErr   bool
	}{
		{
			description: "parse version correct",
			in:          "v0.10.0",
			out:         semver.MustParse("0.10.0"),
		},
		{
			description: "parse version correct without leading v",
			in:          "0.10.0",
			out:         semver.MustParse("0.10.0"),
		},
		{
			description: "parse error",
			in:          "notasemver",
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			actual, err := ParseVersion(test.in)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.out, actual)
		})
	}
}

func TestUserAgent(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.Override(&platform, "osx")
		t.Override(&version, "1.0")

		userAgent := UserAgent()

		t.CheckDeepEqual("skaffold/osx/1.0", userAgent)
	})
}
