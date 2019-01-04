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

package kubernetes

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/kubectl/cmd"
	"k8s.io/kubernetes/pkg/kubectl/scheme"
)

func Exec(opts cmd.StreamOptions, command []string) error {
	config, err := getClientConfig()
	if err != nil {
		return errors.Wrap(err, "getting rest config")
	}
	setConfigDefaults(config)

	client, err := internalclientset.NewForConfig(config)
	execOpts := cmd.ExecOptions{
		Config:        config,
		PodClient:     client.Core(),
		Executor:      &cmd.DefaultRemoteExecutor{},
		Command:       command,
		StreamOptions: opts,
	}

	return execOpts.Run()
}

//see https://github.com/kubernetes/client-go/blob/master/kubernetes/typed/core/v1/core_client.go#L145
func setConfigDefaults(config *rest.Config) *rest.Config {
	config.APIPath = "/api"
	config.GroupVersion = &corev1.SchemeGroupVersion
	config.NegotiatedSerializer = serializer.NewCodecFactory(scheme.Scheme)
	return config
}
