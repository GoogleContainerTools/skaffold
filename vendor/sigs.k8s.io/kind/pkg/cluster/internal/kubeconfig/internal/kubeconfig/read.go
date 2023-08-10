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
	"io/ioutil"
	"os"

	yaml "gopkg.in/yaml.v3"

	"sigs.k8s.io/kind/pkg/errors"
)

// KINDFromRawKubeadm returns a kind kubeconfig derived from the raw kubeadm kubeconfig,
// the kind clusterName, and the server.
// server is ignored if unset.
func KINDFromRawKubeadm(rawKubeadmKubeConfig, clusterName, server string) (*Config, error) {
	cfg := &Config{}
	if err := yaml.Unmarshal([]byte(rawKubeadmKubeConfig), cfg); err != nil {
		return nil, err
	}

	// verify assumptions about kubeadm kubeconfigs
	if err := checkKubeadmExpectations(cfg); err != nil {
		return nil, err
	}

	// compute unique kubeconfig key for this cluster
	key := KINDClusterKey(clusterName)

	// use the unique key for all named references
	cfg.Clusters[0].Name = key
	cfg.Users[0].Name = key
	cfg.Contexts[0].Name = key
	cfg.Contexts[0].Context.User = key
	cfg.Contexts[0].Context.Cluster = key
	cfg.CurrentContext = key

	// patch server field if server was set
	if server != "" {
		cfg.Clusters[0].Cluster.Server = server
	}

	return cfg, nil
}

// read loads a KUBECONFIG file from configPath
func read(configPath string) (*Config, error) {
	// try to open, return default if no such file
	f, err := os.Open(configPath)
	if os.IsNotExist(err) {
		return &Config{}, nil
	} else if err != nil {
		return nil, errors.WithStack(err)
	}

	// otherwise read in and deserialize
	cfg := &Config{}
	rawExisting, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if err := yaml.Unmarshal(rawExisting, cfg); err != nil {
		return nil, errors.WithStack(err)
	}

	return cfg, nil
}
