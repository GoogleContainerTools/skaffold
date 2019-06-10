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

package kubernetes

import (
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestSinceSeconds(t *testing.T) {
	var tests = []struct {
		description string
		duration    time.Duration
		expected    int64
	}{
		{"0s", 0, 1},
		{"1ms", 1 * time.Millisecond, 1},
		{"500ms", 500 * time.Millisecond, 1},
		{"999ms", 999 * time.Millisecond, 1},
		{"1s", 1 * time.Second, 1},
		{"1.1s", 1100 * time.Millisecond, 2},
		{"1.5s", 1500 * time.Millisecond, 2},
		{"1.9s", 1500 * time.Millisecond, 2},
		{"2s", 2 * time.Second, 2},
		{"10s", 10 * time.Second, 10},
		{"60s", 60 * time.Second, 60},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			since := sinceSeconds(test.duration)

			t.CheckDeepEqual(test.expected, since)
		})
	}
}
