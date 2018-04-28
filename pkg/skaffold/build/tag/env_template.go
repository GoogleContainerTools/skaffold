/*
Copyright 2018 Google LLC

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
	"os"
	"strings"
	"text/template"

	"github.com/sirupsen/logrus"

	"github.com/pkg/errors"
)

// EnvTemplateTagger implements Tag
type EnvTemplateTagger struct {
	Template *template.Template
}

// For testing
var environ = os.Environ

// NewEnvTemplateTagger creates a new EnvTemplateTagger
func NewEnvTemplateTagger(t string) (*EnvTemplateTagger, error) {
	tmpl, err := template.New("envTemplate").Parse(t)
	if err != nil {
		return nil, errors.Wrap(err, "parsing template")
	}
	return &EnvTemplateTagger{
		Template: tmpl,
	}, nil
}

// GenerateFullyQualifiedImageName tags an image with the custom tag
func (c *EnvTemplateTagger) GenerateFullyQualifiedImageName(workingDir string, opts *TagOptions) (string, error) {
	var buf bytes.Buffer
	envMap := map[string]string{}
	for _, env := range environ() {
		kvp := strings.SplitN(env, "=", 2)
		if len(kvp) != 2 {
			return "", fmt.Errorf("error parsing environment variables, %s does not contain an =", kvp)
		}
		envMap[kvp[0]] = kvp[1]
	}

	envMap["IMAGE_NAME"] = opts.ImageName
	digest := opts.Digest
	envMap["DIGEST"] = digest
	if digest != "" {
		names := strings.SplitN(digest, ":", 2)
		if len(names) >= 2 {
			envMap["DIGEST_ALGO"] = names[0]
			envMap["DIGEST_HEX"] = names[1]
		}
	}

	logrus.Debugf("Executing template %v with environment %v", c.Template, envMap)
	if err := c.Template.Execute(&buf, envMap); err != nil {
		return "", errors.Wrap(err, "executing template")
	}
	return buf.String(), nil
}
