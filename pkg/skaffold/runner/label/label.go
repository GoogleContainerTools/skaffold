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

	"k8s.io/apimachinery/pkg/types"
	patch "k8s.io/apimachinery/pkg/util/strategicpatch"
)

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
	obj := *res.Obj
	originalJSON, _ := json.Marshal(obj)
	modifiedObj := obj.DeepCopyObject()
	switch obj.(type) {
	case *corev1.Pod:
		apiObject := modifiedObj.(*corev1.Pod)
		addSkaffoldLabels(labels, &apiObject.ObjectMeta)
		modifiedJSON, _ := json.Marshal(modifiedObj)
		p, _ := patch.CreateTwoWayMergePatch(originalJSON, modifiedJSON, apiObject)
		_, err = client.CoreV1().Pods(retrieveNamespace(res.Namespace, apiObject.ObjectMeta)).Patch(apiObject.Name, types.StrategicMergePatchType, p)
	case *appsv1.Deployment:
		apiObject := modifiedObj.(*appsv1.Deployment)
		addSkaffoldLabels(labels, &apiObject.ObjectMeta)
		modifiedJSON, _ := json.Marshal(modifiedObj)
		p, _ := patch.CreateTwoWayMergePatch(originalJSON, modifiedJSON, apiObject)
		_, err = client.AppsV1().Deployments(retrieveNamespace(res.Namespace, apiObject.ObjectMeta)).Patch(apiObject.Name, types.StrategicMergePatchType, p)
	case *appsv1beta1.Deployment:
		apiObject := modifiedObj.(*appsv1beta1.Deployment)
		addSkaffoldLabels(labels, &apiObject.ObjectMeta)
		modifiedJSON, _ := json.Marshal(modifiedObj)
		p, _ := patch.CreateTwoWayMergePatch(originalJSON, modifiedJSON, apiObject)
		_, err = client.AppsV1beta1().Deployments(retrieveNamespace(res.Namespace, apiObject.ObjectMeta)).Patch(apiObject.Name, types.StrategicMergePatchType, p)
	case *appsv1beta2.Deployment:
		apiObject := modifiedObj.(*appsv1beta2.Deployment)
		addSkaffoldLabels(labels, &apiObject.ObjectMeta)
		modifiedJSON, _ := json.Marshal(modifiedObj)
		p, _ := patch.CreateTwoWayMergePatch(originalJSON, modifiedJSON, apiObject)
		_, err = client.AppsV1beta2().Deployments(retrieveNamespace(res.Namespace, apiObject.ObjectMeta)).Patch(apiObject.Name, types.StrategicMergePatchType, p)
	case *extensionsv1beta1.Deployment:
		apiObject := modifiedObj.(*extensionsv1beta1.Deployment)
		addSkaffoldLabels(labels, &apiObject.ObjectMeta)
		modifiedJSON, _ := json.Marshal(modifiedObj)
		p, _ := patch.CreateTwoWayMergePatch(originalJSON, modifiedJSON, apiObject)
		_, err = client.ExtensionsV1beta1().Deployments(retrieveNamespace(res.Namespace, apiObject.ObjectMeta)).Patch(apiObject.Name, types.StrategicMergePatchType, p)
	case *corev1.Service:
		apiObject := modifiedObj.(*corev1.Service)
		addSkaffoldLabels(labels, &apiObject.ObjectMeta)
		modifiedJSON, _ := json.Marshal(modifiedObj)
		p, _ := patch.CreateTwoWayMergePatch(originalJSON, modifiedJSON, apiObject)
		_, err = client.CoreV1().Services(retrieveNamespace(res.Namespace, apiObject.ObjectMeta)).Patch(apiObject.Name, types.StrategicMergePatchType, p)
	case *appsv1.StatefulSet:
		apiObject := modifiedObj.(*appsv1.StatefulSet)
		addSkaffoldLabels(labels, &apiObject.ObjectMeta)
		modifiedJSON, _ := json.Marshal(modifiedObj)
		p, _ := patch.CreateTwoWayMergePatch(originalJSON, modifiedJSON, apiObject)
		_, err = client.AppsV1().StatefulSets(retrieveNamespace(res.Namespace, apiObject.ObjectMeta)).Patch(apiObject.Name, types.StrategicMergePatchType, p)
	case *appsv1beta1.StatefulSet:
		apiObject := modifiedObj.(*appsv1beta1.StatefulSet)
		addSkaffoldLabels(labels, &apiObject.ObjectMeta)
		modifiedJSON, _ := json.Marshal(modifiedObj)
		p, _ := patch.CreateTwoWayMergePatch(originalJSON, modifiedJSON, apiObject)
		_, err = client.AppsV1beta1().StatefulSets(retrieveNamespace(res.Namespace, apiObject.ObjectMeta)).Patch(apiObject.Name, types.StrategicMergePatchType, p)
	case *appsv1beta2.StatefulSet:
		apiObject := modifiedObj.(*appsv1beta2.StatefulSet)
		addSkaffoldLabels(labels, &apiObject.ObjectMeta)
		modifiedJSON, _ := json.Marshal(modifiedObj)
		p, _ := patch.CreateTwoWayMergePatch(originalJSON, modifiedJSON, apiObject)
		_, err = client.AppsV1beta2().StatefulSets(retrieveNamespace(res.Namespace, apiObject.ObjectMeta)).Patch(apiObject.Name, types.StrategicMergePatchType, p)
	case *extensionsv1beta1.DaemonSet:
		apiObject := modifiedObj.(*extensionsv1beta1.DaemonSet)
		addSkaffoldLabels(labels, &apiObject.ObjectMeta)
		modifiedJSON, _ := json.Marshal(modifiedObj)
		p, _ := patch.CreateTwoWayMergePatch(originalJSON, modifiedJSON, apiObject)
		_, err = client.ExtensionsV1beta1().DaemonSets(retrieveNamespace(res.Namespace, apiObject.ObjectMeta)).Patch(apiObject.Name, types.StrategicMergePatchType, p)
	case *appsv1.ReplicaSet:
		apiObject := modifiedObj.(*appsv1.ReplicaSet)
		addSkaffoldLabels(labels, &apiObject.ObjectMeta)
		modifiedJSON, _ := json.Marshal(modifiedObj)
		p, _ := patch.CreateTwoWayMergePatch(originalJSON, modifiedJSON, apiObject)
		_, err = client.AppsV1().ReplicaSets(retrieveNamespace(res.Namespace, apiObject.ObjectMeta)).Patch(apiObject.Name, types.StrategicMergePatchType, p)
	case *appsv1beta2.ReplicaSet:
		apiObject := modifiedObj.(*appsv1beta2.ReplicaSet)
		addSkaffoldLabels(labels, &apiObject.ObjectMeta)
		modifiedJSON, _ := json.Marshal(modifiedObj)
		p, _ := patch.CreateTwoWayMergePatch(originalJSON, modifiedJSON, apiObject)
		_, err = client.AppsV1beta2().ReplicaSets(retrieveNamespace(res.Namespace, apiObject.ObjectMeta)).Patch(apiObject.Name, types.StrategicMergePatchType, p)
	default:
		logrus.Infof("unknown runtime.Object, skipping label")
		return nil
	}
	return err
}
