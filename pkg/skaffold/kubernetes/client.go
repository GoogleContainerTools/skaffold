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
	"github.com/pkg/errors"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"

	// Initialize all known client auth plugins
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

// for tests
var (
	Client        = getClientset
	DynamicClient = getDynamicClient
)

func getClientset() (kubernetes.Interface, error) {
	config, err := context.GetRestClientConfig()
	if err != nil {
		return nil, errors.Wrap(err, "getting client config for kubernetes client")
	}
	return kubernetes.NewForConfig(config)
}

func getDynamicClient() (dynamic.Interface, error) {
	config, err := context.GetRestClientConfig()
	if err != nil {
		return nil, errors.Wrap(err, "getting client config for dynamic client")
	}
	return dynamic.NewForConfig(config)
}
