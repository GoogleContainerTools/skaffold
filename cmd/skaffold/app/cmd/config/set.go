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
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

func Set(out io.Writer, args []string) error {
	if err := setConfigValue(args[0], args[1]); err != nil {
		return err
	}
	logSetConfigForUser(out, args[0], args[1])
	return nil
}

func setConfigValue(name string, value string) error {
	cfg, err := getOrCreateConfigForKubectx()
	if err != nil {
		return err
	}

	fieldName := getFieldName(cfg, name)
	if fieldName == "" {
		return fmt.Errorf("%s is not a valid config field", name)
	}

	field := reflect.Indirect(reflect.ValueOf(cfg)).FieldByName(fieldName)
	val, err := parseAsType(value, field)
	if err != nil {
		return fmt.Errorf("%s is not a valid value for field %s", value, name)
	}

	reflect.ValueOf(cfg).Elem().FieldByName(fieldName).Set(val)

	return writeConfig(cfg)
}

func getFieldName(cfg *ContextConfig, name string) string {
	cfgValue := reflect.Indirect(reflect.ValueOf(cfg))
	var fieldName string
	for i := 0; i < cfgValue.NumField(); i++ {
		fieldType := reflect.TypeOf(*cfg).Field(i)
		for _, tag := range strings.Split(fieldType.Tag.Get("yaml"), ",") {
			if tag == name {
				fieldName = fieldType.Name
			}
		}
	}
	return fieldName
}

func parseAsType(value string, field reflect.Value) (reflect.Value, error) {
	fieldType := field.Type()
	switch fieldType.String() {
	case "string":
		return reflect.ValueOf(value), nil
	case "[]string":
		if value == "" {
			return reflect.Zero(fieldType), nil
		}
		return reflect.Append(field, reflect.ValueOf(value)), nil
	case "*bool":
		if value == "" {
			return reflect.Zero(fieldType), nil
		}
		valBase, err := strconv.ParseBool(value)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(&valBase), nil
	default:
		return reflect.Value{}, fmt.Errorf("unsupported type: %s", fieldType)
	}
}

func writeConfig(cfg *ContextConfig) error {
	fullConfig, err := readConfig()
	if err != nil {
		return err
	}
	if global {
		fullConfig.Global = cfg
	} else {
		for i, contextCfg := range fullConfig.ContextConfigs {
			if contextCfg.Kubecontext == kubecontext {
				fullConfig.ContextConfigs[i] = cfg
			}
		}
	}
	return writeFullConfig(fullConfig)
}

func writeFullConfig(cfg *Config) error {
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

func logSetConfigForUser(out io.Writer, key string, value string) {
	if global {
		fmt.Fprintf(out, "set global value %s to %s\n", key, value)
	} else {
		fmt.Fprintf(out, "set value %s to %s for context %s\n", key, value, kubecontext)
	}
}
