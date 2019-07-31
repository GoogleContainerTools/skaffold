/*
 *
 * Copyright 2019 The Tekton Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 * /
 */

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

// Check that Task may be validated and defaulted.
var _ apis.Validatable = (*Condition)(nil)
var _ apis.Defaultable = (*Condition)(nil)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Condition declares a step that is used to gate the execution of a Task in a Pipeline.
// A condition execution (ConditionCheck) evaluates to either true or false
// +k8s:openapi-gen=true
type Condition struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata"`

	// Spec holds the desired state of the Condition from the client
	// +optional
	Spec ConditionSpec `json:"spec"`
}

// ConditionCheckStatus defines the observed state of ConditionCheck
type ConditionCheckStatus struct {
	duckv1beta1.Status `json:",inline"`

	// PodName is the name of the pod responsible for executing this condition check.
	PodName string `json:"podName"`

	// StartTime is the time the check is actually started.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is the time the check pod completed.
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Check describes the state of the check container.
	// +optional
	Check corev1.ContainerState `json:"check,omitempty"`
}

// ConditionSpec defines the desired state of the Condition
type ConditionSpec struct {
	// Check declares container whose exit code determines where a condition is true or false
	Check corev1.Container `json:"check,omitempty"`

	// Params is an optional set of parameters which must be supplied by the user when a Condition
	// is evaluated
	// +optional
	Params []ParamSpec `json:"params,omitempty"`
}

// ConditionCheck represents a single evaluation of a Condition step.
type ConditionCheck TaskRun

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ConditionList contains a list of Conditions
type ConditionList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Condition `json:"items"`
}

func NewConditionCheck(tr *TaskRun) *ConditionCheck {
	if tr == nil {
		return nil
	}

	cc := ConditionCheck(*tr)
	return &cc
}

// IsDone returns true if the ConditionCheck's status indicates that it is done.
func (cc *ConditionCheck) IsDone() bool {
	return !cc.Status.GetCondition(apis.ConditionSucceeded).IsUnknown()
}

// IsSuccessful returns true if the ConditionCheck's status indicates that it is done.
func (cc *ConditionCheck) IsSuccessful() bool {
	return cc.Status.GetCondition(apis.ConditionSucceeded).IsTrue()
}
