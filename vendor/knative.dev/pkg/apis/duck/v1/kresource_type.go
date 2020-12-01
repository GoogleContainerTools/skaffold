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

package v1

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis/duck/ducktypes"

	"knative.dev/pkg/apis"
)

// KRShaped is an interface for retrieving the duck elements of an arbitrary resource.
type KRShaped interface {
	metav1.Object
	schema.ObjectKind

	GetStatus() *Status

	GetConditionSet() apis.ConditionSet
}

// Asserts KResource conformance with KRShaped
var _ KRShaped = (*KResource)(nil)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KResource is a skeleton type wrapping Conditions in the manner we expect
// resource writers defining compatible resources to embed it.  We will
// typically use this type to deserialize Conditions ObjectReferences and
// access the Conditions data.  This is not a real resource.
type KResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status Status `json:"status"`
}

// Populate implements duck.Populatable
func (t *KResource) Populate() {
	t.Status.ObservedGeneration = 42
	t.Status.Conditions = Conditions{{
		// Populate ALL fields
		Type:               "Birthday",
		Status:             corev1.ConditionTrue,
		LastTransitionTime: apis.VolatileTime{Inner: metav1.NewTime(time.Date(1984, 02, 28, 18, 52, 00, 00, time.UTC))},
		Reason:             "Celebrate",
		Message:            "n3wScott, find your party hat :tada:",
	}}
}

// Verify KResource resources meet duck contracts.
var (
	_ apis.Listable         = (*KResource)(nil)
	_ ducktypes.Populatable = (*KResource)(nil)
)

// GetListType implements apis.Listable
func (*KResource) GetListType() runtime.Object {
	return &KResourceList{}
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KResourceList is a list of KResource resources
type KResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []KResource `json:"items"`
}

// GetStatus retrieves the status of the KResource. Implements the KRShaped interface.
func (t *KResource) GetStatus() *Status {
	return &t.Status
}

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (t *KResource) GetConditionSet() apis.ConditionSet {
	// Note: KResources are unmarshalled from existing resources. This will only work properly for resources that
	// have already been initialized to their type.
	if cond := t.Status.GetCondition(apis.ConditionSucceeded); cond != nil {
		return apis.NewBatchConditionSet()
	}
	return apis.NewLivingConditionSet()
}
