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
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
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

func TestConsolidateTransformConfiguration(t *testing.T) {
	tests := []struct {
		description           string
		shouldErr             bool
		allowSchemaTransforms []latestV1.ResourceFilter
		denySchemaTransforms  []latestV1.ResourceFilter
		flagTransforms        latestV1.ResourceSelectorConfig
		expected              func(map[schema.GroupKind]latestV1.ResourceFilter, map[schema.GroupKind]latestV1.ResourceFilter) (map[schema.GroupKind]latestV1.ResourceFilter, map[schema.GroupKind]latestV1.ResourceFilter)
	}{
		{
			description: "verify schema transform configuration outprioritizes default hardcoded transform configuration",
			denySchemaTransforms: []latestV1.ResourceFilter{
				{
					GroupKind: "Deployment.apps",
				},
			},
			expected: func(allow map[schema.GroupKind]latestV1.ResourceFilter, deny map[schema.GroupKind]latestV1.ResourceFilter) (map[schema.GroupKind]latestV1.ResourceFilter, map[schema.GroupKind]latestV1.ResourceFilter) {
				// Deployment.apps removed from hardcoded allowlist
				delete(allow, schema.GroupKind{Group: "apps", Kind: "Deployment"})
				// Deployment.apps added to denylist
				deny[schema.GroupKind{Group: "apps", Kind: "Deployment"}] = latestV1.ResourceFilter{GroupKind: "Deployment.apps"}
				return allow, deny
			},
		},
		{
			description: "verify flag transform configuration outprioritizes schema transform configuration",
			flagTransforms: latestV1.ResourceSelectorConfig{
				Allow: []latestV1.ResourceFilter{
					{
						GroupKind: "Test.skaffold.dev",
					},
				},
			},
			denySchemaTransforms: []latestV1.ResourceFilter{
				{
					GroupKind: "Test.skaffold.dev",
				},
			},
			expected: func(allow map[schema.GroupKind]latestV1.ResourceFilter, deny map[schema.GroupKind]latestV1.ResourceFilter) (map[schema.GroupKind]latestV1.ResourceFilter, map[schema.GroupKind]latestV1.ResourceFilter) {
				// Test.skaffold.dev added to allowlist as flag config outprioritizes schema config
				allow[schema.GroupKind{Group: "skaffold.dev", Kind: "Test"}] = latestV1.ResourceFilter{GroupKind: "Test.skaffold.dev"}
				return allow, deny
			},
		},
		{
			description: "verify denylist outprioritizes allowlist transform configuration (for same config input source)",
			flagTransforms: latestV1.ResourceSelectorConfig{
				Allow: []latestV1.ResourceFilter{
					{
						GroupKind: "Test.skaffold.dev",
					},
				},
				Deny: []latestV1.ResourceFilter{
					{
						GroupKind: "Test.skaffold.dev",
					},
				},
			},
			expected: func(allow map[schema.GroupKind]latestV1.ResourceFilter, deny map[schema.GroupKind]latestV1.ResourceFilter) (map[schema.GroupKind]latestV1.ResourceFilter, map[schema.GroupKind]latestV1.ResourceFilter) {
				// Test.skaffold.dev added to denylist as deny config outprioritizes allow config for same priority config source (both flag config)
				deny[schema.GroupKind{Group: "skaffold.dev", Kind: "Test"}] = latestV1.ResourceFilter{GroupKind: "Test.skaffold.dev"}
				return allow, deny
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			// convert flagTransform struct to yaml
			buf, err := yaml.Marshal(test.flagTransforms)
			if err != nil {
				t.Fatalf("error marshalling flagTransforms test inputs: %v", err)
			}

			// denybuf, err := yaml.Marshal(test.denyFlagTransforms)
			// if err != nil {
			// t.Fatalf("error marshalling denyFlagTransforms test inputs: %v", err)
			// }

			flagTransformYAMLFile := t.TempFile("TestConsolidateTransformConfiguration", buf)

			cfg := &mockDeployConfig{
				transformAllowList: test.allowSchemaTransforms,
				transformDenyList:  test.denySchemaTransforms,
				transformRulesFile: flagTransformYAMLFile,
			}
			allowlist, denylist, err := ConsolidateTransformConfiguration(cfg)
			t.CheckError(test.shouldErr, err)

			copyAllow := map[schema.GroupKind]latestV1.ResourceFilter{}
			for k, v := range manifest.TransformAllowlist {
				copyAllow[k] = v
			}

			copyDeny := map[schema.GroupKind]latestV1.ResourceFilter{}
			for k, v := range manifest.TransformDenylist {
				copyDeny[k] = v
			}
			expectedAllowlist, expectedDenyList := test.expected(copyAllow, copyDeny)
			t.CheckDeepEqual(expectedAllowlist, allowlist)
			t.CheckDeepEqual(expectedDenyList, denylist)
		})
	}
}

type mockDeployConfig struct {
	runcontext.RunContext // Embedded to provide the default values.
	transformAllowList    []latestV1.ResourceFilter
	transformDenyList     []latestV1.ResourceFilter
	transformRulesFile    string
}

func (c *mockDeployConfig) ForceDeploy() bool                                     { return false }
func (c *mockDeployConfig) GetKubeConfig() string                                 { return "" }
func (c *mockDeployConfig) GetKubeContext() string                                { return "" }
func (c *mockDeployConfig) GetKubeNamespace() string                              { return "" }
func (c *mockDeployConfig) ConfigurationFile() string                             { return "" }
func (c *mockDeployConfig) PortForwardResources() []*latestV1.PortForwardResource { return nil }
func (c *mockDeployConfig) TransformAllowList() []latestV1.ResourceFilter {
	return c.transformAllowList
}
func (c *mockDeployConfig) TransformDenyList() []latestV1.ResourceFilter {
	return c.transformDenyList
}
func (c *mockDeployConfig) TransformRulesFile() string { return c.transformRulesFile }
