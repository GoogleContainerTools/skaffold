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

package deploy

import (
	"testing"

	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	fakedynclient "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	fakeclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"

	k8s "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func mockClient(m kubernetes.Interface) func() (kubernetes.Interface, error) {
	return func() (kubernetes.Interface, error) {
		return m, nil
	}
}

func mockDynamicClient(m dynamic.Interface) func() (dynamic.Interface, error) {
	return func() (dynamic.Interface, error) {
		return m, nil
	}
}

func TestLabelDeployResults(t *testing.T) {
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
			t.Override(&k8s.Client, mockClient(client))
			dynClient := fakedynclient.NewSimpleDynamicClient(scheme.Scheme, dep)
			t.Override(&k8s.DynamicClient, mockDynamicClient(dynClient))

			// Patch labels
			labelDeployResults(test.appliedLabels, []Artifact{{Obj: dep}})

			// Check modified value
			modified, err := dynClient.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}).Get("foo", metav1.GetOptions{})
			t.CheckNoError(err)
			t.CheckDeepEqual(test.expectedLabels, modified.GetLabels())
		})
	}
}
