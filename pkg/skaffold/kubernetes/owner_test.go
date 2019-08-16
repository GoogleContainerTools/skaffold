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
	"testing"

	v1 "k8s.io/api/core/v1"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/GoogleContainerTools/skaffold/testutil"
	"k8s.io/client-go/kubernetes"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
)

func mockClient(m kubernetes.Interface) func() (kubernetes.Interface, error) {
	return func() (kubernetes.Interface, error) {
		return m, nil
	}
}

func TestTopLevelOwnerKey(t *testing.T) {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod",
			Namespace: "ns",
			OwnerReferences: []metav1.OwnerReference{
				{
					Name: "rs",
					Kind: "ReplicaSet",
				},
			},
		},
	}

	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rs",
			Namespace: "ns",
			OwnerReferences: []metav1.OwnerReference{
				{
					Name: "dep",
					Kind: "Deployment",
				},
			},
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
	}{
		{
			description:   "owner is two levels up",
			initialObject: pod,
			kind:          "Pod",
			objects:       []runtime.Object{pod, rs, deployment},
			expected:      "Deployment-dep",
		}, {
			description:   "object is owner",
			initialObject: deployment,
			kind:          "Deployment",
			objects:       []runtime.Object{pod, rs, deployment},
			expected:      "Deployment-dep",
		}, {
			description:   "error, owner doesn't exist",
			initialObject: pod,
			kind:          "Pod",
			objects:       []runtime.Object{pod, rs},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			client := fakekubeclientset.NewSimpleClientset(test.objects...)
			t.Override(&getClientSet, mockClient(client))
			actual := TopLevelOwnerKey(test.initialObject, test.kind)
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

func TestOwnerMetaObject(t *testing.T) {
	tests := []struct {
		description string
		or          metav1.OwnerReference
		objects     []runtime.Object
		expected    metav1.Object
	}{
		{
			description: "getting a deployment",
			or: metav1.OwnerReference{
				Kind: "Deployment",
				Name: "dep",
			},
			objects: []runtime.Object{
				&v1.Service{},
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dep",
						Namespace: "ns",
					},
				},
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dep",
						Namespace: "ns2",
					},
				},
			},
			expected: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dep",
					Namespace: "ns",
				},
			},
		}, {
			description: "getting a replica set",
			or: metav1.OwnerReference{
				Kind: "ReplicaSet",
				Name: "rs",
			},
			objects: []runtime.Object{
				&v1.Service{},
				&appsv1.ReplicaSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "rs",
						Namespace: "ns",
					},
				},
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dep",
						Namespace: "ns2",
					},
				},
			},
			expected: &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rs",
					Namespace: "ns",
				},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			objs := make([]runtime.Object, len(test.objects))
			for i, s := range test.objects {
				objs[i] = s
			}
			client := fakekubeclientset.NewSimpleClientset(objs...)
			t.Override(&getClientSet, mockClient(client))
			actual, err := ownerMetaObject("ns", test.or)
			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}
