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

package validator

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakedynclient "k8s.io/client-go/dynamic/fake"
	fakeclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/kubectl/pkg/scheme"

	"github.com/GoogleContainerTools/skaffold/proto/v1"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestConfigConnectorValidator(t *testing.T) {
	tests := []struct {
		description string
		status      map[string]interface{}
		expected    []Resource
	}{
		{
			description: "resource ready",
			status: map[string]interface{}{
				"status": "True",
				"type":   "Ready",
			},
			expected: []Resource{
				{
					kind:   "bar",
					name:   "foo1",
					status: "Current",
					ae:     &proto.ActionableErr{ErrCode: proto.StatusCode_STATUSCHECK_SUCCESS},
				},
			},
		},
		{
			description: "resource failed",
			status: map[string]interface{}{
				"status":  "False",
				"type":    "Ready",
				"message": "error",
			},
			expected: []Resource{
				{
					kind:   "bar",
					name:   "foo1",
					status: "InProgress",
					ae:     &proto.ActionableErr{ErrCode: proto.StatusCode_STATUSCHECK_CONFIG_CONNECTOR_FAILED, Message: "error"},
				},
			},
		},
		{
			description: "resource in progress",
			status: map[string]interface{}{
				"status": "False",
				"type":   "Ready",
			},
			expected: []Resource{
				{
					kind:   "bar",
					name:   "foo1",
					status: "InProgress",
					ae:     &proto.ActionableErr{ErrCode: proto.StatusCode_STATUSCHECK_CONFIG_CONNECTOR_IN_PROGRESS},
				},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			targetGvk := schema.GroupVersionKind{Version: "foo", Kind: "bar"}
			obj := []map[string]interface{}{
				{
					"kind":       "bar",
					"apiVersion": "foo",
					"metadata":   map[string]interface{}{"name": "foo1"},
					"status": map[string]interface{}{
						"conditions": []interface{}{
							test.status,
						},
					},
				},
			}
			res := &unstructured.UnstructuredList{}
			gvrs := map[schema.GroupVersionResource]string{}
			for _, item := range obj {
				obj := unstructured.Unstructured{Object: item}
				gvk := obj.GroupVersionKind()
				res.Items = append(res.Items, obj)
				gvr := schema.GroupVersionResource{Group: gvk.Group, Version: gvk.Version, Resource: fmt.Sprintf("%ss", gvk.Kind)}
				gvrs[gvr] = gvk.Kind + "List"
			}

			// Mock Kubernetes
			client := fakeclient.NewSimpleClientset(res)
			dynClient := fakedynclient.NewSimpleDynamicClientWithCustomListKinds(scheme.Scheme, gvrs, res)
			for _, item := range res.Items {
				gvk := item.GroupVersionKind()
				client.Resources = append(client.Resources, &metav1.APIResourceList{
					GroupVersion: item.GetObjectKind().GroupVersionKind().GroupVersion().String(),
					APIResources: []metav1.APIResource{{
						Kind:    gvk.Kind,
						Version: gvk.Version,
						Group:   gvk.Group,

						Name: fmt.Sprintf("%ss", gvk.Kind),
					}},
				})
				dynClient.Resources = append(dynClient.Resources, &metav1.APIResourceList{
					GroupVersion: item.GetObjectKind().GroupVersionKind().GroupVersion().String(),
					APIResources: []metav1.APIResource{{
						Kind: item.GetObjectKind().GroupVersionKind().String(),
						Name: item.GetName(),
					}},
				})
				client.Tracker().Add(&item)
			}
			v := NewConfigConnectorValidator(client, dynClient, targetGvk)
			r, err := v.Validate(context.Background(), "", metav1.ListOptions{})
			t.CheckNoError(err)

			t.CheckDeepEqual(test.expected, r, cmp.AllowUnexported(Resource{}), cmp.Comparer(func(x, y error) bool {
				if x == nil && y == nil {
					return true
				} else if x != nil && y != nil {
					return x.Error() == y.Error()
				}
				return false
			}), protocmp.Transform())
		})
	}
}
