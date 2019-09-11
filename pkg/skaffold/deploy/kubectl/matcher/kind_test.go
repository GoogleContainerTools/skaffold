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

package matcher

import (
	"bytes"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/sirupsen/logrus"
)

func TestKindIsMatchKey(t *testing.T) {
	tests := []struct {
		description string
		key         string
		expected    bool
	}{
		{
			description: "returns false for kind key",
			key:         "kind",
			expected:    true,
		},
		{
			description: "returns false for other key",
			key:         "not-kind",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			actual := New([]string{}).IsMatchKey(test.key)
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

func TestKindMatches(t *testing.T) {
	tests := []struct {
		description string
		blacklist   []string
		value       interface{}
		expectedLog string
		expected    bool
	}{
		{
			description: "returns true for value not in the list",
			blacklist:   []string{"taboo", "taboo-1"},
			value:       "test",
			expected:    true,
		},
		{
			description: "returns false for value in the list",
			blacklist:   []string{"taboo", "taboo-1"},
			value:       "taboo",
		},
		{
			description: "returns true for value not in the list and not string",
			blacklist:   []string{"taboo", "taboo-1"},
			value:       1,
			expectedLog: "1 is type int but type string expected for key `kind`",
			expected:    true,
		},
		{
			description: "returns true for value not in the list and not string",
			blacklist:   []string{"taboo", "taboo-1"},
			value:       struct{}{},
			expectedLog: "{} is type struct {} but type string expected for key `kind`",
			expected:    true,
		},
	}
	for _, test := range tests {
		var buf bytes.Buffer
		logrus.SetOutput(&buf)
		logrus.SetLevel(logrus.DebugLevel)
		testutil.Run(t, test.description, func(t *testutil.T) {
			actual := New(test.blacklist).Matches(test.value)
			t.CheckDeepEqual(test.expected, actual)
			t.CheckContains(test.expectedLog, buf.String())
		})
	}
}
