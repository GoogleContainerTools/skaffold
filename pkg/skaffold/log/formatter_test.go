/*
Copyright 2022 The Skaffold Authors

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

package log

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestParseJson(t *testing.T) {
	tests := []struct {
		name     string
		config   latest.JSONParseConfig
		line     string
		expected string
	}{
		{
			name:     "standard json parse",
			config:   latest.JSONParseConfig{Fields: []string{"message", "severity", "timestampSeconds"}},
			line:     "{\"timestampSeconds\":1643740871,\"timestampNanos\":446000000,\"severity\":\"INFO\",\"message\":\"Hello World\"}\n",
			expected: "message: Hello World, severity: INFO, timestampSeconds: 1.643740871e+09\n",
		},
		{
			name:     "mix of found and not found fields",
			config:   latest.JSONParseConfig{Fields: []string{"message", "severity", "invalid"}},
			line:     "{\"timestampSeconds\":1643740871,\"timestampNanos\":446000000,\"severity\":\"INFO\",\"message\":\"Hello World\"}\n",
			expected: "message: Hello World, severity: INFO\n",
		},
		{
			name:     "all specified fields not found in json object",
			config:   latest.JSONParseConfig{Fields: []string{"invalid"}},
			line:     "{\"valid\":\"Hello World\"}\n",
			expected: "{\"valid\":\"Hello World\"}\n",
		},
		{
			name:     "non json line input",
			config:   latest.JSONParseConfig{Fields: []string{"message", "severity"}},
			line:     "Hello World!\n",
			expected: "Hello World!\n",
		},
		{
			name:     "json input with no config",
			config:   latest.JSONParseConfig{},
			line:     "{\"timestampSeconds\":1643740871,\"timestampNanos\":446000000,\"severity\":\"INFO\",\"message\":\"Hello World\"}\n",
			expected: "{\"timestampSeconds\":1643740871,\"timestampNanos\":446000000,\"severity\":\"INFO\",\"message\":\"Hello World\"}\n",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			result := ParseJSON(test.config, test.line)

			t.CheckDeepEqual(test.expected, result)
		})
	}
}
