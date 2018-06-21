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
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/labels"
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

type withLabels struct {
	deploy.Deployer

	labellers []labels.Labeller
}

// WithLabels creates a deployer that sets labels on deployed resources.
func WithLabels(d deploy.Deployer, labellers ...labels.Labeller) deploy.Deployer {
	return &withLabels{
		Deployer:  d,
		labellers: labellers,
	}
}

func (w *withLabels) Deploy(ctx context.Context, out io.Writer, artifacts []build.Artifact) ([]deploy.Artifact, error) {
	dRes, err := w.Deployer.Deploy(ctx, out, artifacts)
	labelDeployResults(labels.Merge(w.labellers...), dRes)
	return dRes, err
}

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

// patcher is responsible for applying a given patch to the provided object
type patcher func(clientgo.Interface, string, string, []byte) error

// objectMeta is responsible for returning a generic runtime.Object's metadata
type objectMeta func(runtime.Object) (*metav1.ObjectMeta, bool)

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
	corev1Pod: func(r runtime.Object) (*metav1.ObjectMeta, bool) {
		obj, ok := r.(*corev1.Pod)
		if !ok {
			return nil, ok
		}
		return &obj.ObjectMeta, ok
	},
	appsv1Deployment: func(r runtime.Object) (*metav1.ObjectMeta, bool) {
		obj, ok := r.(*appsv1.Deployment)
		if !ok {
			return nil, ok
		}
		return &obj.ObjectMeta, ok
	},
	appsv1Beta1Deployment: func(r runtime.Object) (*metav1.ObjectMeta, bool) {
		obj, ok := r.(*appsv1beta1.Deployment)
		if !ok {
			return nil, ok
		}
		return &obj.ObjectMeta, ok
	},
	appsv1Beta2Deployment: func(r runtime.Object) (*metav1.ObjectMeta, bool) {
		obj, ok := r.(*appsv1beta2.Deployment)
		if !ok {
			return nil, ok
		}
		return &obj.ObjectMeta, ok
	},
	extensionsv1Beta1Deployment: func(r runtime.Object) (*metav1.ObjectMeta, bool) {
		obj, ok := r.(*extensionsv1beta1.Deployment)
		if !ok {
			return nil, ok
		}
		return &obj.ObjectMeta, ok
	},
	corev1Service: func(r runtime.Object) (*metav1.ObjectMeta, bool) {
		obj, ok := r.(*corev1.Service)
		if !ok {
			return nil, ok
		}
		return &obj.ObjectMeta, ok
	},
	appv1StatefulSet: func(r runtime.Object) (*metav1.ObjectMeta, bool) {
		obj, ok := r.(*appsv1.StatefulSet)
		if !ok {
			return nil, ok
		}
		return &obj.ObjectMeta, ok
	},
	appsv1Beta1StatefulSet: func(r runtime.Object) (*metav1.ObjectMeta, bool) {
		obj, ok := r.(*appsv1beta1.StatefulSet)
		if !ok {
			return nil, ok
		}
		return &obj.ObjectMeta, ok
	},
	appsv1Beta2StatefulSet: func(r runtime.Object) (*metav1.ObjectMeta, bool) {
		obj, ok := r.(*appsv1beta1.StatefulSet)
		if !ok {
			return nil, ok
		}
		return &obj.ObjectMeta, ok
	},
	extensionsv1Beta1DaemonSet: func(r runtime.Object) (*metav1.ObjectMeta, bool) {
		obj, ok := r.(*extensionsv1beta1.DaemonSet)
		if !ok {
			return nil, ok
		}
		return &obj.ObjectMeta, ok
	},
	appsv1ReplicaSet: func(r runtime.Object) (*metav1.ObjectMeta, bool) {
		obj, ok := r.(*appsv1.ReplicaSet)
		if !ok {
			return nil, ok
		}
		return &obj.ObjectMeta, ok
	},
	appsv1Beta2ReplicaSet: func(r runtime.Object) (*metav1.ObjectMeta, bool) {
		obj, ok := r.(*appsv1beta2.ReplicaSet)
		if !ok {
			return nil, ok
		}
		return &obj.ObjectMeta, ok
	},
}

// retry 3 times to give the object time to propagate to the API server
const tries int = 3
const sleeptime time.Duration = 300 * time.Millisecond

func labelDeployResults(labels map[string]string, results []deploy.Artifact) {
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
	var metadata *metav1.ObjectMeta
	originalJSON, _ := json.Marshal(*res.Obj)
	modifiedObj := (*res.Obj).DeepCopyObject()
	for typeStr, m := range objectMetas {
		if metadata, applied = m(modifiedObj); applied {
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
