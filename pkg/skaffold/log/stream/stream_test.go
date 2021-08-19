/*
Copyright 2021 The Skaffold Authors

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

package stream

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestIsEmptyOrContainerNotReady(t *testing.T) {
	tests := []struct {
		description string
		line        string
		expected    bool
	}{
		{
			description: "empty line",
			line:        "",
			expected:    true,
		},
		{
			description: "container not ready",
			line:        "rpc error: code = Unknown desc = Error: No such container: fa7802b2206f84f4f1e166a7a640523a281031c4c95d1709d38d62680391b97c",
			expected:    true,
		},
		{
			description: "logs could not be retrieved",
			line:        "unable to retrieve container logs for fa7802b2206f84f4f1e166a7a640523a281031c4c95d1709d38d62680391b97c",
			expected:    true,
		},
		{
			description: "logs could not be retrieved",
			line:        "Unable to retrieve container logs for fa7802b2206f84f4f1e166a7a640523a281031c4c95d1709d38d62680391b97c",
			expected:    true,
		},
		{
			description: "actual log",
			line:        "log line",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.CheckDeepEqual(test.expected, isEmptyOrContainerNotReady(test.line))
		})
	}
}
