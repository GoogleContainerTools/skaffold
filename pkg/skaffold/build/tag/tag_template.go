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
