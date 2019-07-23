/*
Copyright 2019 The Tekton Authors.

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

package v1alpha1

import (
	"fmt"
	"time"

	"github.com/knative/pkg/apis"
	duckv1beta1 "github.com/knative/pkg/apis/duck/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Check that TaskRun may be validated and defaulted.
var _ apis.Validatable = (*TaskRun)(nil)
var _ apis.Defaultable = (*TaskRun)(nil)

// TaskRunSpec defines the desired state of TaskRun
type TaskRunSpec struct {
	// +optional
	Inputs TaskRunInputs `json:"inputs,omitempty"`
	// +optional
	Outputs TaskRunOutputs `json:"outputs,omitempty"`
	// +optional
	Results *Results `json:"results,omitempty"`
	// +optional
	ServiceAccount string `json:"serviceAccount,omitempty"`
	// no more than one of the TaskRef and TaskSpec may be specified.
	// +optional
	TaskRef *TaskRef `json:"taskRef,omitempty"`
	// +optional
	TaskSpec *TaskSpec `json:"taskSpec,omitempty"`
	// Used for cancelling a taskrun (and maybe more later on)
	// +optional
	Status TaskRunSpecStatus `json:"status,omitempty"`
	// Time after which the build times out. Defaults to 10 minutes.
	// Specified build timeout should be less than 24h.
	// Refer Go's ParseDuration documentation for expected format: https://golang.org/pkg/time/#ParseDuration
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`
	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// If specified, the pod's tolerations.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// If specified, the pod's scheduling constraints
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
}

// TaskRunSpecStatus defines the taskrun spec status the user can provide
type TaskRunSpecStatus string

const (
	// TaskRunSpecStatusCancelled indicates that the user wants to cancel the task,
	// if not already cancelled or terminated
	TaskRunSpecStatusCancelled = "TaskRunCancelled"
)

// TaskRunInputs holds the input values that this task was invoked with.
type TaskRunInputs struct {
	// +optional
	Resources []TaskResourceBinding `json:"resources,omitempty"`
	// +optional
	Params []Param `json:"params,omitempty"`
}

// TaskResourceBinding points to the PipelineResource that
// will be used for the Task input or output called Name. The optional Path field
// corresponds to a path on disk at which the Resource can be found (used when providing
// the resource via mounted volume, overriding the default logic to fetch the Resource).
type TaskResourceBinding struct {
	Name string `json:"name"`
	// no more than one of the ResourceRef and ResourceSpec may be specified.
	// +optional
	ResourceRef PipelineResourceRef `json:"resourceRef,omitempty"`
	// +optional
	ResourceSpec *PipelineResourceSpec `json:"resourceSpec,omitempty"`
	// +optional
	Paths []string `json:"paths,omitempty"`
}

// TaskRunOutputs holds the output values that this task was invoked with.
type TaskRunOutputs struct {
	// +optional
	Resources []TaskResourceBinding `json:"resources,omitempty"`
}

var taskRunCondSet = apis.NewBatchConditionSet()

// TaskRunStatus defines the observed state of TaskRun
type TaskRunStatus struct {
	duckv1beta1.Status `json:",inline"`

	// In #107 should be updated to hold the location logs have been uploaded to
	// +optional
	Results *Results `json:"results,omitempty"`

	// PodName is the name of the pod responsible for executing this task's steps.
	PodName string `json:"podName"`

	// StartTime is the time the build is actually started.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is the time the build completed.
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Steps describes the state of each build step container.
	// +optional
	Steps []StepState `json:"steps,omitempty"`
	// RetriesStatus contains the history of TaskRunStatus in case of a retry in order to keep record of failures.
	// All TaskRunStatus stored in RetriesStatus will have no date within the RetriesStatus as is redundant.
	// +optional
	RetriesStatus []TaskRunStatus `json:"retriesStatus,omitempty"`
	// Results from Resources built during the taskRun. currently includes
	// the digest of build container images
	// optional
	ResourcesResult []PipelineResourceResult `json:"resourcesResult,omitempty"`
}

// GetCondition returns the Condition matching the given type.
func (tr *TaskRunStatus) GetCondition(t apis.ConditionType) *apis.Condition {
	return taskRunCondSet.Manage(tr).GetCondition(t)
}

// InitializeConditions will set all conditions in taskRunCondSet to unknown for the TaskRun
// and set the started time to the current time
func (tr *TaskRunStatus) InitializeConditions() {
	if tr.StartTime.IsZero() {
		tr.StartTime = &metav1.Time{Time: time.Now()}
	}
	taskRunCondSet.Manage(tr).InitializeConditions()
}

// SetCondition sets the condition, unsetting previous conditions with the same
// type as necessary.
func (tr *TaskRunStatus) SetCondition(newCond *apis.Condition) {
	if newCond != nil {
		taskRunCondSet.Manage(tr).SetCondition(*newCond)
	}
}

// StepState reports the results of running a step in the Task.
type StepState struct {
	corev1.ContainerState
	Name string `json:"name,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TaskRun represents a single execution of a Task. TaskRuns are how the steps
// specified in a Task are executed; they specify the parameters and resources
// used to run the steps in a Task.
//
// +k8s:openapi-gen=true
type TaskRun struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	Spec TaskRunSpec `json:"spec,omitempty"`
	// +optional
	Status TaskRunStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TaskRunList contains a list of TaskRun
type TaskRunList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TaskRun `json:"items"`
}

// GetBuildPodRef for task
func (tr *TaskRun) GetBuildPodRef() corev1.ObjectReference {
	return corev1.ObjectReference{
		APIVersion: "v1",
		Kind:       "Pod",
		Namespace:  tr.Namespace,
		Name:       tr.Name,
	}
}

// GetPipelineRunPVCName for taskrun gets pipelinerun
func (tr *TaskRun) GetPipelineRunPVCName() string {
	if tr == nil {
		return ""
	}
	for _, ref := range tr.GetOwnerReferences() {
		if ref.Kind == pipelineRunControllerName {
			return fmt.Sprintf("%s-pvc", ref.Name)
		}
	}
	return ""
}

// HasPipelineRunOwnerReference returns true of TaskRun has
// owner reference of type PipelineRun
func (tr *TaskRun) HasPipelineRunOwnerReference() bool {
	for _, ref := range tr.GetOwnerReferences() {
		if ref.Kind == pipelineRunControllerName {
			return true
		}
	}
	return false
}

// IsDone returns true if the TaskRun's status indicates that it is done.
func (tr *TaskRun) IsDone() bool {
	return !tr.Status.GetCondition(apis.ConditionSucceeded).IsUnknown()
}

// HasStarted function check whether taskrun has valid start time set in its status
func (tr *TaskRun) HasStarted() bool {
	return tr.Status.StartTime != nil && !tr.Status.StartTime.IsZero()
}

// IsSuccessful returns true if the TaskRun's status indicates that it is done.
func (tr *TaskRun) IsSuccessful() bool {
	return tr.Status.GetCondition(apis.ConditionSucceeded).IsTrue()
}

// IsCancelled returns true if the TaskRun's spec status is set to Cancelled state
func (tr *TaskRun) IsCancelled() bool {
	return tr.Spec.Status == TaskRunSpecStatusCancelled
}

// GetRunKey return the taskrun key for timeout handler map
func (tr *TaskRun) GetRunKey() string {
	return fmt.Sprintf("%s/%s/%s", "TaskRun", tr.Namespace, tr.Name)
}
