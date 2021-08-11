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

func TestPublishOptions(t *testing.T) {
	tests := []struct {
		description string
		ref         string
		pushImages  bool
		repo        string
		tag         string
	}{
		{
			description: "sideloaded image",
			ref:         "registry.example.com/repository/myapp1:tag1",
			pushImages:  false,
			repo:        "registry.example.com/repository/myapp1",
			tag:         "tag",
		},
		{
			description: "published image",
			ref:         "registry.example.com/repository/myapp2:tag2",
			pushImages:  true,
			repo:        "registry.example.com/repository/myapp2",
			tag:         "tag",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			dockerClient := fakeDockerAPIClient(test.ref, "imageID")
			po, err := publishOptions(test.ref, test.pushImages, dockerClient)
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
			if !test.pushImages && po.LocalDomain != test.repo {
				t.Errorf("wanted LocalDomain (%q), got (%q)", test.repo, po.DockerRepo)
			}
			if test.pushImages == po.Local {
				t.Errorf("Local (%v) should be the inverse of pushImages (%v)", po.Local, test.pushImages)
			}
			if test.pushImages != po.Push {
				t.Errorf("Push (%v) should match pushImages (%v)", po.Push, test.pushImages)
			}
			if len(po.Tags) != 1 && po.Tags[0] != test.tag {
				t.Errorf("wanted Tags (%+v), got (%+v)", []string{test.tag}, po.Tags)
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
