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
	"reflect"
	"strings"

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
	return cmd
}

func setConfigValue(name string, value interface{}) error {
	configs, err := readConfig()
	if err != nil {
		return err
	}
	var cfg *ContextConfig
	for _, contextCfg := range *configs {
		if kubectx == "all" || contextCfg.Context == kubectx {
			cfg = contextCfg
			cfgValue := reflect.ValueOf(cfg.Values)
			var fieldName string
			for i := 0; i < cfgValue.NumField(); i++ {
				fieldType := reflect.TypeOf(cfg.Values).Field(i)
				for _, tag := range strings.Split(fieldType.Tag.Get("yaml"), ",") {
					if tag == name {
						fieldName = fieldType.Name
					}
				}
			}
			if fieldName == "" {
				return fmt.Errorf("%s is not a valid config field", name)
			}
			fieldValue := cfgValue.FieldByName(fieldName)

			fieldType := fieldValue.Type()
			val := reflect.ValueOf(value)

			if fieldType != val.Type() {
				return fmt.Errorf("%s is not a valid value for field %s", value, fieldName)
			}

			reflect.ValueOf(&cfg.Values).Elem().FieldByName(fieldName).Set(val)
		}
	}
	if cfg == nil {
		return fmt.Errorf("no config entry found for kubectx %s", kubectx)
	}

	return writeConfig(configs)
}

func writeConfig(cfg *Config) error {
	contents, err := yaml.Marshal(cfg)
	if err != nil {
		return errors.Wrap(err, "marshaling config")
	}
	err = ioutil.WriteFile(configFile, contents, 0644)
	if err != nil {
		return errors.Wrap(err, "writing config file")
	}
	return nil
}
