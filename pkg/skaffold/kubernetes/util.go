/*
Copyright 2019 The Skaffold Authors

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

package kubernetes

import (
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/apps/v1beta1"
	"k8s.io/api/apps/v1beta2"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	v1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
)

// GetPodTemplateSpec extracts the PodTemplateSpec from public k8s api-resources.
// This list will need to be extended manually for new api-versions.
func GetPodTemplateSpec(o interface{}) (podTpl *v1.PodTemplateSpec) {
	switch o := o.(type) {
	// ReplicationControllers
	case *v1.ReplicationController:
		podTpl = o.Spec.Template

	// ReplicaSets
	case *extv1beta1.ReplicaSet:
		podTpl = &o.Spec.Template
	case *v1beta2.ReplicaSet:
		podTpl = &o.Spec.Template
	case *appsv1.ReplicaSet:
		podTpl = &o.Spec.Template

	// StatefulSets
	case *v1beta1.StatefulSet:
		podTpl = &o.Spec.Template
	case *v1beta2.StatefulSet:
		podTpl = &o.Spec.Template
	case *appsv1.StatefulSet:
		podTpl = &o.Spec.Template

	// Deployments
	case *extv1beta1.Deployment:
		podTpl = &o.Spec.Template
	case *v1beta1.Deployment:
		podTpl = &o.Spec.Template
	case *v1beta2.Deployment:
		podTpl = &o.Spec.Template
	case *appsv1.Deployment:
		podTpl = &o.Spec.Template

	// DaemonSets
	case *extv1beta1.DaemonSet:
		podTpl = &o.Spec.Template
	case *v1beta2.DaemonSet:
		podTpl = &o.Spec.Template
	case *appsv1.DaemonSet:
		podTpl = &o.Spec.Template

	// Job
	case *batchv1.Job:
		podTpl = &o.Spec.Template

	// CronJob
	case *batchv1beta1.CronJob:
		podTpl = &o.Spec.JobTemplate.Spec.Template
	}

	return
}
