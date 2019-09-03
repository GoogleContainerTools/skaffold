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

package resources

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestUpdateTimestamp(t *testing.T) {
	dep := NewDeployment("test", "test-ns", time.Millisecond)

	// Check updated bool is false
	testutil.CheckDeepEqual(t, false, dep.status.updated)

	// Update the status
	dep.UpdateStatus("success", "success", nil)
	// Check the updated bool is true
	testutil.CheckDeepEqual(t, true, dep.status.updated)
}

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
			err:         fmt.Errorf("cannot pull image"),
			expected:    " - deployment/test is pending due to cannot pull image\n",
		}, {
			description: "updating a non error status",
			message:     "is waiting for container",
			expected:    " - deployment/test is pending due to is waiting for container\n",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			dep := NewDeployment("test", "test-ns", time.Millisecond)
			out := new(bytes.Buffer)
			dep.UpdateStatus(test.message, test.message, test.err)
			dep.ReportSinceLastUpdated(out)
			t.CheckDeepEqual(test.expected, out.String())
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
			description: "report first time should write to out",
			times:       1,
			expected:    " - deployment/test is pending due to cannot pull image\n",
		}, {
			description: "report 2nd time should not write to out",
			times:       2,
			expected:    "",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			dep := NewDeployment("test", "test-ns", time.Millisecond)
			dep.UpdateStatus("cannot pull image", "err", nil)
			var out *bytes.Buffer
			for i := 0; i < test.times; i++ {
				out = new(bytes.Buffer)
				dep.ReportSinceLastUpdated(out)
				// Check reported timestamp is set to false
				t.CheckDeepEqual(false, dep.status.updated)
			}
			t.CheckDeepEqual(test.expected, out.String())
		})
	}
}

func TestUpdateStatus(t *testing.T) {
	var tests = []struct {
		description  string
		old          Status
		new          Status
		expectChange bool
	}{
		{
			description:  "updated should be false for same statuses",
			old:          Status{details: "Waiting for 0/1 replicas to be available...", reason: "Waiting for 0/1 replicas to be available...", err: nil},
			new:          Status{details: "Waiting for 0/1 replicas to be available...", reason: "Waiting for 0/1 replicas to be available...", err: nil},
			expectChange: false,
		}, {
			description:  "updated should be true if reason changes",
			old:          Status{details: "same", reason: "same", err: nil},
			new:          Status{details: "same", reason: "another", err: nil},
			expectChange: true,
		}, {
			description: "updated should be false if reason is same",
			old:         Status{details: "same", reason: "same", err: nil},
			new:         Status{details: "same", reason: "same", err: fmt.Errorf("see this error")},
		}, {
			description:  "updated should be true if reason and err change",
			old:          Status{details: "same", reason: "same", err: nil},
			new:          Status{details: "same", reason: "another", err: fmt.Errorf("see this error")},
			expectChange: true,
		}, {
			description:  "updated should be true if both reason and details change",
			old:          Status{details: "same", reason: "same", err: nil},
			new:          Status{details: "error", reason: "error", err: nil},
			expectChange: true,
		}, {
			description:  "updated should be false if both reason has a new line",
			old:          Status{details: "same", reason: "same\n", err: nil},
			new:          Status{details: "same", reason: "same", err: nil},
			expectChange: false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			dep := NewDeployment("test", "test-ns", time.Millisecond).WithStatus(test.old)
			dep.UpdateStatus(test.new.details, test.new.reason, test.new.err)
			t.CheckDeepEqual(test.expectChange, dep.status.updated)
		})
	}
}
