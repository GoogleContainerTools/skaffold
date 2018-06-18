/*
Copyright 2018 The Skaffold Authors

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

package label

import (
	"encoding/json"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/sirupsen/logrus"

	clientgo "k8s.io/client-go/kubernetes"

	appsv1 "k8s.io/api/apps/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	patch "k8s.io/apimachinery/pkg/util/strategicpatch"
)

type objectType int

// List of API Objects supported by the Skaffold Labeler
const (
	_ = iota
	corev1Pod
	appsv1Deployment
	appsv1Beta1Deployment
	appsv1Beta2Deployment
	extensionsv1Beta1Deployment
	corev1Service
	appv1StatefulSet
	appsv1Beta1StatefulSet
	appsv1Beta2StatefulSet
	extensionsv1Beta1DaemonSet
	appsv1ReplicaSet
	appsv1Beta2ReplicaSet
)

// converter is responsible for determining whether an object can be converted to a given type
type converter func(runtime.Object) bool

// patcher is responsible for applying a given patch to the provided object
type patcher func(clientgo.Interface, string, string, []byte) error

// objectMeta is responsible for returning a generic runtime.Object's metadata
type objectMeta func(runtime.Object) *metav1.ObjectMeta

var converters = map[objectType]converter{
	corev1Pod: func(r runtime.Object) bool {
		_, ok := r.(*corev1.Pod)
		return ok
	},
	appsv1Deployment: func(r runtime.Object) bool {
		_, ok := r.(*appsv1.Deployment)
		return ok
	},
	appsv1Beta1Deployment: func(r runtime.Object) bool {
		_, ok := r.(*appsv1beta1.Deployment)
		return ok
	},
	appsv1Beta2Deployment: func(r runtime.Object) bool {
		_, ok := r.(*appsv1beta2.Deployment)
		return ok
	},
	extensionsv1Beta1Deployment: func(r runtime.Object) bool {
		_, ok := r.(*extensionsv1beta1.Deployment)
		return ok
	},
	corev1Service: func(r runtime.Object) bool {
		_, ok := r.(*corev1.Service)
		return ok
	},
	appv1StatefulSet: func(r runtime.Object) bool {
		_, ok := r.(*appsv1.StatefulSet)
		return ok
	},
	appsv1Beta1StatefulSet: func(r runtime.Object) bool {
		_, ok := r.(*appsv1beta1.StatefulSet)
		return ok
	},
	appsv1Beta2StatefulSet: func(r runtime.Object) bool {
		_, ok := r.(*appsv1beta2.StatefulSet)
		return ok
	},
	extensionsv1Beta1DaemonSet: func(r runtime.Object) bool {
		_, ok := r.(*extensionsv1beta1.DaemonSet)
		return ok
	},
	appsv1ReplicaSet: func(r runtime.Object) bool {
		_, ok := r.(*appsv1.ReplicaSet)
		return ok
	},
	appsv1Beta2ReplicaSet: func(r runtime.Object) bool {
		_, ok := r.(*appsv1beta2.ReplicaSet)
		return ok
	},
}

var patchers = map[objectType]patcher{
	corev1Pod: func(client clientgo.Interface, ns string, name string, p []byte) error {
		_, err := client.CoreV1().Pods(ns).Patch(name, types.StrategicMergePatchType, p)
		return err
	},
	appsv1Deployment: func(client clientgo.Interface, ns string, name string, p []byte) error {
		_, err := client.AppsV1().Deployments(ns).Patch(name, types.StrategicMergePatchType, p)
		return err
	},
	appsv1Beta1Deployment: func(client clientgo.Interface, ns string, name string, p []byte) error {
		_, err := client.AppsV1beta1().Deployments(ns).Patch(name, types.StrategicMergePatchType, p)
		return err
	},
	appsv1Beta2Deployment: func(client clientgo.Interface, ns string, name string, p []byte) error {
		_, err := client.AppsV1beta2().Deployments(ns).Patch(name, types.StrategicMergePatchType, p)
		return err
	},
	extensionsv1Beta1Deployment: func(client clientgo.Interface, ns string, name string, p []byte) error {
		_, err := client.ExtensionsV1beta1().Deployments(ns).Patch(name, types.StrategicMergePatchType, p)
		return err
	},
	corev1Service: func(client clientgo.Interface, ns string, name string, p []byte) error {
		_, err := client.CoreV1().Services(ns).Patch(name, types.StrategicMergePatchType, p)
		return err
	},
	appv1StatefulSet: func(client clientgo.Interface, ns string, name string, p []byte) error {
		_, err := client.AppsV1().StatefulSets(ns).Patch(name, types.StrategicMergePatchType, p)
		return err
	},
	appsv1Beta1StatefulSet: func(client clientgo.Interface, ns string, name string, p []byte) error {
		_, err := client.AppsV1beta1().StatefulSets(ns).Patch(name, types.StrategicMergePatchType, p)
		return err
	},
	appsv1Beta2StatefulSet: func(client clientgo.Interface, ns string, name string, p []byte) error {
		_, err := client.AppsV1beta2().StatefulSets(ns).Patch(name, types.StrategicMergePatchType, p)
		return err
	},
	extensionsv1Beta1DaemonSet: func(client clientgo.Interface, ns string, name string, p []byte) error {
		_, err := client.ExtensionsV1beta1().DaemonSets(ns).Patch(name, types.StrategicMergePatchType, p)
		return err
	},
	appsv1ReplicaSet: func(client clientgo.Interface, ns string, name string, p []byte) error {
		_, err := client.AppsV1().ReplicaSets(ns).Patch(name, types.StrategicMergePatchType, p)
		return err
	},
	appsv1Beta2ReplicaSet: func(client clientgo.Interface, ns string, name string, p []byte) error {
		_, err := client.AppsV1beta2().ReplicaSets(ns).Patch(name, types.StrategicMergePatchType, p)
		return err
	},
}

var objectMetas = map[objectType]objectMeta{
	corev1Pod: func(r runtime.Object) *metav1.ObjectMeta {
		return &(r.(*corev1.Pod).ObjectMeta)
	},
	appsv1Deployment: func(r runtime.Object) *metav1.ObjectMeta {
		return &(r.(*appsv1.Deployment).ObjectMeta)
	},
	appsv1Beta1Deployment: func(r runtime.Object) *metav1.ObjectMeta {
		return &(r.(*appsv1beta1.Deployment).ObjectMeta)
	},
	appsv1Beta2Deployment: func(r runtime.Object) *metav1.ObjectMeta {
		return &(r.(*appsv1beta2.Deployment).ObjectMeta)
	},
	extensionsv1Beta1Deployment: func(r runtime.Object) *metav1.ObjectMeta {
		return &(r.(*extensionsv1beta1.Deployment).ObjectMeta)
	},
	corev1Service: func(r runtime.Object) *metav1.ObjectMeta {
		return &(r.(*corev1.Service).ObjectMeta)
	},
	appv1StatefulSet: func(r runtime.Object) *metav1.ObjectMeta {
		return &(r.(*appsv1.StatefulSet).ObjectMeta)
	},
	appsv1Beta1StatefulSet: func(r runtime.Object) *metav1.ObjectMeta {
		return &(r.(*appsv1beta1.StatefulSet).ObjectMeta)
	},
	appsv1Beta2StatefulSet: func(r runtime.Object) *metav1.ObjectMeta {
		return &(r.(*appsv1beta1.StatefulSet).ObjectMeta)
	},
	extensionsv1Beta1DaemonSet: func(r runtime.Object) *metav1.ObjectMeta {
		return &(r.(*extensionsv1beta1.DaemonSet).ObjectMeta)
	},
	appsv1ReplicaSet: func(r runtime.Object) *metav1.ObjectMeta {
		return &(r.(*appsv1.ReplicaSet).ObjectMeta)
	},
	appsv1Beta2ReplicaSet: func(r runtime.Object) *metav1.ObjectMeta {
		return &(r.(*appsv1beta2.ReplicaSet).ObjectMeta)
	},
}

// retry 3 times to give the object time to propagate to the API server
const tries int = 3
const sleeptime time.Duration = 300 * time.Millisecond

//nolint
func LabelDeployResults(labels map[string]string, results []deploy.Artifact) {
	// use the kubectl client to update all k8s objects with a skaffold watermark
	client, err := kubernetes.Client()
	if err != nil {
		logrus.Warnf("error retrieving kubernetes client: %s", err.Error())
		return
	}
	for _, res := range results {
		err = nil
		for i := 0; i < tries; i++ {
			if err = updateRuntimeObject(client, labels, res); err == nil {
				break
			}
			time.Sleep(sleeptime)
		}
		if err != nil {
			logrus.Warnf("error adding label to runtime object: %s", err.Error())
		}
	}
}

func addSkaffoldLabels(labels map[string]string, m *metav1.ObjectMeta) {
	if m.Labels == nil {
		m.Labels = map[string]string{}
	}
	for k, v := range labels {
		m.Labels[k] = v
	}
}

func retrieveNamespace(ns string, m metav1.ObjectMeta) string {
	if ns != "" {
		return ns
	}
	if m.Namespace != "" {
		return m.Namespace
	}
	return "default"
}

// TODO(nkubala): change this to use the client-go dynamic client or something equally clean
func updateRuntimeObject(client clientgo.Interface, labels map[string]string, res deploy.Artifact) error {
	for k, v := range constants.Labels.DefaultLabels {
		labels[k] = v
	}
	var err error
	applied := false
	obj := *res.Obj
	originalJSON, _ := json.Marshal(obj)
	modifiedObj := obj.DeepCopyObject()
	for typeStr, c := range converters {
		if applied = c(modifiedObj); applied {
			metadata := objectMetas[typeStr](modifiedObj)
			addSkaffoldLabels(labels, metadata)
			modifiedJSON, _ := json.Marshal(modifiedObj)
			p, _ := patch.CreateTwoWayMergePatch(originalJSON, modifiedJSON, modifiedObj)
			err = patchers[typeStr](client, retrieveNamespace(res.Namespace, *metadata), metadata.GetName(), p)
			break // we should only ever apply one patch, so stop here
		}
	}
	if !applied {
		logrus.Infof("unknown runtime.Object, skipping label")
	}
	return err
}
