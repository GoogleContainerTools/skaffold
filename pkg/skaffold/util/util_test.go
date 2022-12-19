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

package util

import (
	"path/filepath"
	"testing"

	"github.com/mitchellh/go-homedir"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestExpandPathsGlob(t *testing.T) {
	tests := []struct {
		description string
		in          []string
		out         []string
		shouldErr   bool
	}{
		{
			description: "match exact filename",
			in:          []string{"dir/sub_dir/file"},
			out:         []string{"dir/sub_dir/file"},
		},
		{
			description: "match leaf directory glob",
			in:          []string{"dir/sub_dir/*"},
			out:         []string{"dir/sub_dir/file"},
		},
		{
			description: "top level glob",
			in:          []string{"dir*"},
			out:         []string{"dir/sub_dir/file", "dir_b/sub_dir_b/file"},
		},
		{
			description: "invalid pattern",
			in:          []string{"[]"},
			shouldErr:   true,
		},
		{
			description: "keep top level order",
			in:          []string{"dir_b/*", "dir/*"},
			out:         []string{"dir_b/sub_dir_b/file", "dir/sub_dir/file"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir().
				Touch("dir/sub_dir/file", "dir_b/sub_dir_b/file")

			actual, err := ExpandPathsGlob(tmpDir.Root(), test.in)

			expected := tmpDir.Paths(test.out...)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, expected, actual)
		})
	}
}

