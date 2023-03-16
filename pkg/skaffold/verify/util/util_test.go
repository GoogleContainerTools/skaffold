/*
Copyright 2023 The Skaffold Authors

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

package util

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestParseEnvVariablesFromFile(t *testing.T) {
	tests := []struct {
		description string
		text        string
		expected    map[string]string
		shouldErr   bool
	}{
		{
			description: "parsing dotenv file text works and expected map was created",
			text:        "FOO=my-foo-var",
			expected:    map[string]string{"FOO": "my-foo-var"},
			shouldErr:   false,
		},
		{
			description: "parsing dotenv file fails works as file is malformed",
			text:        "MALFORMED",
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpfile := t.TempFile("", []byte(test.text))
			envMap, err := ParseEnvVariablesFromFile(tmpfile)
			t.CheckError(test.shouldErr, err)
			t.CheckDeepEqual(test.expected, envMap)
		})
	}
}
