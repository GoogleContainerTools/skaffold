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
func (t *envTemplateTagger) GenerateFullyQualifiedImageName(workingDir string, opts *Options) (string, error) {
	customMap := CreateEnvVarMap(opts.ImageName, opts.Digest)
	return util.ExecuteEnvTemplate(t.Template, customMap)
}

// CreateEnvVarMap creates a set of environment variables for use in Templates from the given
// image name and digest
func CreateEnvVarMap(imageName string, digest string) map[string]string {
	customMap := map[string]string{}
	customMap["IMAGE_NAME"] = imageName
	customMap["DIGEST"] = digest
	if digest != "" {
		names := strings.SplitN(digest, ":", 2)
		if len(names) >= 2 {
			customMap["DIGEST_ALGO"] = names[0]
			customMap["DIGEST_HEX"] = names[1]
		} else {
			customMap["DIGEST_HEX"] = digest
		}
	}
	return customMap
}
