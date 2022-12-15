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

package main

import (
	"bytes"
	"os"
	"path"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestUpdateChangelog(t *testing.T) {
	data := changelogData{
		SkaffoldVersion: "1.11.1",
		Date:            "01/11/2001",
		SchemaString:    "\nSomething about a schema\n",
	}

	// Read original `testdata/changelog.md` and write it back after test
	changelogB, _ := os.ReadFile(path.Join("testdata", "changelog.md"))
	defer os.WriteFile(path.Join("testdata", "changelog.md"), changelogB, 0644)

	// function should only error on standard lib errors
	err := updateChangelog(path.Join("testdata", "changelog.md"), path.Join("template.md"), data)
	if err != nil {
		t.Fatalf("%s", err)
	}

	gotB, _ := os.ReadFile(path.Join("testdata", "changelog.md"))
	wantB, _ := os.ReadFile(path.Join("testdata", "expected.md"))
	gotB = bytes.ReplaceAll(gotB, []byte("\r\n"), []byte("\n"))
	wantB = bytes.ReplaceAll(wantB, []byte("\r\n"), []byte("\n"))

	testutil.CheckDeepEqual(t, wantB, gotB)
}