func TestExpand(t *testing.T) {
	tests := []struct {
		description string
		text        string
		key         string
		value       string
		expected    string
	}{
		{
			description: "${key} syntax",
			text:        "BEFORE[${key}]AFTER",
			key:         "key",
			value:       "VALUE",
			expected:    "BEFORE[VALUE]AFTER",
		},
		{
			description: "$key syntax",
			text:        "BEFORE[$key]AFTER",
			key:         "key",
			value:       "VALUE",
			expected:    "BEFORE[VALUE]AFTER",
		},
		{
			description: "replace all",
			text:        "BEFORE[$key][${key}][$key][${key}]AFTER",
			key:         "key",
			value:       "VALUE",
			expected:    "BEFORE[VALUE][VALUE][VALUE][VALUE]AFTER",
		},
		{
			description: "ignore common prefix",
			text:        "BEFORE[$key1][${key1}]AFTER",
			key:         "key",
			value:       "VALUE",
			expected:    "BEFORE[$key1][${key1}]AFTER",
		},
		{
			description: "just the ${key} placeholder",
			text:        "${key}",
			key:         "key",
			value:       "VALUE",
			expected:    "VALUE",
		},
		{
			description: "just the $key placeholder",
			text:        "$key",
			key:         "key",
			value:       "VALUE",
			expected:    "VALUE",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			actual := Expand(test.text, test.key, test.value)

			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

func TestAbsFile(t *testing.T) {
	tmpDir := testutil.NewTempDir(t)
	tmpDir.Touch("file")

	expectedFile, err := filepath.Abs(filepath.Join(tmpDir.Root(), "file"))
	testutil.CheckError(t, false, err)

	file, err := AbsFile(tmpDir.Root(), "file")
	testutil.CheckErrorAndDeepEqual(t, false, err, expectedFile, file)

	_, err = AbsFile(tmpDir.Root(), "")
	testutil.CheckErrorAndDeepEqual(t, true, err, tmpDir.Root()+" is a directory", err.Error())

	_, err = AbsFile(tmpDir.Root(), "does-not-exist")
	testutil.CheckError(t, true, err)
}

func TestNonEmptyLines(t *testing.T) {
	tests := []struct {
		in  string
		out []string
	}{
		{"", nil},
		{"a\n", []string{"a"}},
		{"a\r\n", []string{"a"}},
		{"a\r\nb", []string{"a", "b"}},
		{"a\r\nb\n\n", []string{"a", "b"}},
		{"\na\r\n\n\n", []string{"a"}},
	}
	for _, test := range tests {
		testutil.Run(t, "", func(t *testutil.T) {
			result := NonEmptyLines([]byte(test.in))

			t.CheckDeepEqual(test.out, result)
		})
	}
}

func TestCloneThroughJSON(t *testing.T) {
	tests := []struct {
		description string
		old         interface{}
		new         interface{}
		expected    interface{}
	}{
		{
			description: "google cloud build",
			old: map[string]string{
				"projectId": "unit-test",
			},
			new: &latest.GoogleCloudBuild{},
			expected: &latest.GoogleCloudBuild{
				ProjectID: "unit-test",
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			CloneThroughJSON(test.old, test.new)

			t.CheckDeepEqual(test.expected, test.new)
		})
	}
}

func TestCloneThroughYAML(t *testing.T) {
	tests := []struct {
		description string
		old         interface{}
		new         interface{}
		expected    interface{}
	}{
		{
			description: "google cloud build",
			old: map[string]string{
				"projectId": "unit-test",
			},
			new: &latest.GoogleCloudBuild{},
			expected: &latest.GoogleCloudBuild{
				ProjectID: "unit-test",
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			CloneThroughYAML(test.old, test.new)

			t.CheckDeepEqual(test.expected, test.new)
		})
	}
}

func TestIsHiddenDir(t *testing.T) {
	tests := []struct {
		description string
		filename    string
		expected    bool
	}{
		{
			description: "hidden dir",
			filename:    ".hidden",
			expected:    true,
		},
		{
			description: "not hidden dir",
			filename:    "not_hidden",
			expected:    false,
		},
		{
			description: "current dir",
			filename:    ".",
			expected:    false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			isHidden := IsHiddenDir(test.filename)

			t.CheckDeepEqual(test.expected, isHidden)
		})
	}
}

func TestIsHiddenFile(t *testing.T) {
	tests := []struct {
		description string
		filename    string
		expected    bool
	}{
		{
			description: "hidden file name",
			filename:    ".hidden",
			expected:    true,
		},
		{
			description: "not hidden file",
			filename:    "not_hidden",
			expected:    false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			isHidden := IsHiddenDir(test.filename)

			t.CheckDeepEqual(test.expected, isHidden)
		})
	}
}

func TestEnvMapToSlice(t *testing.T) {
	tests := []struct {
		description string
		args        map[string]string
		expected    []string
	}{
		{
			description: "regular key:value",
			args: map[string]string{
				"one": "1",
				"two": "2",
			},
			expected: []string{"one=1", "two=2"},
		}, {
			description: "empty key:value",
			args: map[string]string{
				"one": "",
				"two": "",
			},
			expected: []string{"one=", "two="},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			actual := EnvMapToSlice(test.args, "=")

			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

func TestMapPtrToSlice(t *testing.T) {
	tests := []struct {
		description string
		args        map[string]*string
		expected    []string
	}{
		{
			description: "regular key:value",
			args: map[string]*string{
				"one": Ptr("1"),
				"two": Ptr("2"),
			},
			expected: []string{"one=1", "two=2"},
		}, {
			description: "empty key:value",
			args: map[string]*string{
				"one": Ptr(""),
				"two": Ptr(""),
			},
			expected: []string{"one=", "two="},
		}, {
			description: "nil value",
			args: map[string]*string{
				"one": nil,
				"two": nil,
			},
			expected: []string{"one", "two"},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			actual := EnvPtrMapToSlice(test.args, "=")

			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

func TestEnvSliceToMap(t *testing.T) {
	tests := []struct {
		description string
		args        []string
		expected    map[string]string
	}{
		{
			description: "regular key=value",
			args:        []string{"one=1", "two=2"},
			expected:    map[string]string{"one": "1", "two": "2"},
		},
		{
			description: "empty key=",
			args:        []string{"one=", "two="},
			expected:    map[string]string{"one": "", "two": ""},
		},
		{
			description: "last repeated key wins",
			args:        []string{"one=a", "one=b"},
			expected:    map[string]string{"one": "b"},
		},
		{
			description: "elements missing separator is dropped",
			args:        []string{"one", "two=2"},
			expected:    map[string]string{"two": "2"},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			actual := EnvSliceToMap(test.args, "=")

			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

func TestIsSubPath(t *testing.T) {
	home, _ := homedir.Dir()
	tests := []struct {
		description string
		basePath    string
		targetPath  string
		expected    bool
	}{
		{
			description: "target path within base path",
			basePath:    filepath.Join(home, ".minikube"),
			targetPath:  filepath.Join(home, ".minikube", "ca.crt"),
			expected:    true,
		},
		{
			description: "target path outside base path",
			basePath:    filepath.Join(home, "bar"),
			targetPath:  filepath.Join(home, "foo", "bar"),
			expected:    false,
		},
		{
			description: "base path inside target path",
			basePath:    filepath.Join(home, "foo", "bar"),
			targetPath:  filepath.Join(home, "foo"),
			expected:    false,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.CheckDeepEqual(test.expected, IsSubPath(test.basePath, test.targetPath))
		})
	}
}

func TestIsFileIsDir(t *testing.T) {
	tmpDir := testutil.NewTempDir(t).Touch("file")

	testutil.CheckDeepEqual(t, false, IsFile(tmpDir.Root()))
	testutil.CheckDeepEqual(t, true, IsDir(tmpDir.Root()))

	testutil.CheckDeepEqual(t, true, IsFile(filepath.Join(tmpDir.Root(), "file")))
	testutil.CheckDeepEqual(t, false, IsDir(filepath.Join(tmpDir.Root(), "file")))

	testutil.CheckDeepEqual(t, false, IsFile(filepath.Join(tmpDir.Root(), "nonexistent")))
	testutil.CheckDeepEqual(t, false, IsDir(filepath.Join(tmpDir.Root(), "nonexistent")))
}

func TestIsURL(t *testing.T) {
	testutil.CheckDeepEqual(t, false, IsURL("foo"))
	testutil.CheckDeepEqual(t, false, IsURL("http:bar"))
	testutil.CheckDeepEqual(t, false, IsURL("https:bar"))

	testutil.CheckDeepEqual(t, true, IsURL("http://bar"))
	testutil.CheckDeepEqual(t, true, IsURL("https://bar"))
}

func TestIsEmptyDir(t *testing.T) {
	tmpDir := testutil.NewTempDir(t).Touch("file")
	emptyTmpDir := testutil.NewTempDir(t)

	testutil.CheckDeepEqual(t, true, IsEmptyDir(emptyTmpDir.Root()))
	testutil.CheckDeepEqual(t, false, IsEmptyDir(tmpDir.Root()))
	testutil.CheckDeepEqual(t, false, IsEmptyDir(filepath.Join(tmpDir.Root(), "file")))
}
