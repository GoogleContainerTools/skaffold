/*
Copyright 2026 The Skaffold Authors

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

package cache

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

// cmpInputChanges lets the comparison reach inputChanges' unexported fields.
var cmpInputChanges = cmp.AllowUnexported(inputChanges{})

func TestDiffInputs(t *testing.T) {
	tests := []struct {
		description string
		previous    map[string]string
		current     map[string]string
		expected    inputChanges
	}{
		{
			description: "nothing changed",
			previous:    map[string]string{"app.py": "a"},
			current:     map[string]string{"app.py": "a"},
			expected:    inputChanges{},
		},
		{
			description: "file modified",
			previous:    map[string]string{"app.py": "a"},
			current:     map[string]string{"app.py": "b"},
			expected:    inputChanges{modified: []string{"app.py"}},
		},
		{
			description: "file added",
			previous:    map[string]string{"app.py": "a"},
			current:     map[string]string{"app.py": "a", "new.txt": "b"},
			expected:    inputChanges{added: []string{"new.txt"}},
		},
		{
			description: "file removed",
			previous:    map[string]string{"app.py": "a", "old.txt": "b"},
			current:     map[string]string{"app.py": "a"},
			expected:    inputChanges{removed: []string{"old.txt"}},
		},
		{
			description: "everything at once, each list sorted",
			previous:    map[string]string{"b": "1", "a": "2", "gone": "3", "alsogone": "4"},
			current:     map[string]string{"b": "changed", "a": "changed", "new": "5", "alsonew": "6"},
			expected: inputChanges{
				modified: []string{"a", "b"},
				added:    []string{"alsonew", "new"},
				removed:  []string{"alsogone", "gone"},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.CheckDeepEqual(test.expected, diffInputs(test.previous, test.current), cmpInputChanges)
		})
	}
}

func TestInputKey(t *testing.T) {
	tests := []struct {
		description string
		workspace   string
		dep         string
		expected    string
	}{
		{
			description: "absolute dependency under an absolute workspace",
			workspace:   "/home/user/repo/app",
			dep:         "/home/user/repo/app/src/main.go",
			expected:    "src/main.go",
		},
		{
			description: "the same file in another checkout gives the same key",
			workspace:   "/home/user/other-checkout/app",
			dep:         "/home/user/other-checkout/app/src/main.go",
			expected:    "src/main.go",
		},
		{
			description: "dependency outside the workspace stays relative to it",
			workspace:   "/home/user/repo/app",
			dep:         "/home/user/repo/shared/lib.go",
			expected:    "../shared/lib.go",
		},
		{
			description: "relative workspace and dependency",
			workspace:   ".",
			dep:         "main.go",
			expected:    "main.go",
		},
		{
			description: "no workspace to relativise against",
			workspace:   "",
			dep:         "/home/user/repo/app/main.go",
			expected:    "/home/user/repo/app/main.go",
		},
		{
			description: "mixed absolute workspace and relative dependency is left alone",
			workspace:   "/home/user/repo/app",
			dep:         "main.go",
			expected:    "main.go",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.CheckDeepEqual(test.expected, inputKey(test.workspace, test.dep))
		})
	}
}

func TestInputChangesLines(t *testing.T) {
	testutil.Run(t, "modified, then added, then removed", func(t *testutil.T) {
		changes := inputChanges{
			modified: []string{"app.py"},
			added:    []string{"new.txt"},
			removed:  []string{"old.txt"},
		}
		t.CheckDeepEqual([]string{"~ app.py", "+ new.txt", "- old.txt"}, changes.lines())
	})

	testutil.Run(t, "the marker is stripped from non-file inputs", func(t *testutil.T) {
		changes := inputChanges{modified: []string{metaInputPrefix + "build args"}}
		t.CheckDeepEqual([]string{"~ build args"}, changes.lines())
	})
}

func TestInputRecorderReportsChangesAfterFirstRun(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		dir := t.NewTempDir().Path("inputs")
		ctx := context.Background()

		// First run: there is nothing to compare against, so no changes are reported.
		first := newInputRecorder(dir)
		first.record(ctx, "app", map[string]string{"app.py": "a", "keep.txt": "k"})
		if _, ok := first.changesFor("app"); ok {
			t.Errorf("expected no changes to be recorded on the first run")
		}

		// Second run: one input changed, one appeared, one went away.
		second := newInputRecorder(dir)
		second.record(ctx, "app", map[string]string{"app.py": "b", "new.txt": "n"})
		changes, ok := second.changesFor("app")
		if !ok {
			t.Fatalf("expected changes to be recorded on the second run")
		}
		t.CheckDeepEqual(inputChanges{
			modified: []string{"app.py"},
			added:    []string{"new.txt"},
			removed:  []string{"keep.txt"},
		}, changes, cmpInputChanges)

		// Third run: back to matching the second, so nothing is reported as changed.
		third := newInputRecorder(dir)
		third.record(ctx, "app", map[string]string{"app.py": "b", "new.txt": "n"})
		changes, ok = third.changesFor("app")
		if !ok {
			t.Fatalf("expected a snapshot to be compared on the third run")
		}
		if !changes.empty() {
			t.Errorf("expected no changes, got %v", changes.lines())
		}
	})
}

func TestInputRecorderDoesNotWriteValuesInTheClear(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		dir := t.NewTempDir().Path("inputs")

		const secret = "SUPER_SECRET_TOKEN_VALUE"
		recorder := newInputRecorder(dir)
		recorder.record(context.Background(), "app", map[string]string{
			metaInputPrefix + "build args": "TOKEN=" + secret,
		})

		contents, err := os.ReadFile(filepath.Join(dir, "app.txt"))
		t.CheckNoError(err)
		if strings.Contains(string(contents), secret) {
			t.Errorf("snapshot must not contain input values verbatim, got:\n%s", contents)
		}
		if !strings.Contains(string(contents), digestOf("TOKEN="+secret)) {
			t.Errorf("snapshot should record the digest of the value, got:\n%s", contents)
		}
	})
}

func TestInputRecorderSkipsKeysItCannotRoundTrip(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		dir := t.NewTempDir().Path("inputs")
		ctx := context.Background()

		newInputRecorder(dir).record(ctx, "app", map[string]string{
			"fine.txt":          "a",
			"with\ttab.txt":     "b",
			"with\nnewline.txt": "c",
		})

		snapshot, err := readSnapshot(filepath.Join(dir, "app.txt"))
		t.CheckNoError(err)
		t.CheckDeepEqual(1, len(snapshot))
		if _, ok := snapshot["fine.txt"]; !ok {
			t.Errorf("expected the well-formed key to be recorded, got %v", snapshot)
		}
	})
}

func TestNilInputRecorderIsANoOp(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		var recorder *inputRecorder
		t.CheckDeepEqual((*inputRecorder)(nil), newInputRecorder(""))

		// Neither call may panic.
		recorder.record(context.Background(), "app", map[string]string{"app.py": "a"})
		_, ok := recorder.changesFor("app")
		t.CheckDeepEqual(false, ok)
	})
}

func TestSnapshotPathReplacesUnsafeCharacters(t *testing.T) {
	tests := []struct {
		description string
		imageName   string
		expected    string
	}{
		{description: "plain name", imageName: "app", expected: "app.txt"},
		{description: "registry path", imageName: "gcr.io/project/app", expected: "gcr.io_project_app.txt"},
		{description: "tagged name", imageName: "app:v1", expected: "app_v1.txt"},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.CheckDeepEqual(filepath.Join("dir", test.expected), snapshotPath("dir", test.imageName))
		})
	}
}

func TestWriteSnapshotIsAtomic(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		dir := t.NewTempDir().Path("inputs")
		path := filepath.Join(dir, "app.txt")

		t.CheckNoError(writeSnapshot(path, map[string]string{"app.py": "a"}))
		t.CheckNoError(writeSnapshot(path, map[string]string{"app.py": "b"}))

		// No temporary files may be left behind.
		entries, err := os.ReadDir(dir)
		t.CheckNoError(err)
		t.CheckDeepEqual(1, len(entries))
		t.CheckDeepEqual("app.txt", entries[0].Name())

		snapshot, err := readSnapshot(path)
		t.CheckNoError(err)
		t.CheckDeepEqual(map[string]string{"app.py": "b"}, snapshot)
	})
}

func TestReadSnapshotSkipsMalformedLines(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		tmp := t.NewTempDir().Write("snapshot.txt", "app.py\tabc\ngarbage-with-no-tab\n\nother.txt\tdef\n")

		snapshot, err := readSnapshot(tmp.Path("snapshot.txt"))
		t.CheckNoError(err)
		t.CheckDeepEqual(map[string]string{"app.py": "abc", "other.txt": "def"}, snapshot)
	})
}
