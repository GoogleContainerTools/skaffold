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

package docker

import (
	"testing"

	v1 "k8s.io/api/core/v1"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestContainerFromImage(t *testing.T) {
	tests := []struct {
		description       string
		tag               string
		container         string
		expectedContainer v1.Container
		shouldErr         bool
	}{
		{
			description:       "single image",
			tag:               "foo:bar",
			container:         "my_container",
			expectedContainer: v1.Container{Name: "my_container", Image: "foo:bar"},
		},
		{
			description: "single image",
			tag: `multiline strings don't work\nimage: should break
`,
			container: "my_container",
			shouldErr: true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			container, _, err := containerFromImage(test.tag, test.container)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedContainer, container)
		})
	}
}
