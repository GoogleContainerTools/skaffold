/*
Copyright 2019 The Kubernetes Authors.

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

package kubeconfig

import (
	"sigs.k8s.io/kind/pkg/errors"
)

// KINDClusterKey identifies kind clusters in kubeconfig files
func KINDClusterKey(clusterName string) string {
	return "kind-" + clusterName
}

// checkKubeadmExpectations validates that a kubeadm created KUBECONFIG meets
// our expectations, namely on the number of entries
func checkKubeadmExpectations(cfg *Config) error {
	if len(cfg.Clusters) != 1 {
		return errors.Errorf("kubeadm KUBECONFIG should have one cluster, but read %d", len(cfg.Clusters))
	}
	if len(cfg.Users) != 1 {
		return errors.Errorf("kubeadm KUBECONFIG should have one user, but read %d", len(cfg.Users))
	}
	if len(cfg.Contexts) != 1 {
		return errors.Errorf("kubeadm KUBECONFIG should have one context, but read %d", len(cfg.Contexts))
	}
	return nil
}
