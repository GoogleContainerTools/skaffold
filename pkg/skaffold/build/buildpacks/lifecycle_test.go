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
	"testing"
)

func TestRewriteLifecycleStatusCode(t *testing.T) {
	tests := []struct {
		errText  string
		expected string
	}{
		{"blah blah", "blah blah"},
		{"failed with status code: 0", "lifecycle failed with status code 0"},
		{"failed with status code: 1", "lifecycle failed with status code 1"},
		{"failed with status code: 2", "lifecycle failed with status code 2"},
		{"failed with status code: 3", "lifecycle reported invalid arguments"}, //CodeInvalidArgs
		{"failed with status code: 4", "lifecycle failed with status code 4"},
		{"failed with status code: 5", "lifecycle failed with status code 5"},
		{"failed with status code: 6", "buildpacks could not determine application type"}, //CodeFailedDetect
		{"failed with status code: 7", "buildpacks failed to build"},                      //CodeFailedBuild
		{"failed with status code: 8", "lifecycle failed with status code 8"},
		{"failed with status code: 9", "lifecycle failed with status code 9"},
		{"failed with status code: 10", "buildpacks failed to save image"}, //CodeFailedSave
		{"failed with status code: 11", "incompatible lifecycle version"},  //CodeIncompatible
	}
	for _, test := range tests {
		result := rewriteLifecycleStatusCode(errors.New(test.errText))
		if result.Error() != test.expected {
			t.Errorf("got %q, wanted %q", result.Error(), test.expected)
		}
	}
}

func TestMapLifecycleStatusCode(t *testing.T) {
	tests := []struct {
		code     int
		expected string
	}{
		{0, "lifecycle failed with status code 0"},
		{1, "lifecycle failed with status code 1"},
		{2, "lifecycle failed with status code 2"},
		{3, "lifecycle reported invalid arguments"}, // CodeInvalidArgs
		{4, "lifecycle failed with status code 4"},
		{5, "lifecycle failed with status code 5"},
		{6, "buildpacks could not determine application type"}, // CodeFailedDetect
		{7, "buildpacks failed to build"},                      // CodeFailedBuild
		{8, "lifecycle failed with status code 8"},
		{9, "lifecycle failed with status code 9"},
		{10, "buildpacks failed to save image"}, //CodeFailedSave
		{11, "incompatible lifecycle version"},  // CodeIncompatible
		{12, "lifecycle failed with status code 12"},
	}
	for _, test := range tests {
		result := mapLifecycleStatusCode(test.code)
		if result != test.expected {
			t.Errorf("code %d: got %q, wanted %q", test.code, result, test.expected)
		}
	}
}
