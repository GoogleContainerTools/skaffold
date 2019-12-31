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

package main

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestPrintMan(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	printMan(&stdout, &stderr)
	output := stdout.String()

	// Sanity checks
	testutil.CheckDeepEqual(t, "", stderr.String())
	testutil.CheckContains(t, "skaffold build", output)
	testutil.CheckContains(t, "skaffold run", output)
	testutil.CheckContains(t, "skaffold dev", output)
	testutil.CheckContains(t, "Env vars", output)

	// Compare to current man page
	header, err := ioutil.ReadFile(filepath.Join("..", "..", "docs", "content", "en", "docs", "references", "cli", "index_header"))
	testutil.CheckError(t, false, err)
	header = bytes.Replace(header, []byte("\r\n"), []byte("\n"), -1)

	expected, err := ioutil.ReadFile(filepath.Join("..", "..", "docs", "content", "en", "docs", "references", "cli", "_index.md"))
	testutil.CheckError(t, false, err)
	expected = bytes.Replace(expected, []byte("\r\n"), []byte("\n"), -1)

	if string(expected) != string(header)+output {
		t.Error("You have skaffold command changes but haven't generated the CLI reference docs. Please run ./hack/generate-man.sh and commit the results!")
	}
}
