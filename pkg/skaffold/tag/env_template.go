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

package tag

import (
	"context"
	"fmt"
	"text/template"

	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// envTemplateTagger implements Tagger
type envTemplateTagger struct {
	Template *template.Template
}

// NewEnvTemplateTagger creates a new envTemplateTagger
func NewEnvTemplateTagger(t string) (Tagger, error) {
	tmpl, err := util.ParseEnvTemplate(t)
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	return &envTemplateTagger{
		Template: tmpl,
	}, nil
}

// GenerateTag generates a tag from a template referencing environment variables.
func (t *envTemplateTagger) GenerateTag(ctx context.Context, image latestV1.Artifact) (string, error) {
	// missingkey=error throws error when map is indexed with an undefined key
	tag, err := util.ExecuteEnvTemplate(t.Template.Option("missingkey=error"), map[string]string{
		"IMAGE_NAME": image.ImageName,
	})
	if err != nil {
		return "", err
	}

	return tag, nil
}
