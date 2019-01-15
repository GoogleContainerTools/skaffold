/*
Copyright 2018 The Skaffold Authors

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

package runner

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestShouldWatch(t *testing.T) {
	var tests = []struct {
		description   string
		watch         []string
		expectedMatch bool
	}{
		{
			description:   "match all",
			watch:         nil,
			expectedMatch: true,
		},
		{
			description:   "match full name",
			watch:         []string{"domain/image"},
			expectedMatch: true,
		},
		{
			description:   "match partial name",
			watch:         []string{"image"},
			expectedMatch: true,
		},
		{
			description:   "match any",
			watch:         []string{"other", "image"},
			expectedMatch: true,
		},
		{
			description:   "no match",
			watch:         []string{"other"},
			expectedMatch: false,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			runner := &SkaffoldRunner{
				opts: &config.SkaffoldOptions{
					Watch: test.watch,
				},
			}

			match := runner.shouldWatch(&latest.Artifact{
				ImageName: "domain/image",
			})

			testutil.CheckDeepEqual(t, test.expectedMatch, match)
		})
	}
}
