/*
Copyright 2020 The Skaffold Authors

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

package fsnotify

import (
	"bytes"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestLogWatchToUser(t *testing.T) {
	tests := []struct {
		description string
		isActive    bool
		expected    string
	}{
		{
			description: "active notify Trigger",
			isActive:    true,
			expected:    "Watching for changes...\n",
		},
		{
			description: "inactive notify Trigger",
			isActive:    false,
			expected:    "Not watching for changes...\n",
		},
	}
	for _, test := range tests {
		out := new(bytes.Buffer)

		trigger := &Trigger{
			Interval: 10,
			isActive: func() bool {
				return test.isActive
			},
		}
		trigger.LogWatchToUser(out)

		got, want := out.String(), test.expected
		testutil.CheckDeepEqual(t, want, got)
	}
}

func Test_Debounce(t *testing.T) {
	tr := &Trigger{}
	got, want := tr.Debounce(), false
	testutil.CheckDeepEqual(t, want, got)
}

func Test_Ignore(t *testing.T) {
	tr := &Trigger{}
	testutil.CheckDeepEqual(t, false, tr.Ignore(nil))
}
