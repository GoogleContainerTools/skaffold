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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
)

// envTemplateTagger implements Tagger
type envTemplateTagger struct {
	Template *template.Template
}

// NewEnvTemplateTagger creates a new envTemplateTagger
func NewEnvTemplateTagger(t string) (*envTemplateTagger, error) {
	tmpl, err := util.ParseEnvTemplate(t)
	if err != nil {
		return nil, errors.Wrap(err, "parsing template")
	}
	return &envTemplateTagger{
		Template: tmpl,
	}, nil
}

// GenerateFullyQualifiedImageName tags an image with the custom tag
func (c *envTemplateTagger) GenerateFullyQualifiedImageName(workingDir string, opts *TagOptions) (string, error) {
	customMap := map[string]string{}

	customMap["IMAGE_NAME"] = opts.ImageName
	digest := opts.Digest
	customMap["DIGEST"] = digest
	if digest != "" {
		names := strings.SplitN(digest, ":", 2)
		if len(names) >= 2 {
			customMap["DIGEST_ALGO"] = names[0]
			customMap["DIGEST_HEX"] = names[1]
		}
	}

	return util.ExecuteEnvTemplate(c.Template, customMap)
}
