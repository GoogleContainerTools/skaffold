/*
Copyright 2020 The Skaffold Authors

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

package build

import (
	"errors"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestCreateBuildArgsFromArtifacts(t *testing.T) {
	tests := []struct {
		description string
		r           ArtifactResolver
		deps        []*latest.ArtifactDependency
		expected    map[string]*string
		shouldErr   bool
	}{
		{
			description: "can resolve artifacts",
			r:           mockArtifactResolver{m: map[string]string{"img1": "tag1", "img2": "tag2", "img3": "tag3", "img4": "tag4"}},
			deps:        []*latest.ArtifactDependency{{ImageName: "img3", Alias: "alias3"}, {ImageName: "img4", Alias: "alias4"}},
			expected:    map[string]*string{"alias3": util.StringPtr("tag3"), "alias4": util.StringPtr("tag4")},
		},
		{
			description: "cannot resolve artifacts",
			r:           failingArtifactResolver{},
			deps:        []*latest.ArtifactDependency{{ImageName: "img3", Alias: "alias3"}, {ImageName: "img4", Alias: "alias4"}},
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			args, err := CreateBuildArgsFromArtifacts(test.deps, test.r)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, args)
		})
	}
}

type mockArtifactResolver struct {
	m map[string]string
}

func (r mockArtifactResolver) GetImageTag(imageName string) (string, error) {
	return r.m[imageName], nil
}

type failingArtifactResolver struct{}

func (failingArtifactResolver) GetImageTag(string) (string, error) {
	return "", errors.New("failed to retrieve image tag")
}
