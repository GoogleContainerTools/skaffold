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
	corev1 "k8s.io/api/core/v1"

	"github.com/knative/pkg/apis"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (t *Task) TaskSpec() TaskSpec {
	return t.Spec
}

func (t *Task) TaskMetadata() metav1.ObjectMeta {
	return t.ObjectMeta
}

func (t *Task) Copy() TaskInterface {
	return t.DeepCopy()
}

// TaskSpec defines the desired state of Task.
type TaskSpec struct {
	// Inputs is an optional set of parameters and resources which must be
	// supplied by the user when a Task is executed by a TaskRun.
	// +optional
	Inputs *Inputs `json:"inputs,omitempty"`
	// Outputs is an optional set of resources and results produced when this
	// Task is run.
	// +optional
	Outputs *Outputs `json:"outputs,omitempty"`

	// Steps are the steps of the build; each step is run sequentially with the
	// source mounted into /workspace.
	Steps []corev1.Container `json:"steps,omitempty"`

	// Volumes is a collection of volumes that are available to mount into the
	// steps of the build.
	Volumes []corev1.Volume `json:"volumes,omitempty"`

	// StepTemplate can be used as the basis for all step containers within the
	// Task, so that the steps inherit settings on the base container.
	StepTemplate *corev1.Container `json:"stepTemplate,omitempty"`

	// ContainerTemplate is the deprecated previous name of the StepTemplate
	// field (#977).
	ContainerTemplate *corev1.Container `json:"containerTemplate,omitempty"`
}

// Check that Task may be validated and defaulted.
var _ apis.Validatable = (*Task)(nil)
var _ apis.Defaultable = (*Task)(nil)

const (
	// TaskOutputImageDefaultDir is the default directory for output image resource,
	TaskOutputImageDefaultDir = "/builder/home/image-outputs"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Task represents a collection of sequential steps that are run as part of a
// Pipeline using a set of inputs and producing a set of outputs. Tasks execute
// when TaskRuns are created that provide the input parameters and resources and
// output resources the Task requires.
//
// +k8s:openapi-gen=true
type Task struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata"`

	// Spec holds the desired state of the Task from the client
	// +optional
	Spec TaskSpec `json:"spec"`
}

// Inputs are the requirements that a task needs to run a Build.
type Inputs struct {
	// Resources is a list of the input resources required to run the task.
	// Resources are represented in TaskRuns as bindings to instances of
	// PipelineResources.
	// +optional
	Resources []TaskResource `json:"resources,omitempty"`
	// Params is a list of input parameters required to run the task. Params
	// must be supplied as inputs in TaskRuns unless they declare a default
	// value.
	// +optional
	Params []ParamSpec `json:"params,omitempty"`
}

// TaskResource defines an input or output Resource declared as a requirement
// by a Task. The Name field will be used to refer to these Resources within
// the Task definition, and when provided as an Input, the Name will be the
// path to the volume mounted containing this Resource as an input (e.g.
// an input Resource named `workspace` will be mounted at `/workspace`).
type TaskResource struct {
	// Name declares the name by which a resource is referenced in the Task's
	// definition. Resources may be referenced by name in the definition of a
	// Task's steps.
	Name string `json:"name"`
	// Type is the type of this resource;
	Type PipelineResourceType `json:"type"`
	// TargetPath is the path in workspace directory where the task resource
	// will be copied.
	// +optional
	TargetPath string `json:"targetPath"`
	// Path to the index.json file for output container images.
	// +optional
	OutputImageDir string `json:"outputImageDir"`
}

// Outputs allow a task to declare what data the Build/Task will be producing,
// i.e. results such as logs and artifacts such as images.
type Outputs struct {
	// +optional
	Results []TestResult `json:"results,omitempty"`
	// +optional
	Resources []TaskResource `json:"resources,omitempty"`
}

// TestResult allows a task to specify the location where test logs
// can be found and what format they will be in.
type TestResult struct {
	// Name declares the name by which a result is referenced in the Task's
	// definition. Results may be referenced by name in the definition of a
	// Task's steps.
	Name string `json:"name"`
	// TODO: maybe this is an enum with types like "go test", "junit", etc.
	Format string `json:"format"`
	Path   string `json:"path"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TaskList contains a list of Task
type TaskList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Task `json:"items"`
}
