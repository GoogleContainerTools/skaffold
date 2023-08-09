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

package common

import (
	"os"
	"strings"

	"sigs.k8s.io/kind/pkg/internal/apis/config"
)

const (
	// HTTPProxy is the HTTP_PROXY environment variable key
	HTTPProxy = "HTTP_PROXY"
	// HTTPSProxy is the HTTPS_PROXY environment variable key
	HTTPSProxy = "HTTPS_PROXY"
	// NOProxy is the NO_PROXY environment variable key
	NOProxy = "NO_PROXY"
)

// GetProxyEnvs returns a map of proxy environment variables to their values
// If proxy settings are set, NO_PROXY is modified to include the cluster subnets
func GetProxyEnvs(cfg *config.Cluster) map[string]string {
	return getProxyEnvs(cfg, os.Getenv)
}

func getProxyEnvs(cfg *config.Cluster, getEnv func(string) string) map[string]string {
	envs := make(map[string]string)
	for _, name := range []string{HTTPProxy, HTTPSProxy, NOProxy} {
		val := getEnv(name)
		if val == "" {
			val = getEnv(strings.ToLower(name))
		}
		if val != "" {
			envs[name] = val
			envs[strings.ToLower(name)] = val
		}
	}
	// Specifically add the cluster subnets to NO_PROXY if we are using a proxy
	if len(envs) > 0 {
		noProxy := envs[NOProxy]
		if noProxy != "" {
			noProxy += ","
		}
		noProxy += cfg.Networking.ServiceSubnet + "," + cfg.Networking.PodSubnet
		envs[NOProxy] = noProxy
		envs[strings.ToLower(NOProxy)] = noProxy
	}
	return envs
}
