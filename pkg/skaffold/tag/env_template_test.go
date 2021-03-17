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

package tag

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/warnings"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestEnvTemplateTagger_GenerateTag(t *testing.T) {
	tests := []struct {
		description      string
		template         string
		imageName        string
		env              []string
		expected         string
		expectedWarnings []string
		shouldErr        bool
	}{
		{
			description: "empty env",
			template:    "{{.IMAGE_NAME}}",
			imageName:   "foo",
			expected:    "foo",
		},
		{
			description: "env",
			template:    "{{.FOO}}-{{.BAZ}}:latest",
			env:         []string{"FOO=BAR", "BAZ=BAT"},
			imageName:   "foo",
			expected:    "BAR-BAT:latest",
		},
		{
			description: "missing env",
			template:    "{{.FOO}}:latest",
			shouldErr:   true,
		},
		{
			description: "opts precedence",
			template:    "{{.IMAGE_NAME}}-{{.FROM_ENV}}:latest",
			env:         []string{"FROM_ENV=FOO", "IMAGE_NAME=BAT"},
			imageName:   "image_name",
			expected:    "image_name-FOO:latest",
		},
		{
			description: "ignore @{{.DIGEST}} suffix",
			template:    "{{.IMAGE_NAME}}:tag@{{.DIGEST}}",
			imageName:   "foo",
			shouldErr:   true,
		},
		{
			description: "ignore @{{.DIGEST_ALGO}}:{{.DIGEST_HEX}} suffix",
			template:    "{{.IMAGE_NAME}}:tag@{{.DIGEST_ALGO}}:{{.DIGEST_HEX}}",
			imageName:   "image_name",
			shouldErr:   true,
		},
		{
			description: "digest is deprecated",
			template:    "{{.IMAGE_NAME}}:{{.DIGEST}}",
			imageName:   "foo",
			shouldErr:   true,
		},
		{
			description: "digest algo is deprecated",
			template:    "{{.IMAGE_NAME}}:{{.DIGEST_ALGO}}",
			imageName:   "foo",
			shouldErr:   true,
		},
		{
			description: "digest hex is deprecated",
			template:    "{{.IMAGE_NAME}}:{{.DIGEST_HEX}}",
			imageName:   "foo",
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			fakeWarner := &warnings.Collect{}
			t.Override(&warnings.Printf, fakeWarner.Warnf)
			t.Override(&util.OSEnviron, func() []string { return test.env })

			c, err := NewEnvTemplateTagger(test.template)
			t.CheckNoError(err)

			got, err := c.GenerateTag(".", test.imageName)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, got)
			t.CheckDeepEqual(test.expectedWarnings, fakeWarner.Warnings)
		})
	}
}

func TestNewEnvTemplateTagger(t *testing.T) {
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
			_, err := NewEnvTemplateTagger(test.template)

			t.CheckError(test.shouldErr, err)
		})
	}
}
