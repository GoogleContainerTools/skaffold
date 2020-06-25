/*
Copyright 2020 The Skaffold Authors

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
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/GoogleContainerTools/skaffold/proto"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNewResource(t *testing.T) {
	tests := []struct {
		description  string
		resource     objectWithMetadata
		expected     Resource
		expectedName string
	}{
		{
			description: "pod in default namespace",
			resource: &v1.Pod{
				TypeMeta: metav1.TypeMeta{Kind: "pod"},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "foo",
				},
			},
			expected:     Resource{"default", "pod", "foo", nil, Status(""), proto.ActionableErr{}},
			expectedName: "pod/foo",
		},
		{
			description: "pod in another namespace",
			resource: &v1.Pod{
				TypeMeta: metav1.TypeMeta{Kind: "pod"},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "bar",
				},
			},
			expected:     Resource{"test", "pod", "bar", nil, Status(""), proto.ActionableErr{}},
			expectedName: "test:pod/bar",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			actual := NewResourceFromObject(test.resource, Status(""), proto.ActionableErr{}, nil)
			t.CheckDeepEqual(test.expected, actual, cmp.AllowUnexported(Resource{}))
			t.CheckDeepEqual(test.expectedName, actual.String(), cmp.AllowUnexported(Resource{}))
		})
	}
}
