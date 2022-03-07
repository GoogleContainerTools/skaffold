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

package util

import (
	"testing"

	v1 "k8s.io/api/core/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestConsolidateNamespaces(t *testing.T) {
	tests := []struct {
		description   string
		oldNamespaces []string
		newNamespaces []string
		expected      []string
	}{
		{
			description:   "update namespace when not present in runContext",
			oldNamespaces: []string{"test"},
			newNamespaces: []string{"another"},
			expected:      []string{"another", "test"},
		},
		{
			description:   "update namespace with duplicates should not return duplicate",
			oldNamespaces: []string{"test", "foo"},
			newNamespaces: []string{"another", "foo", "another"},
			expected:      []string{"another", "foo", "test"},
		},
		{
			description:   "update namespaces when namespaces is empty",
			oldNamespaces: []string{"test", "foo"},
			newNamespaces: []string{},
			expected:      []string{"test", "foo"},
		},
		{
			description:   "update namespaces when runcontext namespaces is empty",
			oldNamespaces: []string{},
			newNamespaces: []string{"test", "another"},
			expected:      []string{"another", "test"},
		},
		{
			description:   "update namespaces when both namespaces and runcontext namespaces is empty",
			oldNamespaces: []string{},
			newNamespaces: []string{},
			expected:      []string{},
		},
		{
			description:   "update namespace when runcontext namespace has an empty string",
			oldNamespaces: []string{""},
			newNamespaces: []string{"another"},
			expected:      []string{"another"},
		},
		{
			description:   "update namespace when namespace is empty string",
			oldNamespaces: []string{"test"},
			newNamespaces: []string{""},
			expected:      []string{"test"},
		},
		{
			description:   "update namespace when namespace is empty string and runContext is empty",
			oldNamespaces: []string{},
			newNamespaces: []string{""},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			ns := ConsolidateNamespaces(test.oldNamespaces, test.newNamespaces)

			t.CheckDeepEqual(test.expected, ns)
		})
	}
}

func TestAddTagsToPodSelector(t *testing.T) {
	tests := []struct {
		description       string
		artifacts         []graph.Artifact
		deployerArtifacts []graph.Artifact
		expectedImages    []string
	}{
		{
			description: "empty image list",
		},
		{
			description: "non-matching image results in empty list",
			artifacts: []graph.Artifact{
				{
					ImageName: "my-image",
					Tag:       "my-image-tag",
				},
			},
			deployerArtifacts: []graph.Artifact{
				{
					ImageName: "not-my-image",
				},
			},
		},
		{
			description: "matching images appear in list",
			artifacts: []graph.Artifact{
				{
					ImageName: "my-image1",
					Tag:       "registry.example.com/repo/my-image1:tag1",
				},
				{
					ImageName: "my-image2",
					Tag:       "registry.example.com/repo/my-image2:tag2",
				},
				{
					ImageName: "image-not-in-deployer",
					Tag:       "registry.example.com/repo/my-image3:tag3",
				},
			},
			deployerArtifacts: []graph.Artifact{
				{
					ImageName: "my-image1",
				},
				{
					ImageName: "my-image2",
				},
			},
			expectedImages: []string{
				"registry.example.com/repo/my-image1:tag1",
				"registry.example.com/repo/my-image2:tag2",
			},
		},
		{
			description: "images from manifest files with ko:// scheme prefix are sanitized before matching",
			artifacts: []graph.Artifact{
				{
					ImageName: "ko://git.example.com/Foo/bar",
					Tag:       "registry.example.com/repo/git.example.com/foo/bar:tag",
				},
			},
			deployerArtifacts: []graph.Artifact{
				{
					ImageName: "git.example.com/foo/bar",
					Tag:       "ko://git.example.com/Foo/bar",
				},
			},
			expectedImages: []string{
				"registry.example.com/repo/git.example.com/foo/bar:tag",
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			podSelector := kubernetes.NewImageList()
			AddTagsToPodSelector(test.artifacts, test.deployerArtifacts, podSelector)
			for _, expectedImage := range test.expectedImages {
				if exists := podSelector.Select(&v1.Pod{
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{Image: expectedImage},
						},
					},
				}); !exists {
					t.Errorf("expected image list to contain %s", expectedImage)
				}
			}
		})
	}
}
