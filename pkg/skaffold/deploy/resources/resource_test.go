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

const (
	ZeroTimestamp int64 = 0
)

func TestUpdateTimestamp(t *testing.T) {
	dep := NewDeployment("test", "test-ns", time.Millisecond)

	// Check both updated and reported timestamp are 0
	testutil.CheckDeepEqual(t, dep.status.lastUpdated, ZeroTimestamp)
	testutil.CheckDeepEqual(t, dep.status.lastReported, ZeroTimestamp)

	// Update the status
	dep.UpdateStatus("success", "success", nil)
	// Check the updated timestamp is present
	if dep.status.lastUpdated == ZeroTimestamp {
		t.Errorf("expeceted lastUpdated timestamp to be non zero. Got %d", dep.status.lastUpdated)
	}
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
			err:         fmt.Errorf("cannot pull image"),
			expected:    "deployment/test is pending due to cannot pull image\n",
		}, {
			description: "updating a non error status",
			message:     "is waiting for container",
			expected:    "deployment/test is pending due to is waiting for container\n",
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
			expected:    "deployment/test is pending due to cannot pull image\n",
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
				// Check reported timestamp is present
				if dep.status.lastReported == ZeroTimestamp {
					t.Errorf("expeceted lastReported timestamp to be non zero. Got %d", dep.status.lastReported)
				}
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
			description: "updateTimestamp should not change for same statuses",
			old:         Status{details: "same", reason: "same", err: nil},
			new:         Status{details: "same", reason: "same", err: nil},
		}, {
			description: "updateTimestamp should not change for same statuses if details change",
			old:         Status{details: "same", reason: "same", err: nil},
			new:         Status{details: "another", reason: "same", err: nil},
		}, {
			description:  "updateTimestamp should change if reason change",
			old:          Status{details: "same", reason: "same", err: nil},
			new:          Status{details: "same", reason: "another", err: nil},
			expectChange: true,
		}, {
			description:  "updateTimestamp should change if error change",
			old:          Status{details: "same", reason: "same", err: nil},
			new:          Status{details: "same", reason: "same", err: fmt.Errorf("see this error")},
			expectChange: true,
		}, {
			description:  "updateTimestamp should change if both reason and error change",
			old:          Status{details: "same", reason: "same", err: nil},
			new:          Status{reason: "error", err: fmt.Errorf("cannot pull image")},
			expectChange: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			dep := NewDeployment("test", "test-ns", time.Millisecond)
			dep.UpdateStatus(test.old.details, test.old.reason, test.old.err)
			oldUpdateTimestamp := dep.Status().lastUpdated
			dep.UpdateStatus(test.new.details, test.new.reason, test.new.err)
			t.CheckDeepEqual(test.expectChange, oldUpdateTimestamp != dep.status.lastUpdated)
		})
	}
}
