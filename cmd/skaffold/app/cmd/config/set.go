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
	"io"
	"reflect"
	"strconv"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

const (
	surveyFieldName = "Survey"
)

type cfgStruct struct {
	value reflect.Value
	rType reflect.Type
	idx   []int
}

func Set(ctx context.Context, out io.Writer, args []string) error {
	if err := setConfigValue(args[0], args[1]); err != nil {
		return err
	}
	logSetConfigForUser(out, args[0], args[1])
	return nil
}

func setConfigValue(name string, value string) error {
	cfg, err := getConfigForKubectxOrDefault()
	if err != nil {
		return err
	}

	fieldIdx, err := getFieldIndex(cfg, name)
	if err != nil {
		return err
	}

	field := reflect.Indirect(reflect.ValueOf(cfg)).FieldByIndex(fieldIdx)
	val, err := parseAsType(value, field)
	if err != nil {
		return fmt.Errorf("%s is not a valid value for field %s", value, name)
	}

	reflect.ValueOf(cfg).Elem().FieldByIndex(fieldIdx).Set(val)

	return writeConfig(cfg)
}

func getFieldIndex(cfg *config.ContextConfig, name string) ([]int, error) {
	cs, err := getConfigStructWithIndex(cfg)
	if err != nil {
		return nil, err
	}
	for i := 0; i < cs.value.NumField(); i++ {
		fieldType := cs.rType.Field(i)
		for _, tag := range strings.Split(fieldType.Tag.Get("yaml"), ",") {
			if tag == name {
				if f, ok := cs.rType.FieldByName(fieldType.Name); ok {
					return append(cs.idx, f.Index...), nil
				}
				return nil, fmt.Errorf("could not find config field %s", name)
			}
		}
	}
	return nil, fmt.Errorf("%s is not a valid config field", name)
}

func getConfigStructWithIndex(cfg *config.ContextConfig) (*cfgStruct, error) {
	t := reflect.TypeOf(*cfg)
	if survey {
		if cfg.Survey == nil {
			cfg.Survey = &config.SurveyConfig{}
		}
		return surveyStruct(cfg.Survey, t)
	}
	return &cfgStruct{
		value: reflect.Indirect(reflect.ValueOf(cfg)),
		rType: t,
		idx:   []int{},
	}, nil
}

func surveyStruct(s *config.SurveyConfig, t reflect.Type) (*cfgStruct, error) {
	surveyType := reflect.TypeOf(*s)
	if surveyField, ok := t.FieldByName(surveyFieldName); ok {
		return &cfgStruct{
			value: reflect.Indirect(reflect.ValueOf(s)),
			rType: surveyType,
			idx:   surveyField.Index,
		}, nil
	}
	return nil, fmt.Errorf("survey config field 'Survey' not found in config struct %s", t.Name())
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

func writeConfig(cfg *config.ContextConfig) error {
	fullConfig, err := config.ReadConfigFile(configFile)
	if err != nil {
		return err
	}
	if global {
		fullConfig.Global = cfg
	} else {
		found := false
		for i, contextCfg := range fullConfig.ContextConfigs {
			if util.RegexEqual(contextCfg.Kubecontext, kubecontext) {
				fullConfig.ContextConfigs[i] = cfg
				found = true
			}
		}
		if !found {
			fullConfig.ContextConfigs = append(fullConfig.ContextConfigs, cfg)
		}
	}
	return config.WriteFullConfig(configFile, fullConfig)
}

func logSetConfigForUser(out io.Writer, key string, value string) {
	if global {
		fmt.Fprintf(out, "set global value %s to %s\n", key, value)
	} else {
		fmt.Fprintf(out, "set value %s to %s for context %s\n", key, value, kubecontext)
	}
}
