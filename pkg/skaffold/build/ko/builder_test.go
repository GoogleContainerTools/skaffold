/*
Copyright 2021 The Skaffold Authors

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

package ko

import (
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		description          string
		baseImage            string
		platforms            []string
		workspace            string
		sourceDir            string
		wantPlatform         string
		wantWorkingDirectory string
	}{
		{
			description: "all zero value",
		},
		{
			description: "empty platforms",
			platforms:   []string{},
		},
		{
			description: "base image",
			baseImage:   "gcr.io/distroless/static:nonroot",
		},
		{
			description:  "multiple platforms",
			platforms:    []string{"linux/amd64", "linux/arm64"},
			wantPlatform: "linux/amd64,linux/arm64",
		},
		{
			description:          "workspace",
			workspace:            "my-app-subdirectory",
			wantWorkingDirectory: "my-app-subdirectory",
		},
		{
			description:          "source dir",
			sourceDir:            "my-go-mod-is-here",
			wantWorkingDirectory: "my-go-mod-is-here",
		},
		{
			description:          "workspace and source dir",
			workspace:            "my-app-subdirectory",
			sourceDir:            "my-go-mod-is-here",
			wantWorkingDirectory: "my-app-subdirectory" + string(filepath.Separator) + "my-go-mod-is-here",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			bo := buildOptions(test.baseImage, test.platforms, test.workspace, test.sourceDir)
			t.CheckDeepEqual(test.baseImage, bo.BaseImage)
			if bo.ConcurrentBuilds < 1 {
				t.Errorf("ConcurrentBuilds must always be >= 1 for the ko builder")
			}
			t.CheckDeepEqual(test.wantPlatform, bo.Platform)
			t.CheckDeepEqual(version.UserAgentWithClient(), bo.UserAgent)
			t.CheckDeepEqual(test.wantWorkingDirectory, bo.WorkingDirectory)
		})
	}
}
