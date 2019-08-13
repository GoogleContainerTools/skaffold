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
)

// PodTemplate holds pod specific configuration
type PodTemplate struct {
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

	// SecurityContext holds pod-level security attributes and common container settings.
	// Optional: Defaults to empty.  See type description for default values of each field.
	// +optional
	SecurityContext *corev1.PodSecurityContext `json:"securityContext,omitempty"`

	// List of volumes that can be mounted by containers belonging to the pod.
	// More info: https://kubernetes.io/docs/concepts/storage/volumes
	// +optional
	Volumes []corev1.Volume `json:"volumes,omitempty" patchStrategy:"merge,retainKeys" patchMergeKey:"name" protobuf:"bytes,1,rep,name=volumes"`
}

// CombinePodTemplate takes a PodTemplate (either from TaskRun or PipelineRun) and merge it with deprecated field that were inlined.
func CombinedPodTemplate(template PodTemplate, deprecatedNodeSelector map[string]string, deprecatedTolerations []corev1.Toleration, deprecatedAffinity *corev1.Affinity) PodTemplate {
	if len(template.NodeSelector) == 0 && len(deprecatedNodeSelector) != 0 {
		template.NodeSelector = deprecatedNodeSelector
	}
	if len(template.Tolerations) == 0 && len(deprecatedTolerations) != 0 {
		template.Tolerations = deprecatedTolerations
	}
	if template.Affinity == nil && deprecatedAffinity != nil {
		template.Affinity = deprecatedAffinity
	}
	return template
}
