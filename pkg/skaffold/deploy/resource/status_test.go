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

package resource

import (
	"errors"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestString(t *testing.T) {
	var tests = []struct {
		description string
		details     string
		err         error
		expected    string
	}{
		{
			description: "should return error string if error is set",
			err:         errors.New("some error"),
			expected:    "some error",
		},
		{
			description: "should return error details if error is not set",
			details:     "details",
			expected:    "details",
		},
		{
			description: "should return error if both details and error are set",
			details:     "error details",
			err:         errors.New("error happened due to something"),
			expected:    "error happened due to something",
		},
		{
			description: "should return empty string if all empty",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			status := newStatus(test.details, test.err)
			t.CheckDeepEqual(test.expected, status.String())
		})
	}
}
