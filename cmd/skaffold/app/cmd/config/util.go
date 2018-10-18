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

package config

import (
	"io/ioutil"
	"path/filepath"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

const defaultConfigDir = ".skaffold"
const defaultConfigFile = "config"

func resolveKubectlContext() {
	if kubecontext != "" {
		return
	}

	context, err := context.CurrentContext()
	if err != nil {
		logrus.Warn(errors.Wrap(err, "retrieving current kubectl context"))
	}
	if context == "" {
		logrus.Infof("no kubectl context currently set, using global values")
		global = true
	}
	kubecontext = context
}

func resolveConfigFile() error {
	if configFile == "" {
		home, err := homedir.Dir()
		if err != nil {
			return errors.Wrap(err, "retrieving home directory")
		}
		configFile = filepath.Join(home, defaultConfigDir, defaultConfigFile)
	}
	return util.VerifyOrCreateFile(configFile)
}

func ReadConfigForFile(filename string) (*Config, error) {
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.Wrap(err, "reading global config")
	}
	config := Config{}
	if err := yaml.Unmarshal(contents, &config); err != nil {
		return nil, errors.Wrap(err, "unmarshalling global skaffold config")
	}
	return &config, nil
}

func readConfig() (*Config, error) {
	if err := resolveConfigFile(); err != nil {
		return nil, errors.Wrap(err, "resolving config file location")
	}
	return ReadConfigForFile(configFile)
}

// return the specific config to be modified based on the provided kube context.
// either returns the config corresponding to the provided or current context,
// or the global config if that is specified (or if no current context is set).
func getConfigForKubectx() (*ContextConfig, error) {
	cfg, err := readConfig()
	if err != nil {
		return nil, err
	}
	if global {
		return cfg.Global, nil
	}
	for _, contextCfg := range cfg.ContextConfigs {
		if contextCfg.Kubecontext == kubecontext {
			return contextCfg, nil
		}
	}
	logrus.Infof("no config entry found for kube-context %s", kubecontext)
	return nil, nil
}

func getOrCreateConfigForKubectx() (*ContextConfig, error) {
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
