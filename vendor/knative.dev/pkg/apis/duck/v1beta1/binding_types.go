/*
Copyright 2020 The Knative Authors

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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"knative.dev/pkg/apis"
	"knative.dev/pkg/apis/duck"
	"knative.dev/pkg/tracker"
)

// +genduck
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Binding is a duck type that specifies the partial schema to which all
// Binding implementations should adhere.
type Binding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec BindingSpec `json:"spec"`
}

// Verify that Binding implements the appropriate interfaces.
var (
	_ duck.Implementable = (*Binding)(nil)
	_ duck.Populatable   = (*Binding)(nil)
	_ apis.Listable      = (*Binding)(nil)
)

// BindingSpec specifies the spec portion of the Binding partial-schema.
type BindingSpec struct {
	// Subject references the resource(s) whose "runtime contract" should be
	// augmented by Binding implementations.
	Subject tracker.Reference `json:"subject"`
}

// GetFullType implements duck.Implementable
func (*Binding) GetFullType() duck.Populatable {
	return &Binding{}
}

// Populate implements duck.Populatable
func (t *Binding) Populate() {
	t.Spec = BindingSpec{
		Subject: tracker.Reference{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Namespace:  "default",
			// Name and Selector are mutually exclusive,
			// but we fill them both in for this test.
			Name: "bazinga",
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"foo": "bar",
					"baz": "blah",
				},
			},
		},
	}
}

// GetListType implements apis.Listable
func (*Binding) GetListType() runtime.Object {
	return &BindingList{}
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BindingList is a list of Binding resources
type BindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Binding `json:"items"`
}
