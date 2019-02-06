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

package runner

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestIsTargetImage(t *testing.T) {
	var tests = []struct {
		description   string
		targetImages  []string
		expectedMatch bool
	}{
		{
			description:   "match all",
			targetImages:  nil,
			expectedMatch: true,
		},
		{
			description:   "match full name",
			targetImages:  []string{"domain/image"},
			expectedMatch: true,
		},
		{
			description:   "match partial name",
			targetImages:  []string{"image"},
			expectedMatch: true,
		},
		{
			description:   "match any",
			targetImages:  []string{"other", "image"},
			expectedMatch: true,
		},
		{
			description:   "no match",
			targetImages:  []string{"other"},
			expectedMatch: false,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			runner := &SkaffoldRunner{
				opts: &config.SkaffoldOptions{
					TargetImages: test.targetImages,
				},
			}

			match := runner.IsTargetImage(&latest.Artifact{
				ImageName: "domain/image",
			})

			testutil.CheckDeepEqual(t, test.expectedMatch, match)
		})
	}
}
