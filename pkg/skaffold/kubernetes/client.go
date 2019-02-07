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

package kubernetes

import (
	"fmt"

	"github.com/pkg/errors"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	// Initialize all known client auth plugins
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func GetClientset() (kubernetes.Interface, error) {
	config, err := getClientConfig()
	if err != nil {
		return nil, errors.Wrap(err, "getting client config for kubernetes client")
	}
	return kubernetes.NewForConfig(config)
}

func getClientConfig() (*restclient.Config, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
	clientConfig, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("error creating kubeConfig: %s", err)
	}
	return clientConfig, nil
}

func GetDynamicClient() (dynamic.Interface, error) {
	config, err := getClientConfig()
	if err != nil {
		return nil, errors.Wrap(err, "getting client config for dynamic client")
	}
	return dynamic.NewForConfig(config)
}
