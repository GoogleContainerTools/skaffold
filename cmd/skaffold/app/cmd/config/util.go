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
	"io/ioutil"
	"path/filepath"
	"reflect"

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
	return nil, fmt.Errorf("no config entry found for kube-context %s", kubecontext)
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

// IsZero reports whether a value is a zero value of its kind.
// If value.Kind() is Struct, it traverses each field of the struct
// recursively calling IsZero, returning true only if each field's IsZero
// result is also true.
// Directly adapted from https://golang.org/cl/23064.
func IsZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Bool:
		return !v.Bool()
	case reflect.Array, reflect.String:
		return v.Len() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Slice, reflect.Interface:
		return v.IsNil()
	case reflect.UnsafePointer:
		return !v.IsValid()
	case reflect.Invalid:
		return true
	}

	if v.Kind() != reflect.Struct {
		return false
	}

	// Traverse the struct and only return true
	// if all of its fields return IsZero == true
	n := v.NumField()
	for i := 0; i < n; i++ {
		vf := v.Field(i)
		if !IsZero(vf) {
			return false
		}
	}
	return true
}
