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
	"fmt"

	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

func GetPodDetails(pods corev1.PodInterface, podName string) error {
	pod, err := pods.Get(podName, meta_v1.GetOptions{
		IncludeUninitialized: true,
	})
	if err != nil {
		return err
	}
	for _, c := range append(pod.Status.Conditions) {
		if c.Status == v1.ConditionFalse {
			reason := c.Reason
			if reason == "" {
				reason = "could not determine."
			}
			if details := c.Message; details != "" {
				reason = fmt.Sprintf("%s. Detail: %s", reason, details)
			}
			return fmt.Errorf("pod in phase %s due to reason %s", pod.Status.Phase, reason)
		}
	}
	return nil
}
