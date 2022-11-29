/*
Copyright 2022 The Skaffold Authors

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

package platform

import (
	"testing"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		description string
		m           Matcher
		isEmpty     bool
	}{
		{
			description: "all matcher",
			m:           Matcher{All: true},
			isEmpty:     false,
		},
		{
			description: "non-empty",
			m:           Matcher{Platforms: []v1.Platform{{OS: "linux"}}},
			isEmpty:     false,
		},
		{
			description: "empty",
			m:           Matcher{},
			isEmpty:     true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			isEmpty := test.m.IsEmpty()
			isNotEmpty := test.m.IsNotEmpty()

			t.CheckDeepEqual(test.isEmpty, isEmpty)
			t.CheckDeepEqual(!test.isEmpty, isNotEmpty)
		})
	}
}

func TestIsMultiOrCrossPlatform(t *testing.T) {
	tests := []struct {
		description     string
		m               Matcher
		isMultiPlatform bool
		isCrossPlatform bool
	}{
		{
			description:     "all matcher",
			m:               Matcher{All: true},
			isMultiPlatform: true,
			isCrossPlatform: true,
		},
		{
			description:     "multiple platform targets",
			m:               Matcher{Platforms: []v1.Platform{{Architecture: "amd64"}, {Architecture: "arm64"}}},
			isMultiPlatform: true,
			isCrossPlatform: true,
		},
		{
			description:     "single platform target",
			m:               Matcher{Platforms: []v1.Platform{{Architecture: "arm", OS: "freebsd"}}},
			isMultiPlatform: false,
			isCrossPlatform: true,
		},
		{
			description:     "no platform target",
			isMultiPlatform: false,
			isCrossPlatform: false,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			isMultiPlatform := test.m.IsMultiPlatform()
			t.CheckDeepEqual(test.isMultiPlatform, isMultiPlatform)
			isCrossPlatform := test.m.IsCrossPlatform()
			t.CheckDeepEqual(test.isCrossPlatform, isCrossPlatform)
		})
	}
}

func TestArray(t *testing.T) {
	tests := []struct {
		description string
		m           Matcher
		expected    []string
	}{
		{
			description: "all matcher",
			m:           Matcher{All: true},
			expected:    []string{"all"},
		}, {
			description: "multiple platform targets",
			m: Matcher{Platforms: []v1.Platform{
				{OS: "linux", Architecture: "amd64"},
				{OS: "linux", Architecture: "arm64"},
			}},
			expected: []string{"linux/amd64", "linux/arm64"},
		},
		{
			description: "single platform target",
			m:           Matcher{Platforms: []v1.Platform{{OS: "linux", Architecture: "arm64"}}},
			expected:    []string{"linux/arm64"},
		},
		{
			description: "no platform target",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.CheckDeepEqual(test.expected, test.m.Array())
		})
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		description string
		m           Matcher
		expected    string
	}{
		{
			description: "all matcher",
			m:           Matcher{All: true},
			expected:    "all",
		}, {
			description: "multiple platform targets",
			m: Matcher{Platforms: []v1.Platform{
				{OS: "linux", Architecture: "amd64"},
				{OS: "linux", Architecture: "arm64"},
			}},
			expected: "linux/amd64,linux/arm64",
		},
		{
			description: "single platform target",
			m:           Matcher{Platforms: []v1.Platform{{OS: "linux", Architecture: "arm64"}}},
			expected:    "linux/arm64",
		},
		{
			description: "no platform target",
			expected:    "",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.CheckDeepEqual(test.expected, test.m.String())
		})
	}
}

func TestIntersect(t *testing.T) {
	tests := []struct {
		description string
		m1          Matcher
		m2          Matcher
		expected    Matcher
	}{
		{
			description: "all with all",
			m1:          Matcher{All: true},
			m2:          Matcher{All: true},
			expected:    Matcher{All: true},
		},
		{
			description: "all with empty",
			m1:          Matcher{All: true},
			m2:          Matcher{},
			expected:    Matcher{},
		},
		{
			description: "empty with all",
			m1:          Matcher{},
			m2:          Matcher{All: true},
			expected:    Matcher{},
		},
		{
			description: "all with selected platforms",
			m1:          Matcher{All: true},
			m2: Matcher{Platforms: []v1.Platform{
				{OS: "linux", Architecture: "amd64"},
				{OS: "linux", Architecture: "arm64"},
			}},
			expected: Matcher{Platforms: []v1.Platform{
				{OS: "linux", Architecture: "amd64"},
				{OS: "linux", Architecture: "arm64"},
			}},
		},
		{
			description: "selected platforms with all",
			m1: Matcher{Platforms: []v1.Platform{
				{OS: "linux", Architecture: "amd64"},
				{OS: "linux", Architecture: "arm64"},
			}},
			m2: Matcher{All: true},
			expected: Matcher{Platforms: []v1.Platform{
				{OS: "linux", Architecture: "amd64"},
				{OS: "linux", Architecture: "arm64"},
			}},
		},
		{
			description: "some matching",
			m1: Matcher{Platforms: []v1.Platform{
				{OS: "linux", Architecture: "amd64"},
				{OS: "windows", Architecture: "amd64"},
			}},
			m2: Matcher{Platforms: []v1.Platform{
				{OS: "linux", Architecture: "amd64"},
				{OS: "darwin", Architecture: "arm64"},
			}},
			expected: Matcher{Platforms: []v1.Platform{
				{OS: "linux", Architecture: "amd64"},
			}},
		},
		{
			description: "no matching",
			m1: Matcher{Platforms: []v1.Platform{
				{OS: "linux", Architecture: "arm64"},
				{OS: "windows", Architecture: "amd64"},
			}},
			m2: Matcher{Platforms: []v1.Platform{
				{OS: "linux", Architecture: "amd64"},
				{OS: "darwin", Architecture: "arm64"},
			}},
			expected: Matcher{},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.CheckDeepEqual(test.expected, test.m1.Intersect(test.m2))
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		description string
		input       []string
		shouldErr   bool
	}{
		{
			description: "known platforms",
			input:       []string{"linux/arm64", "darwin/amd64", "freebsd/arm"},
			shouldErr:   false,
		},
		{
			description: "unknown os",
			input:       []string{"foo/arm64", "darwin/amd64", "freebsd/arm"},
			shouldErr:   true,
		},
		{
			description: "unknown arch",
			input:       []string{"linux/arm64", "darwin/foo", "freebsd/arm"},
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			_, err := Parse(test.input)
			t.CheckError(test.shouldErr, err)
		})
	}
}
