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

package profile

import (
	"bytes"
	"context"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestList(t *testing.T) {
	tests := []struct {
		description    string
		filename       string
		filecontent    string
		expectedOutput string
	}{
		{
			description: "has profiles",
			filename:    "skaffold.yaml",
			filecontent: `apiVersion: skaffold/v2beta29
kind: Config
profiles:
  - name: p1
  - name: p2
  - name: p3
`,
			expectedOutput: "p1\n---\np2\n---\np3\n",
		},
		{
			description: "has no profiles",
			filename:    "skaffold.yaml",
			filecontent: `apiVersion: skaffold/v2beta29
kind: Config
`,
			expectedOutput: "No profiles found\n",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&filename, test.filename)

			t.NewTempDir().
				Write("skaffold.yaml", test.filecontent).
				Chdir()

			buf := &bytes.Buffer{}
			// list values
			err := List(context.Background(), buf)
			t.CheckNoError(err)

			if buf.String() != test.expectedOutput {
				t.Errorf("expecting output to be\n\n%s\nbut found\n\n%s\ninstead", test.expectedOutput, buf.String())
			}
		})
	}
}
