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

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/version"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestPublishOptions(t *testing.T) {
	tests := []struct {
		description          string
		ref                  string
		repo                 string
		insecureRegistries   map[string]bool
		pushImages           bool
		wantInsecureRegistry bool
		wantTag              string
	}{
		{
			description: "sideloaded image",
			ref:         "registry.example.com/repository/myapp1:tag1",
			repo:        "registry.example.com/repository/myapp1",
			pushImages:  false,
			wantTag:     "tag1",
		},
		{
			description: "published image",
			ref:         "registry.example.com/repository/myapp2:tag2",
			repo:        "registry.example.com/repository/myapp2",
			pushImages:  true,
			wantTag:     "tag2",
		},
		{
			description:          "insecure registry",
			ref:                  "insecure.registry.example.com/repository/myapp3:tag3",
			repo:                 "insecure.registry.example.com/repository/myapp3",
			insecureRegistries:   map[string]bool{"insecure.registry.example.com": true},
			pushImages:           true,
			wantInsecureRegistry: true,
			wantTag:              "tag3",
		},
		{
			description:          "secure registry",
			ref:                  "secure.registry.example.com/repository/myapp4:tag4",
			repo:                 "secure.registry.example.com/repository/myapp4",
			insecureRegistries:   map[string]bool{"insecure.registry.example.com": true},
			pushImages:           true,
			wantInsecureRegistry: false,
			wantTag:              "tag4",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			dockerClient := fakeDockerAPIClient(test.ref, "imageID")
			po, err := publishOptions(test.ref, test.pushImages, dockerClient, test.insecureRegistries)
			t.CheckNoError(err)
			if !po.Bare || po.BaseImportPaths || po.PreserveImportPaths {
				t.Errorf("use ko's Bare naming option as it allow for arbitrary image names")
			}
			if po.DockerClient != dockerClient {
				t.Errorf("use provided docker client")
			}
			if test.pushImages && po.DockerRepo != test.repo {
				t.Errorf("wanted DockerRepo (%q), got (%q)", test.repo, po.DockerRepo)
			}
			if test.wantInsecureRegistry != po.InsecureRegistry {
				t.Errorf("wanted InsecureRegistry (%v), got (%v)", test.wantInsecureRegistry, po.InsecureRegistry)
			}
			if !test.pushImages && po.LocalDomain != test.repo {
				t.Errorf("wanted LocalDomain (%q), got (%q)", test.repo, po.LocalDomain)
			}
			if test.pushImages && po.LocalDomain != "" {
				t.Errorf("wanted zero value for LocalDomain, got (%q)", po.LocalDomain)
			}
			if test.pushImages == po.Local {
				t.Errorf("Local (%v) should be the inverse of pushImages (%v)", po.Local, test.pushImages)
			}
			if test.pushImages != po.Push {
				t.Errorf("Push (%v) should match pushImages (%v)", po.Push, test.pushImages)
			}
			if po.Tags[0] != test.wantTag {
				t.Errorf("wanted Tags (%+v), got (%+v)", []string{test.wantTag}, po.Tags)
			}
			if po.UserAgent != version.UserAgentWithClient() {
				t.Errorf("wanted UserAgent (%s), got (%s)", version.UserAgentWithClient(), po.UserAgent)
			}
		})
	}
}

func fakeDockerAPIClient(ref string, imageID string) *testutil.FakeAPIClient {
	return (&testutil.FakeAPIClient{}).Add(ref, imageID)
}
