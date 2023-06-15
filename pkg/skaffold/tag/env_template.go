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
	"strings"
	"text/template"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
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
func (t *envTemplateTagger) GenerateTag(ctx context.Context, image latest.Artifact) (string, error) {
	// The method with missingkey=invalid option does not fail if the referenced variable does not exist in the map.
	// It will be replaced by <no value> if no other Go template method handles the missing key.
	// This gives us an opportunity to handle missing keys with other Go template methods. For example, the default method
	// can be used to set a default value for a missing key. The evaluation of {{default "bar" .FOO}} should be "bar" and
	// tagging should succeed if the environment variable .FOO does not exist.
	tag, err := util.ExecuteEnvTemplate(t.Template.Option("missingkey=invalid"), map[string]string{
		"IMAGE_NAME": image.ImageName,
	})
	if err != nil {
		return "", err
	}
	// return error due to missing keys
	if strings.Contains(tag, "<no value>") {
		return "", fmt.Errorf("environment variables missing for template keys")
	}

	return tag, nil
}
