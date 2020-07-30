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

package kubernetes

import (
	"errors"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	fakedynclient "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func mockClient(resources *metav1.APIResourceList, objects ...runtime.Object) func() (kubernetes.Interface, error) {
	client := fakekubeclientset.NewSimpleClientset(objects...)
	client.Resources = append(client.Resources, resources)
	return func() (kubernetes.Interface, error) {
		return client, nil
	}
}

func mockDynamicClient(objects ...runtime.Object) func() (dynamic.Interface, error) {
	return func() (dynamic.Interface, error) {
		return fakedynclient.NewSimpleDynamicClient(scheme.Scheme, objects...), nil
	}
}

func TestTopLevelOwnerKey(t *testing.T) {
	apiResources := &metav1.APIResourceList{
		GroupVersion: "apps/v1",
		APIResources: []metav1.APIResource{
			{Kind: "Deployment", Name: "deployments"},
			{Kind: "ReplicaSet", Name: "replicasets"},
			{Kind: "Pod", Name: "pods"},
		},
	}

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod",
			Namespace: "ns",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "apps/v1",
				Kind:       "ReplicaSet",
				Name:       "rs",
			}},
		},
	}

	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rs",
			Namespace: "ns",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       "dep",
			}},
		},
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dep",
			Namespace: "ns",
		},
	}

	tests := []struct {
		description   string
		initialObject metav1.Object
		kind          string
		objects       []runtime.Object
		expected      string
		shouldErr     bool
	}{
		{
			description:   "owner is two levels up",
			initialObject: pod,
			kind:          "Pod",
			objects:       []runtime.Object{pod, rs, deployment},
			expected:      "Deployment-dep",
		},
		{
			description:   "object is owner",
			initialObject: deployment,
			kind:          "Deployment",
			objects:       []runtime.Object{pod, rs, deployment},
			expected:      "Deployment-dep",
		},
		{
			description:   "error, owner doesn't exist",
			initialObject: pod,
			kind:          "Pod",
			objects:       []runtime.Object{pod, rs},
			shouldErr:     true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&Client, mockClient(apiResources, test.objects...))
			t.Override(&DynamicClient, mockDynamicClient(test.objects...))

			actual, err := TopLevelOwnerKey(test.initialObject, test.kind)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, actual)
		})
	}
}

func TestTopLevelOwnerKeyFailToGetClient(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.Override(&Client, func() (kubernetes.Interface, error) { return nil, errors.New("BUG") })
		t.Override(&DynamicClient, mockDynamicClient())

		actual, err := TopLevelOwnerKey(nil, "")

		t.CheckErrorAndDeepEqual(true, err, "", actual)
	})
}

func TestTopLevelOwnerKeyFailToGetDynamicClient(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.Override(&Client, mockClient(nil))
		t.Override(&DynamicClient, func() (dynamic.Interface, error) { return nil, errors.New("BUG") })

		actual, err := TopLevelOwnerKey(nil, "")

		t.CheckErrorAndDeepEqual(true, err, "", actual)
	})
}
