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
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		description  string
		baseImage    string
		platforms    []string
		wantPlatform string
		workspace    string
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
			description: "workspace",
			workspace:   "my-app-subdirectory",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			bo := buildOptions(test.baseImage, test.platforms, test.workspace)
			if bo.BaseImage != test.baseImage {
				t.Errorf("wanted BaseImage (%q), got (%q)", test.baseImage, bo.BaseImage)
			}
			if bo.ConcurrentBuilds < 1 {
				t.Errorf("ConcurrentBuilds must always be >= 1 for the ko builder")
			}
			if bo.Platform != test.wantPlatform {
				t.Errorf("wanted platform (%q), got (%q)", test.wantPlatform, bo.Platform)
			}
			if bo.UserAgent != version.UserAgentWithClient() {
				t.Errorf("need user agent for fetching the base image")
			}
			if bo.WorkingDirectory != test.workspace {
				t.Errorf("wanted WorkingDirectory (%q), got (%q)", test.workspace, bo.WorkingDirectory)
			}
		})
	}
}
