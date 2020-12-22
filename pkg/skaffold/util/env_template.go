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

package util

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/sirupsen/logrus"
)

// For testing
var (
	OSEnviron = os.Environ
)

// ExpandEnvTemplate parses and executes template s with an optional environment map
func ExpandEnvTemplate(s string, envMap map[string]string) (string, error) {
	tmpl, err := ParseEnvTemplate(s)
	if err != nil {
		return "", fmt.Errorf("unable to parse template: %q: %w", s, err)
	}
	return ExecuteEnvTemplate(tmpl, envMap)
}

// ExpandEnvTemplateOrFail parses and executes template s with an optional environment map, and errors if a reference cannot be satisfied.
func ExpandEnvTemplateOrFail(s string, envMap map[string]string) (string, error) {
	tmpl, err := ParseEnvTemplate(s)
	if err != nil {
		return "", fmt.Errorf("unable to parse template: %q: %w", s, err)
	}
	tmpl = tmpl.Option("missingkey=error")
	return ExecuteEnvTemplate(tmpl, envMap)
}

// ParseEnvTemplate is a simple wrapper to parse an env template
func ParseEnvTemplate(t string) (*template.Template, error) {
	return template.New("envTemplate").Parse(t)
}

// ExecuteEnvTemplate executes an envTemplate based on OS environment variables and a custom map
func ExecuteEnvTemplate(envTemplate *template.Template, customMap map[string]string) (string, error) {
	envMap := map[string]string{}
	for _, env := range OSEnviron() {
		kvp := strings.SplitN(env, "=", 2)
		envMap[kvp[0]] = kvp[1]
	}

	for k, v := range customMap {
		envMap[k] = v
	}

	var buf bytes.Buffer
	logrus.Debugf("Executing template %v with environment %v", envTemplate, envMap)
	if err := envTemplate.Execute(&buf, envMap); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// EvaluateEnvTemplateMap parses and executes all map values as templates based on OS environment variables
func EvaluateEnvTemplateMap(args map[string]*string) (map[string]*string, error) {
	return EvaluateEnvTemplateMapWithEnv(args, nil)
}

// EvaluateEnvTemplateMapWithEnv parses and executes all map values as templates based on OS and custom environment variables
func EvaluateEnvTemplateMapWithEnv(args map[string]*string, env map[string]string) (map[string]*string, error) {
	if args == nil {
		return nil, nil
	}

	evaluated := map[string]*string{}
	for k, v := range args {
		if v == nil {
			evaluated[k] = nil
			continue
		}

		value, err := ExpandEnvTemplate(*v, env)
		if err != nil {
			return nil, fmt.Errorf("unable to get value for key %q: %w", k, err)
		}

		evaluated[k] = &value
	}

	return evaluated, nil
}

// MapToFlag parses all map values and returns them as `key=value` with the given flag
// Example: --my-flag key0=value0 --my-flag key1=value1  --my-flag key2=value2
func MapToFlag(m map[string]*string, flag string) ([]string, error) {
	kv, err := EvaluateEnvTemplateMap(m)
	if err != nil {
		return nil, fmt.Errorf("unable to evaluate build args: %w", err)
	}

	var keys []string
	for k := range kv {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var kvFlags []string
	for _, k := range keys {
		v := kv[k]
		if v == nil {
			kvFlags = append(kvFlags, flag, k)
		} else {
			kvFlags = append(kvFlags, flag, fmt.Sprintf("%s=%s", k, *v))
		}
	}

	return kvFlags, nil
}
