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

package config

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestSyncRemoteCacheOption(t *testing.T) {
	tests := []struct {
		description string
		option      string
		shouldErr   bool
		clone       bool
		pull        bool
	}{
		{
			description: "always",
			option:      "always",
			clone:       true,
			pull:        true,
		},
		{
			description: "missing",
			option:      "missing",
			clone:       true,
			pull:        false,
		},
		{
			description: "never",
			option:      "never",
			clone:       false,
			pull:        false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			opt := &SyncRemoteCacheOption{}
			err := opt.Set(test.option)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, !test.clone, opt.CloneDisabled())
			t.CheckErrorAndDeepEqual(test.shouldErr, err, !test.pull, opt.FetchDisabled())
		})
	}
}
