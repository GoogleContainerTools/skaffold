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

package generator

import (
	"bytes"
	"fmt"
	"html/template"
)

type Container struct {
	Name  string
	Image string
}

// Generate generates kubernetes resources for the given image, and returns the generated manifest string
func Generate(name string) ([]byte, error) {
	c := Container{name, name}

	t, err := template.New("deployment").Parse(yamlTemplate)
	if err != nil {
		return nil, fmt.Errorf("error parsing pod template: %w", err)
	}

	var buf bytes.Buffer
	if err = t.Execute(&buf, c); err != nil {
		return nil, fmt.Errorf("error executing template: %w", err)
	}

	return buf.Bytes(), nil
}
