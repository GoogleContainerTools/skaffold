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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/warnings"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestTagger_GenerateFullyQualifiedImageName(t *testing.T) {
	// This is for testing envTemplate
	envTemplateExample, _ := NewEnvTemplateTagger("{{.FOO}}")
	envTemplateDeprecatedExample, _ := NewEnvTemplateTagger("{{.IMAGE_NAME}}:{{.FOO}}")
	invalidEnvTemplate, _ := NewEnvTemplateTagger("{{.BAR}}")
	env := []string{"FOO=BAR"}

	// This is for testing dateTime
	aLocalTimeStamp := time.Date(2015, 03, 07, 11, 06, 39, 123456789, time.Local)
	dateTimeExample := &dateTimeTagger{
		Format:   "2006-01-02",
		TimeZone: "UTC",
		timeFn:   func() time.Time { return aLocalTimeStamp },
	}
	dateTimeExpected := "2015-03-07"

	tests := []struct {
		description      string
		imageName        string
		tagger           Tagger
		expected         string
		expectedWarnings []string
		shouldErr        bool
	}{
		{
			description: "sha256 w/o tag",
			imageName:   "test",
			tagger:      &ChecksumTagger{},
			expected:    "test:latest",
		},
		{
			description: "sha256 w/ tag",
			imageName:   "test:tag",
			tagger:      &ChecksumTagger{},
			expected:    "test:tag",
		},
		{
			description: "envTemplate",
			imageName:   "test",
			tagger:      envTemplateExample,
			expected:    "test:BAR",
		},
		{
			description:      "deprecated envTemplate",
			imageName:        "test",
			tagger:           envTemplateDeprecatedExample,
			expected:         "test:BAR",
			expectedWarnings: []string{"{{.IMAGE_NAME}} is deprecated, envTemplate's template should only specify the tag value. See https://skaffold.dev/docs/pipeline-stages/taggers/"},
		},
		{
			description: "undefined env variable",
			imageName:   "test",
			tagger:      invalidEnvTemplate,
			shouldErr:   true,
		},
		{
			description: "dateTime",
			imageName:   "test",
			tagger:      dateTimeExample,
			expected:    "test:" + dateTimeExpected,
		},
		{
			description: "dateTime",
			imageName:   "test",
			tagger: &dateTimeTagger{
				Format:   "2006-01-02",
				TimeZone: "FOO",
				timeFn:   func() time.Time { return aLocalTimeStamp },
			},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			fakeWarner := &warnings.Collect{}
			t.Override(&warnings.Printf, fakeWarner.Warnf)
			t.Override(&util.OSEnviron, func() []string { return env })

			tag, err := GenerateFullyQualifiedImageName(test.tagger, ".", test.imageName)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, tag)
			t.CheckDeepEqual(test.expectedWarnings, fakeWarner.Warnings)
		})
	}
}
