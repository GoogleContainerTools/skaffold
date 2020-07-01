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

package context

import (
	"errors"
	"sync"

	"github.com/sirupsen/logrus"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// For testing
var (
	CurrentConfig = getCurrentConfig
)

var (
	lock         sync.Mutex
	initialized  bool
	rawCfg       clientcmdapi.Config
	rawCfgErr    error
	clientCfg    *restclient.Config
	clientCfgErr error
)

// ConfigureKubeConfig sets an override for the current context in the k8s config.
// When given, the firstCliValue always takes precedence over the yamlValue.
func ConfigureKubeConfig(cliKubeConfig, cliKubeContext, yamlKubeContext string) {
	lock.Lock()
	defer lock.Unlock()

	var kubeContext string
	if cliKubeContext != "" {
		kubeContext = cliKubeContext
	} else {
		kubeContext = yamlKubeContext
	}

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.ExplicitPath = cliKubeConfig
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{
		CurrentContext: kubeContext,
	})

	// Cache the RawConfig.
	rawCfg, rawCfgErr = kubeConfig.RawConfig()
	if kubeContext != "" {
		// RawConfig() does not respect the override in kubeConfig
		rawCfg.CurrentContext = kubeContext
	}

	// Cache the ClientConfig.
	clientCfg, clientCfgErr = kubeConfig.ClientConfig()
	if kubeContext == "" && cliKubeConfig == "" && clientcmd.IsEmptyConfig(clientCfgErr) {
		logrus.Debug("no kube-context set and no kubeConfig found, attempting in-cluster config")
		clientCfg, clientCfgErr = restclient.InClusterConfig()
	}

	initialized = true
}

// GetRestClientConfig returns a REST client config for API calls against the Kubernetes API.
// The cache ensures that Skaffold always works with the identical kubeconfig,
// even if it was changed on disk.
func GetRestClientConfig() (*restclient.Config, error) {
	lock.Lock()
	defer lock.Unlock()

	if !initialized {
		return nil, errors.New("cannot call GetRestClientConfig() before ConfigureKubeConfig()")
	}
	return clientCfg, clientCfgErr
}

// getCurrentConfig retrieves and caches the raw kubeConfig.
// The cache ensures that Skaffold always works with the identical kubeconfig,
// even if it was changed on disk.
func getCurrentConfig() (clientcmdapi.Config, error) {
	lock.Lock()
	defer lock.Unlock()

	if !initialized {
		return clientcmdapi.Config{}, errors.New("cannot call getCurrentConfig() before ConfigureKubeConfig()")
	}
	return rawCfg, rawCfgErr
}
