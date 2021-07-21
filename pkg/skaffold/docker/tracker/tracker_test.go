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

package tracker

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

type ArtifactIDPair struct {
	artifact graph.Artifact
	id       string
}

func TestDeployedContainers(t *testing.T) {
	tests := []struct {
		name               string
		containers         []ArtifactIDPair
		expectedContainers []string
	}{
		{
			name:               "one container",
			containers:         []ArtifactIDPair{{artifact: graph.Artifact{ImageName: "image1"}, id: "deadbeef"}},
			expectedContainers: []string{"deadbeef"},
		},
		{
			name: "two containers",
			containers: []ArtifactIDPair{
				{artifact: graph.Artifact{ImageName: "image1"}, id: "deadbeef"},
				{artifact: graph.Artifact{ImageName: "image2"}, id: "foobar"},
			},
			expectedContainers: []string{"deadbeef", "foobar"},
		},
		{
			name: "adding the same artifact overwrites previous ID",
			containers: []ArtifactIDPair{
				{artifact: graph.Artifact{ImageName: "image1"}, id: "this will be ignored"},
				{artifact: graph.Artifact{ImageName: "image1"}, id: "deadbeef"},
				{artifact: graph.Artifact{ImageName: "image2"}, id: "foobar"},
			},
			expectedContainers: []string{"deadbeef", "foobar"},
		},
		{
			name: "tags are ignored (keyed only on image name)",
			containers: []ArtifactIDPair{
				{artifact: graph.Artifact{ImageName: "image1", Tag: "image1:tag1"}, id: "this will be ignored"},
				{artifact: graph.Artifact{ImageName: "image1", Tag: "image1:tag2"}, id: "deadbeef"},
				{artifact: graph.Artifact{ImageName: "image2"}, id: "foobar"},
			},
			expectedContainers: []string{"deadbeef", "foobar"},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			tracker := NewContainerTracker()
			for _, pair := range test.containers {
				tracker.Add(pair.artifact, pair.id)
			}
			t.CheckDeepEqual(test.expectedContainers, tracker.DeployedContainers())
		})
	}
}

func TestDeployedContainerForImage(t *testing.T) {
	tests := []struct {
		name       string
		containers []ArtifactIDPair
		target     string
		expected   string
	}{
		{
			name:       "one container",
			containers: []ArtifactIDPair{{artifact: graph.Artifact{ImageName: "image1"}, id: "deadbeef"}},
			target:     "image1",
			expected:   "deadbeef",
		},
		{
			name: "two containers, retrieve the second",
			containers: []ArtifactIDPair{
				{artifact: graph.Artifact{ImageName: "image1"}, id: "deadbeef"},
				{artifact: graph.Artifact{ImageName: "image2"}, id: "foobar"},
			},
			target:   "image2",
			expected: "foobar",
		},
		{
			name: "adding the same artifact overwrites previous ID",
			containers: []ArtifactIDPair{
				{artifact: graph.Artifact{ImageName: "image1"}, id: "this will be ignored"},
				{artifact: graph.Artifact{ImageName: "image1"}, id: "deadbeef"},
				{artifact: graph.Artifact{ImageName: "image2"}, id: "foobar"},
			},
			target:   "image1",
			expected: "deadbeef",
		},
		{
			name: "tags are ignored (keyed only on image name)",
			containers: []ArtifactIDPair{
				{artifact: graph.Artifact{ImageName: "image1", Tag: "image1:tag1"}, id: "this will be ignored"},
				{artifact: graph.Artifact{ImageName: "image1", Tag: "image1:tag2"}, id: "deadbeef"},
				{artifact: graph.Artifact{ImageName: "image2"}, id: "foobar"},
			},
			target:   "image1",
			expected: "deadbeef",
		},
		{
			name:       "untracked image returns nothing",
			containers: []ArtifactIDPair{{artifact: graph.Artifact{ImageName: "image1"}, id: "deadbeef"}},
			target:     "bogus",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			tracker := NewContainerTracker()
			for _, pair := range test.containers {
				tracker.Add(pair.artifact, pair.id)
			}
			t.CheckDeepEqual(test.expected, tracker.DeployedContainerForImage(test.target))
		})
	}
}

func TestArtifactForContainer(t *testing.T) {
	tests := []struct {
		name       string
		containers []ArtifactIDPair
		target     string
		expected   graph.Artifact
	}{
		{
			name:       "one container",
			containers: []ArtifactIDPair{{artifact: graph.Artifact{ImageName: "image1"}, id: "deadbeef"}},
			target:     "deadbeef",
			expected:   graph.Artifact{ImageName: "image1"},
		},
		{
			name: "two containers, retrieve the second",
			containers: []ArtifactIDPair{
				{artifact: graph.Artifact{ImageName: "image1"}, id: "deadbeef"},
				{artifact: graph.Artifact{ImageName: "image2"}, id: "foobar"},
			},
			target:   "foobar",
			expected: graph.Artifact{ImageName: "image2"},
		},
		{
			name: "adding the same artifact overwrites previous ID",
			containers: []ArtifactIDPair{
				{artifact: graph.Artifact{ImageName: "image1"}, id: "this will be ignored"},
				{artifact: graph.Artifact{ImageName: "image1"}, id: "deadbeef"},
				{artifact: graph.Artifact{ImageName: "image2"}, id: "foobar"},
			},
			target:   "deadbeef",
			expected: graph.Artifact{ImageName: "image1"},
		},
		{
			name: "tags are updated on artifact",
			containers: []ArtifactIDPair{
				{artifact: graph.Artifact{ImageName: "image1", Tag: "image1:tag1"}, id: "deadbeef"},
				{artifact: graph.Artifact{ImageName: "image1", Tag: "image1:tag2"}, id: "deadbeef"},
				{artifact: graph.Artifact{ImageName: "image2"}, id: "foobar"},
			},
			target:   "deadbeef",
			expected: graph.Artifact{ImageName: "image1", Tag: "image1:tag2"},
		},
		{
			name:       "untracked id returns empty artifact",
			containers: []ArtifactIDPair{{artifact: graph.Artifact{ImageName: "image1"}, id: "deadbeef"}},
			target:     "bogus",
			expected:   graph.Artifact{},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			tracker := NewContainerTracker()
			for _, pair := range test.containers {
				tracker.Add(pair.artifact, pair.id)
			}
			t.CheckDeepEqual(test.expected, tracker.ArtifactForContainer(test.target))
		})
	}
}
