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

package config

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"

	"github.com/sirupsen/logrus"
)

func resolveKubectlContext() {
	if kubecontext != "" {
		return
	}

	config, err := context.CurrentConfig()
	switch {
	case err != nil:
		logrus.Warn("unable to retrieve current kubectl context, using global values")
		global = true
	case config.CurrentContext == "":
		logrus.Infof("no kubectl context currently set, using global values")
		global = true
	default:
		kubecontext = config.CurrentContext
	}
}

func getOrCreateConfigForKubectx() (*ContextConfig, error) {
	resolveKubectlContext()
	cfg, err := readConfig()
	if err != nil {
		return nil, err
	}
	if global {
		if cfg.Global == nil {
			newCfg := &ContextConfig{}
			cfg.Global = newCfg
			if err := writeFullConfig(cfg); err != nil {
				return nil, err
			}
		}
		return cfg.Global, nil
	}
	for _, contextCfg := range cfg.ContextConfigs {
		if contextCfg.Kubecontext == kubecontext {
			return contextCfg, nil
		}
	}
	newCfg := &ContextConfig{
		Kubecontext: kubecontext,
	}
	cfg.ContextConfigs = append(cfg.ContextConfigs, newCfg)

	if err := writeFullConfig(cfg); err != nil {
		return nil, err
	}

	return newCfg, nil
}
