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

package label

import (
	"context"
	"testing"

	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	fakedynclient "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	fakeclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/types"
	kubernetesclient "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func mockClient(m kubernetes.Interface) func(string) (kubernetes.Interface, error) {
	return func(string) (kubernetes.Interface, error) {
		return m, nil
	}
}

func mockDynamicClient(m dynamic.Interface) func(string) (dynamic.Interface, error) {
	return func(string) (dynamic.Interface, error) {
		return m, nil
	}
}

func TestApplyLabels(t *testing.T) {
	tests := []struct {
		description    string
		existingLabels map[string]string
		appliedLabels  map[string]string
		expectedLabels map[string]string
	}{
		{
			description:    "set labels",
			existingLabels: map[string]string{},
			appliedLabels: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			expectedLabels: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			description: "add labels",
			existingLabels: map[string]string{
				"key0": "value0",
			},
			appliedLabels: map[string]string{
				"key0": "should-be-ignored",
				"key1": "value1",
			},
			expectedLabels: map[string]string{
				"key0": "value0",
				"key1": "value1",
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			dep := &v1.Deployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:   "foo",
					Labels: test.existingLabels,
				},
			}

			// Mock Kubernetes
			client := fakeclient.NewSimpleClientset(dep)
			client.Resources = append(client.Resources, &metav1.APIResourceList{
				GroupVersion: dep.APIVersion,
				APIResources: []metav1.APIResource{{
					Kind: dep.Kind,
					Name: "deployments",
				}},
			})
			t.Override(&kubernetesclient.Client, mockClient(client))
			dynClient := fakedynclient.NewSimpleDynamicClientWithCustomListKinds(scheme.Scheme, nil, dep)
			t.Override(&kubernetesclient.DynamicClient, mockDynamicClient(dynClient))

			// Patch labels
			Apply(context.Background(), test.appliedLabels, []types.Artifact{{Obj: dep}}, "")

			// Check modified value
			modified, err := dynClient.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}).Get(context.Background(), "foo", metav1.GetOptions{})
			t.CheckNoError(err)
			t.CheckDeepEqual(test.expectedLabels, modified.GetLabels())
		})
	}
}
