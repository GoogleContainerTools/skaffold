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
		{lifecycle.CodeFailedDetect, "buildpacks could not determine application type"},
		{lifecycle.CodeFailedDetectWithErrors, "buildpacks could not determine application type"},
		{lifecycle.CodeAnalyzeError, "buildpacks failed analyzing metadata from previous builds"},
		{lifecycle.CodeRestoreError, "buildpacks failed to restoring cached layers"},
		{lifecycle.CodeFailedBuildWithErrors, "buildpacks failed to build image"},
		{lifecycle.CodeBuildError, "buildpacks failed to build image"},
		{lifecycle.CodeExportError, "buildpacks failed to save image and cache layers"},

		{0, "lifecycle failed with status code 0"},
		// we do not handle CodeRebaseError
		{lifecycle.CodeRebaseError, "lifecycle failed with status code 602"},
		// we do not handle CodeLaunchError
		{lifecycle.CodeLaunchError, "lifecycle failed with status code 702"},
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
