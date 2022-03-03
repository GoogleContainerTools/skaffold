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
	"context"
	"fmt"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	kctx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

func resolveKubectlContext() {
	if kubecontext != "" {
		return
	}

	config, err := kctx.CurrentConfig()
	switch {
	case err != nil:
		log.Entry(context.TODO()).Warn("unable to retrieve current kubectl context, using global values")
		global = true
	case config.CurrentContext == "":
		log.Entry(context.TODO()).Info("no kubectl context currently set, using global values")
		global = true
	default:
		kubecontext = config.CurrentContext
	}
}

func getConfigForKubectxOrDefault() (*config.ContextConfig, error) {
	cfg, err := getConfigForKubectx()
	if err != nil {
		return nil, err
	}

	if cfg == nil {
		cfg = &config.ContextConfig{}
		if !global {
			cfg.Kubecontext = kubecontext
		}
	}

	return cfg, nil
}

func getConfigForKubectx() (*config.ContextConfig, error) {
	resolveKubectlContext()

	if kubecontext == "" && !global {
		return nil, fmt.Errorf("missing `--kube-context` or `--global`")
	}

	cfg, err := config.ReadConfigFile(configFile)
	if err != nil {
		return nil, err
	}
	if global {
		return cfg.Global, nil
	}
	for _, contextCfg := range cfg.ContextConfigs {
		if util.RegexEqual(contextCfg.Kubecontext, kubecontext) {
			return contextCfg, nil
		}
	}
	return nil, nil
}
