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

// RemoveKIND removes the kind cluster kindClusterName from the KUBECONFIG
// files at configPaths
func RemoveKIND(kindClusterName string, explicitPath string) error {
	// remove kind from each if present
	for _, configPath := range paths(explicitPath, os.Getenv) {
		if err := func(configPath string) error {
			// lock before modifying
			if err := lockFile(configPath); err != nil {
				return errors.Wrap(err, "failed to lock config file")
			}
			defer func(configPath string) {
				_ = unlockFile(configPath)
			}(configPath)

			// read in existing
			existing, err := read(configPath)
			if err != nil {
				return errors.Wrap(err, "failed to read kubeconfig to remove KIND entry")
			}

			// remove the kind cluster from the config
			if remove(existing, kindClusterName) {
				// write out the updated config if we modified anything
				if err := write(existing, configPath); err != nil {
					return err
				}
			}

			return nil
		}(configPath); err != nil {
			return err
		}
	}
	return nil
}

// remove drops kindClusterName entries from the cfg
func remove(cfg *Config, kindClusterName string) bool {
	mutated := false

	// get kind cluster identifier
	key := KINDClusterKey(kindClusterName)

	// filter out kind cluster from clusters
	kept := 0
	for _, c := range cfg.Clusters {
		if c.Name != key {
			cfg.Clusters[kept] = c
			kept++
		} else {
			mutated = true
		}
	}
	cfg.Clusters = cfg.Clusters[:kept]

	// filter out kind cluster from users
	kept = 0
	for _, u := range cfg.Users {
		if u.Name != key {
			cfg.Users[kept] = u
			kept++
		} else {
			mutated = true
		}
	}
	cfg.Users = cfg.Users[:kept]

	// filter out kind cluster from contexts
	kept = 0
	for _, c := range cfg.Contexts {
		if c.Name != key {
			cfg.Contexts[kept] = c
			kept++
		} else {
			mutated = true
		}
	}
	cfg.Contexts = cfg.Contexts[:kept]

	// unset current context if it points to this cluster
	if cfg.CurrentContext == key {
		cfg.CurrentContext = ""
		mutated = true
	}

	return mutated
}
