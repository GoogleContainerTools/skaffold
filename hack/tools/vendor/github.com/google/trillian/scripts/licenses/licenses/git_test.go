// Copyright 2019 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package licenses

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	git "gopkg.in/src-d/go-git.v4"
)

func TestGitFileURL(t *testing.T) {
	t.Parallel()

	dir, err := ioutil.TempDir("", "git_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	cloneOpts := git.CloneOptions{
		URL:   "https://github.com/google/trillian",
		Depth: 1,
	}
	if _, err := git.PlainClone(dir, false, &cloneOpts); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		desc    string
		file    string
		remote  string
		wantURL string
		wantErr error
	}{
		{
			desc:    "License URL",
			file:    filepath.Join(dir, "LICENSE"),
			remote:  "origin",
			wantURL: "https://github.com/google/trillian/blob/master/LICENSE",
		},
		{
			desc:    "Non-existent remote",
			file:    filepath.Join(dir, "LICENSE"),
			remote:  "foo",
			wantErr: git.ErrRemoteNotFound,
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			url, err := GitFileURL(test.file, test.remote)
			if err != nil {
				if err != test.wantErr {
					t.Fatalf("GitFileURL(%q, %q) = (_, %q), want (_, %q)", test.file, test.remote, err, test.wantErr)
				}
				return
			}
			if url.String() != test.wantURL {
				t.Fatalf("GitFileURL(%q, %q) = (%q, nil), want (%q, nil)", test.file, test.remote, url, test.wantURL)
			}
		})
	}
}
