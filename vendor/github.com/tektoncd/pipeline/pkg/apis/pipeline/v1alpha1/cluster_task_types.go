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
	"context"

	"github.com/knative/pkg/apis"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (t *ClusterTask) TaskSpec() TaskSpec {
	return t.Spec
}

func (t *ClusterTask) TaskMetadata() metav1.ObjectMeta {
	return t.ObjectMeta
}

func (t *ClusterTask) Copy() TaskInterface {
	return t.DeepCopy()
}

func (t *ClusterTask) SetDefaults(ctx context.Context) {
	t.Spec.SetDefaults(ctx)
}

// Check that Task may be validated and defaulted.
var _ apis.Validatable = (*ClusterTask)(nil)
var _ apis.Defaultable = (*ClusterTask)(nil)

// +genclient
// +genclient:noStatus
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterTask is a Task with a cluster scope. ClusterTasks are used to
// represent Tasks that should be publicly addressable from any namespace in the
// cluster.
type ClusterTask struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec holds the desired state of the Task from the client
	// +optional
	Spec TaskSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterTaskList contains a list of ClusterTask
type ClusterTaskList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterTask `json:"items"`
}
