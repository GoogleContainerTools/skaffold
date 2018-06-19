/*
Copyright 2018 The Skaffold Authors

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
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

var (
	currentConfigOnce sync.Once
	currentConfig     clientcmdapi.Config
	currentConfigErr  error
)

func CurrentConfig() (clientcmdapi.Config, error) {
	currentConfigOnce.Do(func() {
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
		cfg, err := kubeConfig.RawConfig()
		if err != nil {
			currentConfigErr = errors.Wrap(err, "loading kubeconfig")
			return
		}
		currentConfig = cfg
	})
	return currentConfig, currentConfigErr
}

func CurrentContext() (string, error) {
	cfg, err := CurrentConfig()
	if err != nil {
		return "", err
	}
	return cfg.CurrentContext, nil
}
