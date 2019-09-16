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
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/pkg/errors"
)

func TestReportSinceLastUpdated(t *testing.T) {
	var tests = []struct {
		description string
		message     string
		err         error
		expected    string
	}{
		{
			description: "updating an error status",
			message:     "cannot pull image",
			err:         errors.New("cannot pull image"),
			expected:    "test-ns:deployment/test cannot pull image",
		},
		{
			description: "updating a non error status",
			message:     "is waiting for container",
			expected:    "test-ns:deployment/test is waiting for container",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			dep := NewDeployment("test", "test-ns", 1)
			dep.UpdateStatus(test.message, test.err)
			t.CheckDeepEqual(test.expected, dep.ReportSinceLastUpdated())
			// Check reported is set to true.
			t.CheckDeepEqual(true, dep.status.reported)
		})
	}
}

func TestReportSinceLastUpdatedMultipleTimes(t *testing.T) {
	var tests = []struct {
		description string
		times       int
		expected    string
	}{
		{
			description: "report first time should return status",
			times:       1,
			expected:    "test-ns:deployment/test cannot pull image",
		},
		{
			description: "report 2nd time should not return",
			times:       2,
			expected:    "",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			dep := NewDeployment("test", "test-ns", 1)
			dep.UpdateStatus("cannot pull image", nil)
			var actual string
			for i := 0; i < test.times; i++ {
				actual = dep.ReportSinceLastUpdated()
			}
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}
