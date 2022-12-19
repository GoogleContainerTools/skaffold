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
	"context"
	"fmt"
	"text/template"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
)

// customTemplateTagger implements Tagger
type customTemplateTagger struct {
	RunCtx     *runcontext.RunContext
	Template   *template.Template
	Components map[string]Tagger
}

// NewCustomTemplateTagger creates a new customTemplateTagger
func NewCustomTemplateTagger(runCtx *runcontext.RunContext, t string, components map[string]Tagger) (Tagger, error) {
	tmpl, err := ParseCustomTemplate(t)
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	return &customTemplateTagger{
		RunCtx:     runCtx,
		Template:   tmpl,
		Components: components,
	}, nil
}

// GenerateTag generates a tag from a template referencing tagging strategies.
func (t *customTemplateTagger) GenerateTag(ctx context.Context, image latest.Artifact) (string, error) {
	customMap, err := t.EvaluateComponents(ctx, image)
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
func (t *customTemplateTagger) EvaluateComponents(ctx context.Context, image latest.Artifact) (map[string]string, error) {
	taggers := map[string]Tagger{}

	templateFields := GetTemplateFields(t.Template)
	for _, field := range templateFields {
		if v, ok := t.Components[field]; ok {
			if _, ok := v.(*customTemplateTagger); ok {
				return nil, fmt.Errorf("invalid component specified in custom template: %v", v)
			}
			taggers[field] = v
			continue
		}
		switch field {
		case "GIT":
			gitTagger, _ := NewGitCommit("", "", false)
			taggers[field] = gitTagger
		case "DATE":
			taggers[field] = NewDateTimeTagger("", "")
		case "SHA":
			taggers[field] = &ChecksumTagger{}
		case "INPUT_DIGEST":
			inputDigestTagger, _ := NewInputDigestTagger(t.RunCtx, graph.ToArtifactGraph(t.RunCtx.Artifacts()))
			taggers[field] = inputDigestTagger
		default:
			return nil, fmt.Errorf("no tagger available for component %+v", field)
		}
	}

	customMap := map[string]string{}
	for k, v := range taggers {
		tag, err := v.GenerateTag(ctx, image)
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
	log.Entry(context.TODO()).Debugf("Executing custom template %v with custom map %v", customTemplate, customMap)
	if err := customTemplate.Execute(&buf, customMap); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}
	return buf.String(), nil
}
