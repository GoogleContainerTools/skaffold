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
	"fmt"
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

func TestEqual(t *testing.T) {
	var tests = []struct {
		description string
		old         Status
		new         Status
		expected    bool
	}{
		{
			description: "status should be same for same details and error",
			old:         Status{details: "Waiting for 0/1 replicas to be available...", err: nil},
			new:         Status{details: "Waiting for 0/1 replicas to be available...", err: nil},
			expected:    true,
		},
		{
			description: "status should be new if error messages are same",
			old:         Status{details: "same", err: errors.New("same error")},
			new:         Status{details: "same", err: errors.New("same error")},
			expected:    true,
		},
		{
			description: "status should be new if error is different",
			old:         Status{details: "same", err: nil},
			new:         Status{details: "same", err: fmt.Errorf("see this error")},
		},
		{
			description: "status should be new if details and err are different",
			old:         Status{details: "same", err: nil},
			new:         Status{details: "same", err: fmt.Errorf("see this error")},
		},
		{
			description: "status should be new if details change",
			old:         Status{details: "same", err: nil},
			new:         Status{details: "error", err: nil},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.CheckDeepEqual(test.expected, test.old.Equal(test.new))
		})
	}
}
