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
	"errors"
	"fmt"
	"strings"
	"text/template"

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
func (t *envTemplateTagger) GenerateTag(_, imageName string) (string, error) {
	tag, err := util.ExecuteEnvTemplate(t.Template.Option("missingkey=error"), map[string]string{
		"IMAGE_NAME":  imageName,
		"DIGEST":      "_DEPRECATED_DIGEST_",
		"DIGEST_ALGO": "_DEPRECATED_DIGEST_ALGO_",
		"DIGEST_HEX":  "_DEPRECATED_DIGEST_HEX_",
	})
	if err != nil {
		return "", err
	}

	if strings.Contains(tag, "_DEPRECATED_DIGEST_") ||
		strings.Contains(tag, "_DEPRECATED_DIGEST_ALGO_") ||
		strings.Contains(tag, "_DEPRECATED_DIGEST_HEX_") {
		return "", errors.New("{{.DIGEST}}, {{.DIGEST_ALGO}} and {{.DIGEST_HEX}} are deprecated, image digest will now automatically be appended to image tags")
	}

	return tag, nil
}
