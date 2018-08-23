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
	"fmt"
	"io"
	"io/ioutil"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
)

func NewCmdSet(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set a value in the global skaffold config",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			resolveKubectlContext()
			return setConfigValue(args[0], args[1])
		},
	}
	AddConfigFlags(cmd)
	AddSetFlags(cmd)
	return cmd
}

func setConfigValue(name string, value interface{}) error {
	cfg, err := getConfigForKubectx()
	if err != nil {
		return err
	}

	switch name {
	case "default-repo":
		repoValue, ok := value.(string)
		if !ok {
			return fmt.Errorf("provided value %v not a valid value for field default-repo", value)
		}
		cfg.DefaultRepo = repoValue
	default:
		return fmt.Errorf("%s is not a valid config field", name)
	}

	return writeConfig(cfg)
}

func writeConfig(cfg *ContextConfig) error {
	configs, err := readConfig()
	if err != nil {
		return err
	}
	if global {
		configs.Global = cfg
	} else {
		for i, contextCfg := range configs.ContextConfigs {
			if contextCfg.Kubectx == kubectx {
				configs.ContextConfigs[i] = cfg
			}
		}
	}
	contents, err := yaml.Marshal(configs)
	if err != nil {
		return errors.Wrap(err, "marshaling config")
	}
	err = ioutil.WriteFile(configFile, contents, 0644)
	if err != nil {
		return errors.Wrap(err, "writing config file")
	}
	return nil
}
