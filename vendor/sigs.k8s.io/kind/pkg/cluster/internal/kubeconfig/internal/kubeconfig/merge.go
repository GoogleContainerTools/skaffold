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
	"os"

	"sigs.k8s.io/kind/pkg/errors"
)

// WriteMerged writes a kind kubeconfig (see KINDFromRawKubeadm) into configPath
// merging with the existing contents if any and setting the current context to
// the kind config's current context.
func WriteMerged(kindConfig *Config, explicitConfigPath string) error {
	// figure out what filepath we should use
	configPath := pathForMerge(explicitConfigPath, os.Getenv)

	// lock config file the same as client-go
	if err := lockFile(configPath); err != nil {
		return errors.Wrap(err, "failed to lock config file")
	}
	defer func() {
		_ = unlockFile(configPath)
	}()

	// read in existing
	existing, err := read(configPath)
	if err != nil {
		return errors.Wrap(err, "failed to get kubeconfig to merge")
	}

	// merge with kind kubeconfig
	if err := merge(existing, kindConfig); err != nil {
		return err
	}

	// write back out
	return write(existing, configPath)
}

// merge kind config into an existing config
func merge(existing, kind *Config) error {
	// verify assumptions about kubeadm / kind kubeconfigs
	if err := checkKubeadmExpectations(kind); err != nil {
		return err
	}

	// insert or append cluster entry
	shouldAppend := true
	for i := range existing.Clusters {
		if existing.Clusters[i].Name == kind.Clusters[0].Name {
			existing.Clusters[i] = kind.Clusters[0]
			shouldAppend = false
		}
	}
	if shouldAppend {
		existing.Clusters = append(existing.Clusters, kind.Clusters[0])
	}

	// insert or append user entry
	shouldAppend = true
	for i := range existing.Users {
		if existing.Users[i].Name == kind.Users[0].Name {
			existing.Users[i] = kind.Users[0]
			shouldAppend = false
		}
	}
	if shouldAppend {
		existing.Users = append(existing.Users, kind.Users[0])
	}

	// insert or append context entry
	shouldAppend = true
	for i := range existing.Contexts {
		if existing.Contexts[i].Name == kind.Contexts[0].Name {
			existing.Contexts[i] = kind.Contexts[0]
			shouldAppend = false
		}
	}
	if shouldAppend {
		existing.Contexts = append(existing.Contexts, kind.Contexts[0])
	}

	// set the current context
	existing.CurrentContext = kind.CurrentContext

	// TODO: We should not need this, but it allows broken clients that depend
	// on apiVersion and kind to work. Notably the upstream javascript client.
	// See: https://github.com/kubernetes-sigs/kind/issues/1242
	if len(existing.OtherFields) == 0 {
		// TODO: Should we be deep-copying? for now we don't need to
		// and doing so would be a pain (re and de-serialize maybe?) :shrug:
		existing.OtherFields = kind.OtherFields
	}

	return nil
}
