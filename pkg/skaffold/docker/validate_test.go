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

package docker

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestValidateDockerfile(t *testing.T) {
	var tests = []struct {
		description    string
		content        string
		fileToValidate string
		expectedValid  bool
	}{
		{
			description:    "valid",
			content:        "FROM scratch",
			fileToValidate: "Dockerfile",
			expectedValid:  true,
		},
		{
			description:    "invalid command",
			content:        "GARBAGE",
			fileToValidate: "Dockerfile",
			expectedValid:  false,
		},
		{
			description:    "not found",
			fileToValidate: "Unknown",
			expectedValid:  false,
		},
		{
			description:    "invalid file",
			content:        "#escape",
			fileToValidate: "Dockerfile",
			expectedValid:  false,
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			tmp, delete := testutil.NewTempDir(t)
			defer delete()

			tmp.Write("Dockerfile", test.content)

			valid := ValidateDockerfile(tmp.Path(test.fileToValidate))

			testutil.CheckDeepEqual(t, test.expectedValid, valid)
		})
	}
}
