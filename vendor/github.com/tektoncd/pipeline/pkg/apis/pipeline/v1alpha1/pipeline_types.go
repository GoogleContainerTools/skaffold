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
	"github.com/knative/pkg/apis"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PipelineSpec defines the desired state of Pipeline.
type PipelineSpec struct {
	// Resources declares the names and types of the resources given to the
	// Pipeline's tasks as inputs and outputs.
	Resources []PipelineDeclaredResource `json:"resources,omitempty"`
	// Tasks declares the graph of Tasks that execute when this Pipeline is run.
	Tasks []PipelineTask `json:"tasks,omitempty"`
	// Params declares a list of input parameters that must be supplied when
	// this Pipeline is run.
	Params []ParamSpec `json:"params,omitempty"`
}

// PipelineStatus does not contain anything because Pipelines on their own
// do not have a status, they just hold data which is later used by a
// PipelineRun.
type PipelineStatus struct {
}

// Check that Pipeline may be validated and defaulted.
var _ apis.Validatable = (*Pipeline)(nil)
var _ apis.Defaultable = (*Pipeline)(nil)

// TaskKind defines the type of Task used by the pipeline.
type TaskKind string

const (
	// NamespacedTaskKind indicates that the task type has a namepace scope.
	NamespacedTaskKind TaskKind = "Task"
	// ClusterTaskKind indicates that task type has a cluster scope.
	ClusterTaskKind TaskKind = "ClusterTask"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Pipeline describes a list of Tasks to execute. It expresses how outputs
// of tasks feed into inputs of subsequent tasks.
// +k8s:openapi-gen=true
type Pipeline struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec holds the desired state of the Pipeline from the client
	// +optional
	Spec PipelineSpec `json:"spec"`
	// Status communicates the observed state of the Pipeline from the
	// controller.
	// +optional
	Status PipelineStatus `json:"status"`
}

// PipelineTask defines a task in a Pipeline, passing inputs from both
// Params and from the output of previous tasks.
type PipelineTask struct {
	// Name is the name of this task within the context of a Pipeline. Name is
	// used as a coordinate with the `from` and `runAfter` fields to establish
	// the execution order of tasks relative to one another.
	Name string `json:"name,omitempty"`
	// TaskRef is a reference to a task definition.
	TaskRef TaskRef `json:"taskRef"`

	// Retries represents how many times this task should be retried in case of task failure: ConditionSucceeded set to False
	// +optional
	Retries int `json:"retries,omitempty"`

	// RunAfter is the list of PipelineTask names that should be executed before
	// this Task executes. (Used to force a specific ordering in graph execution.)
	// +optional
	RunAfter []string `json:"runAfter,omitempty"`

	// Resources declares the resources given to this task as inputs and
	// outputs.
	// +optional
	Resources *PipelineTaskResources `json:"resources,omitempty"`
	// Parameters declares parameters passed to this task.
	// +optional
	Params []Param `json:"params,omitempty"`
}

// PipelineTaskParam is used to provide arbitrary string parameters to a Task.
type PipelineTaskParam struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// PipelineDeclaredResource is used by a Pipeline to declare the types of the
// PipelineResources that it will required to run and names which can be used to
// refer to these PipelineResources in PipelineTaskResourceBindings.
type PipelineDeclaredResource struct {
	// Name is the name that will be used by the Pipeline to refer to this resource.
	// It does not directly correspond to the name of any PipelineResources Task
	// inputs or outputs, and it does not correspond to the actual names of the
	// PipelineResources that will be bound in the PipelineRun.
	Name string `json:"name"`
	// Type is the type of the PipelineResource.
	Type PipelineResourceType `json:"type"`
}

// PipelineTaskResources allows a Pipeline to declare how its DeclaredPipelineResources
// should be provided to a Task as its inputs and outputs.
type PipelineTaskResources struct {
	// Inputs holds the mapping from the PipelineResources declared in
	// DeclaredPipelineResources to the input PipelineResources required by the Task.
	Inputs []PipelineTaskInputResource `json:"inputs,omitempty"`
	// Outputs holds the mapping from the PipelineResources declared in
	// DeclaredPipelineResources to the input PipelineResources required by the Task.
	Outputs []PipelineTaskOutputResource `json:"outputs,omitempty"`
}

// PipelineTaskInputResource maps the name of a declared PipelineResource input
// dependency in a Task to the resource in the Pipeline's DeclaredPipelineResources
// that should be used. This input may come from a previous task.
type PipelineTaskInputResource struct {
	// Name is the name of the PipelineResource as declared by the Task.
	Name string `json:"name"`
	// Resource is the name of the DeclaredPipelineResource to use.
	Resource string `json:"resource"`
	// From is the list of PipelineTask names that the resource has to come from.
	// (Implies an ordering in the execution graph.)
	// +optional
	From []string `json:"from,omitempty"`
}

// PipelineTaskOutputResource maps the name of a declared PipelineResource output
// dependency in a Task to the resource in the Pipeline's DeclaredPipelineResources
// that should be used.
type PipelineTaskOutputResource struct {
	// Name is the name of the PipelineResource as declared by the Task.
	Name string `json:"name"`
	// Resource is the name of the DeclaredPipelienResource to use.
	Resource string `json:"resource"`
}

// TaskRef can be used to refer to a specific instance of a task.
// Copied from CrossVersionObjectReference: https://github.com/kubernetes/kubernetes/blob/169df7434155cbbc22f1532cba8e0a9588e29ad8/pkg/apis/autoscaling/types.go#L64
type TaskRef struct {
	// Name of the referent; More info: http://kubernetes.io/docs/user-guide/identifiers#names
	Name string `json:"name,omitempty"`
	// TaskKind inficates the kind of the task, namespaced or cluster scoped.
	Kind TaskKind `json:"kind,omitempty"`
	// API version of the referent
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PipelineList contains a list of Pipeline
type PipelineList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Pipeline `json:"items"`
}
