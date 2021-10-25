/*
Copyright 2021 The Skaffold Authors

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

package validator

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
)

type deploymentPodsSelector struct {
	k      kubernetes.Interface
	depObj appsv1.Deployment
}

func NewDeploymentPodsSelector(k kubernetes.Interface, d appsv1.Deployment) PodSelector {
	return &deploymentPodsSelector{k, d}
}

func (s *deploymentPodsSelector) Select(ctx context.Context, ns string, opts metav1.ListOptions) ([]v1.Pod, error) {
	_, _, controller, err := getReplicaSet(&s.depObj, s.k.AppsV1())
	if err != nil {
		log.Entry(ctx).Debugf("could not fetch deployment replica set %s", err)
		return nil, err
	} else if controller == nil {
		log.Entry(ctx).Debugf("deployment replica set not created yet.")
		return nil, nil
	}

	pods, err := s.k.CoreV1().Pods(ns).List(ctx, opts)
	if err != nil {
		return nil, err
	}
	var filtered []v1.Pod
	for _, po := range pods.Items {
		if isPodOwnedBy(po, controller) {
			filtered = append(filtered, po)
		}
	}
	return filtered, nil
}
