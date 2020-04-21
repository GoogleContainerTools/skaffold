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
		return "", fmt.Errorf("executing template: %w", err)
	}
	return buf.String(), nil
}
