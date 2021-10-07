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

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestStandalonePodsSelector(t *testing.T) {
	tests := []struct {
		description  string
		allPods      []v1.Pod
		expectedPods []v1.Pod
	}{
		{
			description: "pod don't exist in test namespace",
			allPods: []v1.Pod{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "foo-ns",
				},
				TypeMeta: metav1.TypeMeta{Kind: "Pod"}},
			},
			expectedPods: nil,
		},
		{
			description: "only deployment pods",
			allPods: []v1.Pod{{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "foo",
					Namespace:       "test",
					OwnerReferences: []metav1.OwnerReference{{UID: "", Controller: truePtr()}},
				},
				TypeMeta: metav1.TypeMeta{Kind: "Pod"},
			}},
			expectedPods: nil,
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
			expectedPods: []v1.Pod{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "test",
				},
				TypeMeta: metav1.TypeMeta{Kind: "Pod"},
			}},
		},
		{
			description: "standalone pods and deployment pods",
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
					Name:      "foo1",
					Namespace: "test",
				},
				TypeMeta: metav1.TypeMeta{Kind: "Pod"},
			}},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			var rs []runtime.Object
			for i := range test.allPods {
				p := test.allPods[i]
				rs = append(rs, &p)
			}
			f := fakekubeclientset.NewSimpleClientset(rs...)
			s := NewStandalonePodsSelector(f)
			actualPods, err := s.Select(context.Background(), "test", metav1.ListOptions{})
			t.CheckNoError(err)
			t.CheckDeepEqual(test.expectedPods, actualPods)
		})
	}
}
