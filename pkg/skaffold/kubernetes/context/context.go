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
	"fmt"
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
	kubeConfigOnce sync.Once
	kubeConfig     clientcmd.ClientConfig

	configureOnce  sync.Once
	kubeContext    string
	kubeConfigFile string
)

// ConfigureKubeConfig sets an override for the current context in the k8s config.
// When given, the firstCliValue always takes precedence over the yamlValue.
// Changing the kube-context of a running Skaffold process is not supported, so
// after the first call, the kube-context will be locked.
func ConfigureKubeConfig(cliKubeConfig, cliKubeContext, yamlKubeContext string) {
	newKubeContext := yamlKubeContext
	if cliKubeContext != "" {
		newKubeContext = cliKubeContext
	}
	configureOnce.Do(func() {
		kubeContext = newKubeContext
		kubeConfigFile = cliKubeConfig
		if kubeContext != "" {
			logrus.Infof("Activated kube-context %q", kubeContext)
		}
	})
	if kubeContext != newKubeContext {
		logrus.Warn("Changing the kube-context is not supported after startup. Please restart Skaffold to take effect.")
	}
}

// GetRestClientConfig returns a REST client config for API calls against the Kubernetes API.
// If ConfigureKubeConfig was called before, the CurrentContext will be overridden.
// The kubeconfig used will be cached for the life of the skaffold process after the first call.
// If the CurrentContext is empty and the resulting config is empty, this method attempts to
// create a RESTClient with an in-cluster config.
func GetRestClientConfig() (*restclient.Config, error) {
	return getRestClientConfig(kubeContext, kubeConfigFile)
}

// GetClusterInfo returns the Cluster information for the given kubeContext
func GetClusterInfo(kctx string) (*clientcmdapi.Cluster, error) {
	rawConfig, err := getCurrentConfig()
	if err != nil {
		return nil, err
	}
	c, found := rawConfig.Clusters[kctx]
	if !found {
		return nil, fmt.Errorf("failed to get cluster info for kubeContext: `%s`", kctx)
	}
	return c, nil
}

func getRestClientConfig(kctx string, kcfg string) (*restclient.Config, error) {
	logrus.Debugf("getting client config for kubeContext: `%s`", kctx)

	rawConfig, err := getCurrentConfig()
	if err != nil {
		return nil, err
	}

	clientConfig := clientcmd.NewNonInteractiveClientConfig(rawConfig, kctx, &clientcmd.ConfigOverrides{CurrentContext: kctx}, clientcmd.NewDefaultClientConfigLoadingRules())
	restConfig, err := clientConfig.ClientConfig()
	if kctx == "" && kcfg == "" && clientcmd.IsEmptyConfig(err) {
		logrus.Debug("no kube-context set and no kubeConfig found, attempting in-cluster config")
		restConfig, err := restclient.InClusterConfig()
		if err != nil {
			return restConfig, fmt.Errorf("error creating REST client config in-cluster: %w", err)
		}

		return restConfig, nil
	}
	if err != nil {
		return restConfig, fmt.Errorf("error creating REST client config for kubeContext %q: %w", kctx, err)
	}

	return restConfig, nil
}

// getCurrentConfig retrieves and caches the raw kubeConfig. The cache ensures that Skaffold always works with the identical kubeconfig,
// even if it was changed on disk.
func getCurrentConfig() (clientcmdapi.Config, error) {
	kubeConfigOnce.Do(func() {
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		loadingRules.ExplicitPath = kubeConfigFile
		kubeConfig = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{
			CurrentContext: kubeContext,
		})
	})

	cfg, err := kubeConfig.RawConfig()
	if kubeContext != "" {
		// RawConfig does not respect the override in kubeConfig
		cfg.CurrentContext = kubeContext
	}
	return cfg, err
}
