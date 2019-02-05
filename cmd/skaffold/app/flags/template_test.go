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

package flags

import (
	"bytes"
	"testing"
	"text/template"
)

type templateData struct {
	Field string
}

var (
	data             = &templateData{Field: "test"}
	rawTemplate      = "{{.Field}}"
	expectedFlagType = "*flags.TemplateFlag"
)

func TestNewTemplateFlag(t *testing.T) {
	actual := &bytes.Buffer{}
	expected := &bytes.Buffer{}

	flag := NewTemplateFlag(rawTemplate, nil)
	if err := flag.Template().Execute(actual, &data); err != nil {
		t.Errorf("Error parsing template from flag: %s", err)
	}
	if err := template.Must(template.New("template").Parse(rawTemplate)).Execute(expected, &data); err != nil {
		t.Fatalf("error parsing test template %s", err)
	}

	if actual.String() != expected.String() {
		t.Errorf("Template output did not match. Expected %s, Actual %s", expected.String(), actual.String())
	}
}

func TestTemplateSet(t *testing.T) {
	flag := &TemplateFlag{}
	if err := flag.Set(rawTemplate); err != nil {
		t.Errorf("Error setting flag value: %s", err)
	}

	if err := flag.Set("{{start}} bad template"); err == nil {
		t.Errorf("Expected error setting flag but got none.")
	}
}

func TestTemplateString(t *testing.T) {
	flag := NewTemplateFlag(rawTemplate, nil)
	if rawTemplate != flag.String() {
		t.Errorf("Flag String() does not match. Expected %s, Actual %s", rawTemplate, flag.String())
	}
}

func TestTemplateType(t *testing.T) {
	flag := &TemplateFlag{}
	if flag.Type() != expectedFlagType {
		t.Errorf("Flag returned wrong type. Expected %s, Actual %s", expectedFlagType, flag.Type())
	}
}
