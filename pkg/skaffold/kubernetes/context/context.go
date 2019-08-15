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
	"sync"

	"github.com/pkg/errors"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// For testing
var (
	CurrentConfig = getCurrentConfig
)

var (
	kubeConfigOnce sync.Once
	kubeConfig     clientcmd.ClientConfig
	kubeContext    string
)

// resetConfig is used by tests
func resetConfig() {
	kubeConfigOnce = sync.Once{}
}

// UseKubeContext sets an override for the current context in the k8s config.
func UseKubeContext(overrideKubeContext string) {
	kubeContext = overrideKubeContext
}

// GetRestClientConfig returns a REST client config for API calls against the Kubernetes API.
// If UseKubeContext was called before, the CurrentContext will be overridden.
// The result will be cached after the first call.
func GetRestClientConfig() (*restclient.Config, error) {
	rawConfig, err := getRawKubeConfig()
	if err != nil {
		return nil, err
	}
	clientConfig := clientcmd.NewNonInteractiveClientConfig(rawConfig, kubeContext, &clientcmd.ConfigOverrides{CurrentContext: kubeContext}, nil)
	restConfig, err := clientConfig.ClientConfig()
	return restConfig, errors.Wrap(err, "error creating REST client config")
}

// getCurrentConfig retrieves the kubeconfig file. If UseKubeContext was called before, the CurrentContext will be overridden.
// The result will be cached after the first call.
func getCurrentConfig() (clientcmdapi.Config, error) {
	cfg, err := getRawKubeConfig()
	if kubeContext != "" {
		// RawConfig does not respect the override in kubeConfig
		cfg.CurrentContext = kubeContext
	}
	return cfg, err
}

// getRawKubeConfig retrieves and caches the raw kubeConfig. The cache ensures that Skaffold always works with the identical kubeconfig,
// even if it was changed on disk.
func getRawKubeConfig() (clientcmdapi.Config, error) {
	kubeConfigOnce.Do(func() {
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		kubeConfig = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{
			CurrentContext: kubeContext,
		})
	})
	rawConfig, err := kubeConfig.RawConfig()
	return rawConfig, errors.Wrap(err, "loading kubeconfig")
}
