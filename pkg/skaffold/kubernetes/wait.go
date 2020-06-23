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
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// WatchUntil reads items from the watch until the provided condition succeeds or the context is cancelled.
func watchUntilTimeout(ctx context.Context, timeout time.Duration, w watch.Interface, condition func(event *watch.Event) (bool, error)) error {
	ctx, cancelTimeout := context.WithTimeout(ctx, timeout)
	defer cancelTimeout()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event := <-w.ResultChan():
			done, err := condition(&event)
			if err != nil {
				return err
			}
			if done {
				return nil
			}
		}
	}
}

// WaitForPodSucceeded waits until the Pod status is Succeeded.
func WaitForPodSucceeded(ctx context.Context, pods corev1.PodInterface, podName string, timeout time.Duration) error {
	logrus.Infof("Waiting for %s to be complete", podName)

	w, err := pods.Watch(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("initializing pod watcher: %s", err)
	}
	defer w.Stop()

	return watchUntilTimeout(ctx, timeout, w, isPodSucceeded(podName))
}

func isPodSucceeded(podName string) func(event *watch.Event) (bool, error) {
	return func(event *watch.Event) (bool, error) {
		if event.Object == nil {
			return false, nil
		}
		pod := event.Object.(*v1.Pod)
		if pod.Name != podName {
			return false, nil
		}

		switch pod.Status.Phase {
		case v1.PodSucceeded:
			return true, nil
		case v1.PodRunning:
			return false, nil
		case v1.PodFailed:
			return false, errors.New("pod has failed")
		case v1.PodUnknown, v1.PodPending:
			return false, nil
		}
		return false, fmt.Errorf("unknown phase: %s", pod.Status.Phase)
	}
}

// WaitForPodInitialized waits until init containers have started running
func WaitForPodInitialized(ctx context.Context, pods corev1.PodInterface, podName string) error {
	logrus.Infof("Waiting for %s to be initialized", podName)

	w, err := pods.Watch(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("initializing pod watcher: %s", err)
	}
	defer w.Stop()

	return watchUntilTimeout(ctx, 10*time.Minute, w, func(event *watch.Event) (bool, error) {
		pod := event.Object.(*v1.Pod)
		if pod.Name != podName {
			return false, nil
		}

		for _, ic := range pod.Status.InitContainerStatuses {
			if ic.State.Running != nil {
				return true, nil
			}
		}
		return false, nil
	})
}

// WaitForDeploymentToStabilize waits until the Deployment has a matching generation/replica count between spec and status.
func WaitForDeploymentToStabilize(ctx context.Context, c kubernetes.Interface, ns, name string, timeout time.Duration) error {
	logrus.Infof("Waiting for %s to stabilize", name)

	fields := fields.Set{
		"metadata.name":      name,
		"metadata.namespace": ns,
	}
	w, err := c.AppsV1().Deployments(ns).Watch(metav1.ListOptions{
		FieldSelector: fields.AsSelector().String(),
	})
	if err != nil {
		return fmt.Errorf("initializing deployment watcher: %s", err)
	}

	return watchUntilTimeout(ctx, timeout, w, func(event *watch.Event) (bool, error) {
		if event.Type == watch.Deleted {
			return false, apierrs.NewNotFound(schema.GroupResource{Resource: "deployments"}, "")
		}

		if dp, ok := event.Object.(*appsv1.Deployment); ok {
			if dp.Name == name && dp.Namespace == ns &&
				dp.Generation <= dp.Status.ObservedGeneration &&
				*(dp.Spec.Replicas) == dp.Status.Replicas {
				return true, nil
			}
			glog.Infof("Waiting for deployment %s to stabilize, generation %v observed generation %v spec.replicas %d status.replicas %d",
				name, dp.Generation, dp.Status.ObservedGeneration, *(dp.Spec.Replicas), dp.Status.Replicas)
		}
		return false, nil
	})
}
