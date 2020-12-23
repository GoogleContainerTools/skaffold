/*
Copyright 2020 The Skaffold Authors

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

package tag

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/sirupsen/logrus"
)

// customTemplateTagger implements Tagger
type customTemplateTagger struct {
	Template   *template.Template
	Components map[string]Tagger
}

// NewCustomTemplateTagger creates a new customTemplateTagger
func NewCustomTemplateTagger(t string, components map[string]Tagger) (Tagger, error) {
	tmpl, err := ParseCustomTemplate(t)
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	return &customTemplateTagger{
		Template:   tmpl,
		Components: components,
	}, nil
}

// GenerateTag generates a tag from a template referencing tagging strategies.
func (t *customTemplateTagger) GenerateTag(workingDir, imageName string) (string, error) {
	customMap, err := t.EvaluateComponents(workingDir, imageName)
	if err != nil {
		return "", err
	}

	// missingkey=error throws error when map is indexed with an undefined key
	tag, err := ExecuteCustomTemplate(t.Template.Option("missingkey=error"), customMap)
	if err != nil {
		return "", err
	}

	return tag, nil
}

// EvaluateComponents creates a custom mapping of component names to their tagger string representation.
func (t *customTemplateTagger) EvaluateComponents(workingDir, imageName string) (map[string]string, error) {
	customMap := map[string]string{}

	gitTagger, _ := NewGitCommit("", "", false)
	dateTimeTagger := NewDateTimeTagger("", "")

	for k, v := range map[string]Tagger{"GIT": gitTagger, "DATE": dateTimeTagger, "SHA": &ChecksumTagger{}} {
		tag, _ := v.GenerateTag(workingDir, imageName)
		customMap[k] = tag
	}

	for k, v := range t.Components {
		if _, ok := v.(*customTemplateTagger); ok {
			return nil, fmt.Errorf("invalid component specified in custom template: %v", v)
		}
		tag, err := v.GenerateTag(workingDir, imageName)
		if err != nil {
			return nil, fmt.Errorf("evaluating custom template component: %w", err)
		}
		customMap[k] = tag
	}
	return customMap, nil
}

// ParseCustomTemplate is a simple wrapper to parse an custom template.
func ParseCustomTemplate(t string) (*template.Template, error) {
	return template.New("customTemplate").Parse(t)
}

// ExecuteCustomTemplate executes a customTemplate against a custom map.
func ExecuteCustomTemplate(customTemplate *template.Template, customMap map[string]string) (string, error) {
	var buf bytes.Buffer
	logrus.Debugf("Executing custom template %v with custom map %v", customTemplate, customMap)
	if err := customTemplate.Execute(&buf, customMap); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}
	return buf.String(), nil
}
