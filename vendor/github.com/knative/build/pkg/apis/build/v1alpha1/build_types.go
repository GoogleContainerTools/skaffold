/*
Copyright 2018 The Knative Authors

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
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Build represents a build of a container image. A Build is made up of a
// source, and a set of steps. Steps can mount volumes to share data between
// themselves. A build may be created by instantiating a BuildTemplate.
type Build struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BuildSpec   `json:"spec"`
	Status BuildStatus `json:"status"`
}

// BuildSpec is the spec for a Build resource.
type BuildSpec struct {
	// TODO: Generation does not work correctly with CRD. They are scrubbed
	// by the APIserver (https://github.com/kubernetes/kubernetes/issues/58778)
	// So, we add Generation here. Once that gets fixed, remove this and use
	// ObjectMeta.Generation instead.
	// +optional
	Generation int64 `json:"generation,omitempty"`

	// Source specifies the input to the build.
	Source *SourceSpec `json:"source,omitempty"`

	// Steps are the steps of the build; each step is run sequentially with the
	// source mounted into /workspace.
	Steps []corev1.Container `json:"steps,omitempty"`

	// Volumes is a collection of volumes that are available to mount into the
	// steps of the build.
	Volumes []corev1.Volume `json:"volumes,omitempty"`

	// The name of the service account as which to run this build.
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// Template, if specified, references a BuildTemplate resource to use to
	// populate fields in the build, and optional Arguments to pass to the
	// template.
	Template *TemplateInstantiationSpec `json:"template,omitempty"`

	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
}

// TemplateInstantiationSpec specifies how a BuildTemplate is instantiated into
// a Build.
type TemplateInstantiationSpec struct {
	// Name references the BuildTemplate resource to use.
	//
	// The template is assumed to exist in the Build's namespace.
	Name string `json:"name"`

	// Arguments, if specified, lists values that should be applied to the
	// parameters specified by the template.
	Arguments []ArgumentSpec `json:"arguments,omitempty"`

	// Env, if specified will provide variables to all build template steps.
	// This will override any of the template's steps environment variables.
	Env []corev1.EnvVar `json:"env,omitempty"`
}

// ArgumentSpec defines the actual values to use to populate a template's
// parameters.
type ArgumentSpec struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	// TODO(jasonhall): ValueFrom?
}

// SourceSpec defines the input to the Build
type SourceSpec struct {
	Git    *GitSourceSpec    `json:"git,omitempty"`
	GCS    *GCSSourceSpec    `json:"gcs,omitempty"`
	Custom *corev1.Container `json:"custom,omitempty"`
}

// GitSourceSpec describes a Git repo source input to the Build.
type GitSourceSpec struct {
	// URL of the Git repository to clone from.
	Url string `json:"url"`

	// Git revision (branch, tag, commit SHA or ref) to clone.  See
	// https://git-scm.com/docs/gitrevisions#_specifying_revisions for more
	// information.
	Revision string `json:"revision"`
}

// GCSSourceSpec describes source input to the Build in the form of an archive,
// or a source manifest describing files to fetch.
type GCSSourceSpec struct {
	Type     GCSSourceType `json:"type,omitempty"`
	Location string        `json:"location,omitempty"`
}

type GCSSourceType string

const (
	GCSArchive  GCSSourceType = "Archive"
	GCSManifest GCSSourceType = "Manifest"
)

type BuildProvider string

const (
	// GoogleBuildProvider indicates that this build was performed with Google Container Builder.
	GoogleBuildProvider BuildProvider = "Google"
	// ClusterBuildProvider indicates that this build was performed on-cluster.
	ClusterBuildProvider BuildProvider = "Cluster"
)

// BuildStatus is the status for a Build resource
type BuildStatus struct {
	Builder BuildProvider `json:"builder,omitempty"`

	// Additional information based on the Builder executing this build.
	Cluster *ClusterSpec `json:"cluster,omitempty"`
	Google  *GoogleSpec  `json:"google,omitempty"`

	// Information about the execution of the build.
	StartTime      metav1.Time `json:"startTime,omitEmpty"`
	CompletionTime metav1.Time `json:"completionTime,omitEmpty"`

	// Parallel list to spec.Containers
	StepStates []corev1.ContainerState `json:"stepStates,omitEmpty"`
	Conditions []BuildCondition        `json:"conditions,omitempty"`
}

type ClusterSpec struct {
	Namespace string `json:"namespace"`
	PodName   string `json:"podName"`
}

type GoogleSpec struct {
	Operation string `json:"operation"`
}

type BuildConditionType string

const (
	// BuildSucceeded is set when the build is running, and becomes True
	// when the build finishes successfully.
	//
	// If the build is ongoing, its status will be Unknown. If it fails,
	// its status will be False.
	BuildSucceeded BuildConditionType = "Succeeded"
)

// BuildCondition defines a readiness condition for a Build.
// See: https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#typical-status-properties
type BuildCondition struct {
	Type BuildConditionType `json:"state"`

	Status corev1.ConditionStatus `json:"status" description:"status of the condition, one of True, False, Unknown"`

	// +optional
	Reason string `json:"reason,omitempty" description:"one-word CamelCase reason for the condition's last transition"`
	// +optional
	Message string `json:"message,omitempty" description:"human-readable message indicating details about last transition"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BuildList is a list of Build resources
type BuildList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Build `json:"items"`
}

func (bs *BuildStatus) GetCondition(t BuildConditionType) *BuildCondition {
	for _, cond := range bs.Conditions {
		if cond.Type == t {
			return &cond
		}
	}
	return nil
}

func (b *BuildStatus) SetCondition(newCond *BuildCondition) {
	if newCond == nil {
		return
	}

	t := newCond.Type
	var conditions []BuildCondition
	for _, cond := range b.Conditions {
		if cond.Type != t {
			conditions = append(conditions, cond)
		}
	}
	conditions = append(conditions, *newCond)
	b.Conditions = conditions
}

func (b *BuildStatus) RemoveCondition(t BuildConditionType) {
	var conditions []BuildCondition
	for _, cond := range b.Conditions {
		if cond.Type != t {
			conditions = append(conditions, cond)
		}
	}
	b.Conditions = conditions
}

func (b *Build) GetGeneration() int64           { return b.Spec.Generation }
func (b *Build) SetGeneration(generation int64) { b.Spec.Generation = generation }
func (b *Build) GetSpecJSON() ([]byte, error)   { return json.Marshal(b.Spec) }
