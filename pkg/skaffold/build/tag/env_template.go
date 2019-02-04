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
	"github.com/sirupsen/logrus"
)

// for testing
var warner Warner = &logrusWarner{}

// Warner shows warnings.
type Warner interface {
	Warnf(format string, args ...interface{})
}

type logrusWarner struct{}

func (l *logrusWarner) Warnf(format string, args ...interface{}) {
	logrus.Warnf(format, args...)
}

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
func (t *envTemplateTagger) GenerateFullyQualifiedImageName(workingDir, imageName string) (string, error) {
	tag, err := util.ExecuteEnvTemplate(t.Template, map[string]string{
		"IMAGE_NAME":  imageName,
		"DIGEST":      "[DEPRECATED_DIGEST]",
		"DIGEST_ALGO": "[DEPRECATED_DIGEST_ALGO]",
		"DIGEST_HEX":  "[DEPRECATED_DIGEST_HEX]",
	})
	if err != nil {
		return "", err
	}

	if strings.Contains(tag, "[DEPRECATED_DIGEST]") ||
		strings.Contains(tag, "[DEPRECATED_DIGEST_ALGO]") ||
		strings.Contains(tag, "[DEPRECATED_DIGEST_HEX]") {
		warner.Warnf("{{.DIGEST}}, {{.DIGEST_ALGO}} and {{.DIGEST_HEX}} are deprecated, image digest will now automatically be apppended to image tags")

		switch {
		case strings.HasSuffix(tag, "@[DEPRECATED_DIGEST]"):
			tag = strings.TrimSuffix(tag, "@[DEPRECATED_DIGEST]")

		case strings.HasSuffix(tag, "@[DEPRECATED_DIGEST_ALGO]:[DEPRECATED_DIGEST_HEX]"):
			tag = strings.TrimSuffix(tag, "@[DEPRECATED_DIGEST_ALGO]:[DEPRECATED_DIGEST_HEX]")
		}
	}

	return tag, nil
}
