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
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestTagTemplate_GenerateTag(t *testing.T) {
	aLocalTimeStamp := time.Date(2015, 03, 07, 11, 06, 39, 123456789, time.Local)

	dateTimeExample := &dateTimeTagger{
		Format:   "2006-01-02",
		TimeZone: "UTC",
		timeFn:   func() time.Time { return aLocalTimeStamp },
	}

	envTemplateExample, _ := NewEnvTemplateTagger("{{.FOO}}")
	invalidEnvTemplate, _ := NewEnvTemplateTagger("{{.BAR}}")
	env := []string{"FOO=BAR"}

	customTemplateExample, _ := NewCustomTemplateTagger("", nil)

	tests := []struct {
		description string
		template    string
		customMap   map[string]Tagger
		expected    string
		shouldErr   bool
	}{
		{
			description: "empty template",
		},
		{
			description: "only text",
			template:    "foo-bar",
			expected:    "foo-bar",
		},
		{
			description: "only component (dateTime) in template, providing more components than necessary",
			template:    "{{.FOO}}",
			customMap:   map[string]Tagger{"FOO": dateTimeExample, "BAR": envTemplateExample},
			expected:    "2015-03-07",
		},
		{
			description: "envTemplate and sha256 as components",
			template:    "foo-{{.FOO}}-{{.BAR}}",
			customMap:   map[string]Tagger{"FOO": envTemplateExample, "BAR": &ChecksumTagger{}},
			expected:    "foo-BAR-latest",
		},
		{
			description: "using customTemplate as a component",
			template:    "{{.FOO}}",
			customMap:   map[string]Tagger{"FOO": customTemplateExample},
			shouldErr:   true,
		},
		{
			description: "faulty component, envTemplate has undefined references",
			template:    "{{.FOO}}",
			customMap:   map[string]Tagger{"FOO": invalidEnvTemplate},
			shouldErr:   true,
		},
		{
			description: "missing required components",
			template:    "{{.FOO}}",
			shouldErr:   true,
		},
		{
			description: "default component name SHA",
			template:    "{{.SHA}}",
			expected:    "latest",
		},
		{
			description: "override default components",
			template:    "{{.GIT}}-{{.DATE}}-{{.SHA}}",
			customMap:   map[string]Tagger{"GIT": dateTimeExample, "DATE": envTemplateExample, "SHA": dateTimeExample},
			expected:    "2015-03-07-BAR-2015-03-07",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.OSEnviron, func() []string { return env })

			c, err := NewCustomTemplateTagger(test.template, test.customMap)

			t.CheckNoError(err)

			tag, err := c.GenerateTag(".", "test")

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, tag)
		})
	}
}

func TestCustomTemplate_NewCustomTemplateTagger(t *testing.T) {
	tests := []struct {
		description string
		template    string
		customMap   map[string]Tagger
		shouldErr   bool
	}{
		{
			description: "valid template with nil map",
			template:    "{{.FOO}}",
		},
		{
			description: "valid template with atleast one mapping",
			template:    "{{.FOO}}",
			customMap:   map[string]Tagger{"FOO": &ChecksumTagger{}},
		},
		{
			description: "invalid template with nil mapping",
			template:    "{{.FOO",
			shouldErr:   true,
		},
		{
			description: "invalid template with atleast one mapping",
			template:    "{{.FOO",
			customMap:   map[string]Tagger{"FOO": &ChecksumTagger{}},
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			_, err := NewCustomTemplateTagger(test.template, test.customMap)
			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestCustomTemplate_ExecuteCustomTemplate(t *testing.T) {
	tests := []struct {
		description string
		template    string
		customMap   map[string]string
		expected    string
		shouldErr   bool
	}{
		{
			description: "empty template",
		},
		{
			description: "only text",
			template:    "foo-bar",
			expected:    "foo-bar",
		},
		{
			description: "only component",
			template:    "{{.FOO}}",
			expected:    "2016-02-05",
			customMap:   map[string]string{"FOO": "2016-02-05"},
		},
		{
			description: "both text and components",
			template:    "foo-{{.BAR}}",
			expected:    "foo-2016-02-05",
			customMap:   map[string]string{"BAR": "2016-02-05"},
		},
		{
			description: "component has value with len 0",
			template:    "foo-{{.BAR}}",
			expected:    "foo-",
			customMap:   map[string]string{"BAR": ""},
		},
		{
			description: "undefined component",
			template:    "foo-{{.BAR}}",
			customMap:   map[string]string{"FOO": "2016-02-05"},
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			testTemplate, err := ParseCustomTemplate(test.template)
			t.CheckNoError(err)

			got, err := ExecuteCustomTemplate(testTemplate.Option("missingkey=error"), test.customMap)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, got)
		})
	}
}

func TestCustomTemplate_ParseCustomTemplate(t *testing.T) {
	tests := []struct {
		description string
		template    string
		shouldErr   bool
	}{
		{
			description: "valid template",
			template:    "{{.FOO}}",
		},
		{
			description: "invalid template",
			template:    "{{.FOO",
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			_, err := ParseCustomTemplate(test.template)
			t.CheckError(test.shouldErr, err)
		})
	}
}
