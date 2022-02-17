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

package buildpacks

import (
	"errors"
	"fmt"
	"testing"

	lifecycle "github.com/buildpacks/lifecycle/cmd"
	pack "github.com/buildpacks/pack/pkg/client"

	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestLifecycleStatusCode(t *testing.T) {
	tests := []struct {
		code     int
		expected string
	}{
		{lifecycle.CodeFailed, "buildpacks lifecycle failed"},
		{lifecycle.CodeInvalidArgs, "lifecycle reported invalid arguments"},
		{lifecycle.CodeIncompatiblePlatformAPI, "incompatible version of Platform API"},
		{lifecycle.CodeIncompatibleBuildpackAPI, "incompatible version of Buildpacks API"},

		{0, "lifecycle failed with status code 0"},
	}
	for _, test := range tests {
		result := mapLifecycleStatusCode(test.code)
		if result != test.expected {
			t.Errorf("code %d: got %q, wanted %q", test.code, result, test.expected)
		}
	}
	for _, test := range tests {
		errText := fmt.Sprintf("failed with status code: %d", test.code)
		result := rewriteLifecycleStatusCode(errors.New(errText))
		if result.Error() != test.expected {
			t.Errorf("got %q, wanted %q", result.Error(), test.expected)
		}
	}
}

func TestContainerConfig(t *testing.T) {
	tests := []struct {
		description string
		volumes     []latestV2.BuildpackVolume
		shouldErr   bool
		expected    pack.ContainerConfig
	}{
		{
			description: "single volume with no options",
			volumes:     []latestV2.BuildpackVolume{{Host: "/foo", Target: "/bar"}},
			expected:    pack.ContainerConfig{Volumes: []string{"/foo:/bar"}},
		},
		{
			description: "single volume with  options",
			volumes:     []latestV2.BuildpackVolume{{Host: "/foo", Target: "/bar", Options: "rw"}},
			expected:    pack.ContainerConfig{Volumes: []string{"/foo:/bar:rw"}},
		},
		{
			description: "multiple volumes",
			volumes: []latestV2.BuildpackVolume{
				{Host: "/foo", Target: "/bar", Options: "rw"},
				{Host: "/bat", Target: "/baz", Options: "ro"},
			},
			expected: pack.ContainerConfig{Volumes: []string{"/foo:/bar:rw", "/bat:/baz:ro"}},
		},
		{
			description: "missing host is skipped",
			volumes:     []latestV2.BuildpackVolume{{Host: "", Target: "/bar"}},
			shouldErr:   true,
		},
		{
			description: "missing target is skipped",
			volumes:     []latestV2.BuildpackVolume{{Host: "/foo", Target: ""}},
			shouldErr:   true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			artifact := latestV2.BuildpackArtifact{
				Volumes: &test.volumes,
			}
			result, err := containerConfig(&artifact)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, result)
		})
	}
}
