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

package schema

import (
	"bytes"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/fs"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestPrint(t *testing.T) {
	fakeFS := &testutil.FakeFileSystem{
		Files: map[string][]byte{
			"assets/schemas_generated/v1.json": []byte("{SCHEMA}"),
		},
	}

	testutil.Run(t, "found", func(t *testutil.T) {
		fs.AssetsFS = fakeFS

		var out bytes.Buffer
		err := Print(&out, "skaffold/v1")

		t.CheckNoError(err)
		t.CheckDeepEqual("{SCHEMA}", out.String())
	})

	testutil.Run(t, "not found", func(t *testutil.T) {
		fs.AssetsFS = fakeFS

		var out bytes.Buffer
		err := Print(&out, "skaffold/v0")

		t.CheckErrorContains("schema \"skaffold/v0\" not found", err)
		t.CheckEmpty(out.String())
	})
}
