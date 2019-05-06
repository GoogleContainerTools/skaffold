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
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/apps/v1beta1"
	"k8s.io/api/apps/v1beta2"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	v1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestGetPodTemplateSpec(t *testing.T) {
	tests := []struct {
		name        string
		apiResource interface{}
	}{
		{
			name: "v1.ReplicationController",
			apiResource: &v1.ReplicationController{
				Spec: v1.ReplicationControllerSpec{
					Template: &v1.PodTemplateSpec{},
				},
			},
		},
		// ReplicaSets
		{name: "extv1beta1.ReplicaSet", apiResource: &extv1beta1.ReplicaSet{}},
		{name: "v1beta2.ReplicaSet", apiResource: &v1beta2.ReplicaSet{}},
		{name: "appsv1.ReplicaSet", apiResource: &appsv1.ReplicaSet{}},

		// StatefulSets
		{name: "v1beta1.StatefulSet", apiResource: &v1beta1.StatefulSet{}},
		{name: "v1beta2.StatefulSet", apiResource: &v1beta2.StatefulSet{}},
		{name: "appsv1.StatefulSet", apiResource: &appsv1.StatefulSet{}},

		// Deployments
		{name: "extv1beta1.Deployment", apiResource: &extv1beta1.Deployment{}},
		{name: "v1beta1.Deployment", apiResource: &v1beta1.Deployment{}},
		{name: "v1beta2.Deployment", apiResource: &v1beta2.Deployment{}},
		{name: "appsv1.Deployment", apiResource: &appsv1.Deployment{}},

		// DaemonSets
		{name: "extv1beta1.DaemonSet", apiResource: &extv1beta1.DaemonSet{}},
		{name: "v1beta2.DaemonSet", apiResource: &v1beta2.DaemonSet{}},
		{name: "appsv1.DaemonSet", apiResource: &appsv1.DaemonSet{}},

		// Job
		{name: "batchv1.Job", apiResource: &batchv1.Job{}},

		// CronJob
		{name: "batchv1beta1.CronJob", apiResource: &batchv1beta1.CronJob{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clone := test.apiResource.(runtime.Object).DeepCopyObject()

			actualPodTpl := GetPodTemplateSpec(clone)
			testutil.CheckDeepEqual(t, false, actualPodTpl == nil)

			actualPodTpl.Name = "some-change"
			if diff := cmp.Diff(test.apiResource, clone); diff == "" {
				t.Errorf("podTemplate should alias cloned object")
			}
		})
	}
}
