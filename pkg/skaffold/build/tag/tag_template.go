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

// templateTagger implements Tagger
type templateTagger struct {
	Template   *template.Template
	Components map[string]Tagger
}

// NewTemplateTagger creates a new TemplateTagger
func NewTemplateTagger(t string, components map[string]Tagger) (Tagger, error) {
	tmpl, err := ParseTagTemplate(t)
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	return &templateTagger{
		Template:   tmpl,
		Components: components,
	}, nil
}

// GenerateTag generates a tag from a template referencing tagging strategies.
func (t *templateTagger) GenerateTag(workingDir, imageName string) (string, error) {
	customMap, err := t.EvaluateComponents(workingDir, imageName)
	if err != nil {
		return "", err
	}

	// missingkey=error throws error when map is indexed with an undefined key
	tag, err := ExecuteTagTemplate(t.Template.Option("missingkey=error"), customMap)
	if err != nil {
		return "", err
	}

	return tag, nil
}

// EvaluateComponents creates a custom mapping of component names to their tagger string representation.
func (t *templateTagger) EvaluateComponents(workingDir, imageName string) (map[string]string, error) {
	customMap := map[string]string{}

	gitTagger, _ := NewGitCommit("", "")
	dateTimeTagger := NewDateTimeTagger("", "")

	for k, v := range map[string]Tagger{"GIT": gitTagger, "DATE": dateTimeTagger, "SHA": &ChecksumTagger{}} {
		tag, _ := v.GenerateTag(workingDir, imageName)
		customMap[k] = tag
	}

	for k, v := range t.Components {
		if _, ok := v.(*templateTagger); ok {
			return nil, fmt.Errorf("invalid component specified in tag template: %v", v)
		}
		tag, err := v.GenerateTag(workingDir, imageName)
		if err != nil {
			return nil, fmt.Errorf("evaluating tag template component: %w", err)
		}
		customMap[k] = tag
	}
	return customMap, nil
}

// ParseTagTemplate is a simple wrapper to parse an tag template.
func ParseTagTemplate(t string) (*template.Template, error) {
	return template.New("tagTemplate").Parse(t)
}

// ExecuteTagTemplate executes a tagTemplate against a custom map.
func ExecuteTagTemplate(tagTemplate *template.Template, customMap map[string]string) (string, error) {
	var buf bytes.Buffer
	logrus.Debugf("Executing tag template %v with custom map %v", tagTemplate, customMap)
	if err := tagTemplate.Execute(&buf, customMap); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}
	return buf.String(), nil
}
