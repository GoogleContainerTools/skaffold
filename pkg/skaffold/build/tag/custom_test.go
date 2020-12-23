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

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestCustomTag_GenerateTag(t *testing.T) {
	tests := []struct {
		description string
		c           *CustomTag
		expected    string
		shouldErr   bool
	}{
		{
			description: "valid custom tag",
			c: &CustomTag{
				Tag: "1.2.3-beta",
			},
			expected: "1.2.3-beta",
		},
		{
			description: "invalid custom tag",
			c:           &CustomTag{},
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		tag, err := test.c.GenerateTag(".", "test")
		testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, tag)
	}
}
