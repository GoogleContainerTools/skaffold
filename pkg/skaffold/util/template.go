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
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
)

// A templater containing a base set of available variables
// and a configured root template
type Templater struct {
	valueMap map[string]string
	template *template.Template
}

// Create a new Templater with a `customMap` containing values then available
// to templates
func NewTemplater(customMap map[string]string) Templater {
	// Create central templater
	tpl := template.New("helmTemplate").Funcs(FuncsMap).Funcs(sprig.FuncMap())

	valueMap := map[string]string{}

	// Load environment variables
	// TODO maybe use misc.EvaluateEnv ?
	for _, env := range OSEnviron() {
		kvp := strings.SplitN(env, "=", 2)
		valueMap[kvp[0]] = kvp[1]
	}

	// Combine with custom map
	for k, v := range customMap {
		valueMap[k] = v
	}

	return Templater{template: tpl, valueMap: valueMap}
}

// render a single string `s` with the templater considering the provided `overrides`.
// Overrides always take precedence.
func (tpl *Templater) Render(s string, overrides map[string]interface{}) (string, error) {
	// Clone to have a local template for this invocation
	templater, err := tpl.template.Clone()
	if err != nil {
		return "", fmt.Errorf("unable to clone template: %q: %w", s, err)
	}

	// Parse the template
	tmpl, err := templater.Parse(s)
	if err != nil {
		return "", fmt.Errorf("unable to parse template: %q: %w", s, err)
	}
	tmpl = tmpl.Option("missingkey=invalid")

	// Compute values for this invocation from the global map and overrides
	values := map[string]interface{}{}
	for k, v := range tpl.valueMap {
		values[k] = v
	}

	// Overrides added last, to take precedence
	for k, v := range overrides {
		values[k] = v
	}

	// Execute the actual template with t he provided values
	var buf bytes.Buffer
	log.Entry(context.TODO()).Debugf("Executing template %v with environment %v", tmpl, tpl.valueMap)
	if err := tmpl.Execute(&buf, values); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// renderNonEmpty parses and executes a template with overrides and errors if
// the resulting string is empty or only contains the "<no value>" marker.
func (tpl *Templater) RenderNonEmpty(s string, overrides map[string]interface{}) (string, error) {
	// Render or bail
	templated, err := tpl.Render(s, overrides)
	if err != nil {
		return "", err
	}

	// Check for failing template
	if templated == "" || strings.Contains(templated, "<no value>") {
		return "", fmt.Errorf("field evaluates to empty")
	}

	return templated, nil
}
