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

package gcb

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha3"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGuessProjectID(t *testing.T) {
	var tests = []struct {
		description string
		config      *v1alpha3.GoogleCloudBuild
		artifact    *v1alpha3.Artifact
		expected    string
		shouldErr   bool
	}{
		{
			description: "fixed projectId",
			config:      &v1alpha3.GoogleCloudBuild{ProjectID: "fixed"},
			artifact:    &v1alpha3.Artifact{ImageName: "any"},
			expected:    "fixed",
		},
		{
			description: "gcr.io",
			config:      &v1alpha3.GoogleCloudBuild{},
			artifact:    &v1alpha3.Artifact{ImageName: "gcr.io/project/image"},
			expected:    "project",
		},
		{
			description: "eu.gcr.io",
			config:      &v1alpha3.GoogleCloudBuild{},
			artifact:    &v1alpha3.Artifact{ImageName: "gcr.io/project/image"},
			expected:    "project",
		},
		{
			description: "docker hub",
			config:      &v1alpha3.GoogleCloudBuild{},
			artifact:    &v1alpha3.Artifact{ImageName: "project/image"},
			shouldErr:   true,
		},
		{
			description: "invalid GCR image",
			config:      &v1alpha3.GoogleCloudBuild{},
			artifact:    &v1alpha3.Artifact{ImageName: "gcr.io"},
			shouldErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			builder := NewBuilder(test.config)

			projectID, err := builder.guessProjectID(test.artifact)

			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, projectID)
		})
	}
}
