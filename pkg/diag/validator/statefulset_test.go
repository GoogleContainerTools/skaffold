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
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
	appsclient "k8s.io/client-go/kubernetes/typed/apps/v1"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestStatefulSetPodsSelector(t *testing.T) {
	tests := []struct {
		description  string
		allPods      []v1.Pod
		expectedPods []v1.Pod
	}{
		{
			description: "pod don't exist in test namespace",
			allPods: []v1.Pod{{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "foo",
					Namespace:       "foo-ns",
					OwnerReferences: []metav1.OwnerReference{{UID: "", Controller: truePtr()}},
				},
				TypeMeta: metav1.TypeMeta{Kind: "Pod"}},
			},
			expectedPods: nil,
		},
		{
			description: "only statefulset pods",
			allPods: []v1.Pod{{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "foo",
					Namespace:       "test",
					OwnerReferences: []metav1.OwnerReference{{UID: "", Controller: truePtr()}},
				},
				TypeMeta: metav1.TypeMeta{Kind: "Pod"},
			}},
			expectedPods: []v1.Pod{{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "foo",
					Namespace:       "test",
					OwnerReferences: []metav1.OwnerReference{{UID: "", Controller: truePtr()}},
				},
				TypeMeta: metav1.TypeMeta{Kind: "Pod"},
			}},
		},
		{
			description: "only standalone pods",
			allPods: []v1.Pod{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "test",
				},
				TypeMeta: metav1.TypeMeta{Kind: "Pod"},
			}},
			expectedPods: nil,
		},
		{
			description: "standalone pods and statefulset pods",
			allPods: []v1.Pod{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo1",
					Namespace: "test",
				},
				TypeMeta: metav1.TypeMeta{Kind: "Pod"},
			}, {
				ObjectMeta: metav1.ObjectMeta{
					Name:            "foo2",
					Namespace:       "test",
					OwnerReferences: []metav1.OwnerReference{{UID: "", Controller: truePtr()}},
				},
				TypeMeta: metav1.TypeMeta{Kind: "Pod"},
			}},
			expectedPods: []v1.Pod{{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "foo2",
					Namespace:       "test",
					OwnerReferences: []metav1.OwnerReference{{UID: "", Controller: truePtr()}},
				},
				TypeMeta: metav1.TypeMeta{Kind: "Pod"},
			}},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&getReplicaSet, func(_ *appsv1.Deployment, _ appsclient.AppsV1Interface) ([]*appsv1.ReplicaSet, []*appsv1.ReplicaSet, *appsv1.ReplicaSet, error) {
				return nil, nil, &appsv1.ReplicaSet{
					ObjectMeta: metav1.ObjectMeta{
						UID: types.UID(""),
					},
				}, nil
			})
			var rs []runtime.Object
			for i := range test.allPods {
				p := test.allPods[i]
				rs = append(rs, &p)
			}
			f := fakekubeclientset.NewSimpleClientset(rs...)
			s := NewStatefulSetPodsSelector(f, appsv1.StatefulSet{})
			actualPods, err := s.Select(context.Background(), "test", metav1.ListOptions{})
			t.CheckNoError(err)
			t.CheckDeepEqual(test.expectedPods, actualPods)
		})
	}
}
