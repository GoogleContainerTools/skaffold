/*
Copyright 2018 The Skaffold Authors

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
	"strings"
	"text/template"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
)

// envTemplateTagger implements Tagger
type envTemplateTagger struct {
	Template *template.Template
}

// NewEnvTemplateTagger creates a new envTemplateTagger
func NewEnvTemplateTagger(t string) (Tagger, error) {
	if strings.Contains(t, "{{.DIGEST}}") {
		return nil, errors.New("{{.DIGEST}} is deprecated, image digest will automatically be apppended to image tags")
	}
	if strings.Contains(t, "{{.DIGEST_ALGO}}") {
		return nil, errors.New("{{.DIGEST_ALGO}} is deprecated, image digest will automatically be apppended to image tags")
	}
	if strings.Contains(t, "{{.DIGEST_HEX}}") {
		return nil, errors.New("{{.DIGEST_HEX}} is deprecated, image digest will automatically be apppended to image tags")
	}

	tmpl, err := util.ParseEnvTemplate(t)
	if err != nil {
		return nil, errors.Wrap(err, "parsing template")
	}

	return &envTemplateTagger{
		Template: tmpl,
	}, nil
}

func (t *envTemplateTagger) Labels() map[string]string {
	return map[string]string{
		constants.Labels.TagPolicy: "envTemplateTagger",
	}
}

// GenerateFullyQualifiedImageName tags an image with the custom tag
func (t *envTemplateTagger) GenerateFullyQualifiedImageName(workingDir, imageName string) (string, error) {
	return util.ExecuteEnvTemplate(t.Template, map[string]string{
		"IMAGE_NAME": imageName,
	})
}
