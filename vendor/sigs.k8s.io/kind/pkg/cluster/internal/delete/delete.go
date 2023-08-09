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

package delete

import (
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/log"

	"sigs.k8s.io/kind/pkg/cluster/internal/kubeconfig"
	"sigs.k8s.io/kind/pkg/cluster/internal/providers"
)

// Cluster deletes the cluster identified by ctx
// explicitKubeconfigPath is --kubeconfig, following the rules from
// https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands
func Cluster(logger log.Logger, p providers.Provider, name, explicitKubeconfigPath string) error {
	n, err := p.ListNodes(name)
	if err != nil {
		return errors.Wrap(err, "error listing nodes")
	}

	kerr := kubeconfig.Remove(name, explicitKubeconfigPath)
	if kerr != nil {
		logger.Errorf("failed to update kubeconfig: %v", kerr)
	}

	err = p.DeleteNodes(n)
	if err != nil {
		return err
	}
	if kerr != nil {
		return err
	}
	return nil
}
