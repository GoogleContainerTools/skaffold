/*
Copyright 2018 Google LLC

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

package tag

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/skaffold/testutil"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

func TestGitCommit_GenerateFullyQualifiedImageName(t *testing.T) {
	tests := []struct {
		description   string
		expectedName  string
		createGitRepo func(string)
		opts          *TagOptions
		shouldErr     bool
	}{
		{
			description: "success",
			opts: &TagOptions{
				ImageName: "test",
				Digest:    "sha256:12345abcde",
			},
			expectedName: "test:41cf71e",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					write(t, "source.go", []byte("code")).
					add(t, "source.go").
					commit(t, "initial")
			},
		},
		{
			description: "dirty",
			opts: &TagOptions{
				ImageName: "test",
			},
			expectedName: "test:41cf71e-dirty-17c3e6fb2811b7af",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					write(t, "source.go", []byte("code")).
					add(t, "source.go").
					commit(t, "initial").
					write(t, "source.go", []byte("updated code"))
			},
		},
		{
			description: "untracked",
			opts: &TagOptions{
				ImageName: "test",
			},
			expectedName: "test:41cf71e-dirty-d7bc32e5f6760a99",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					write(t, "source.go", []byte("code")).
					add(t, "source.go").
					commit(t, "initial").
					write(t, "new.go", []byte("new code"))
			},
		},
		{
			description:   "failure",
			createGitRepo: func(dir string) {},
			shouldErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			tmpDir, cleanup := testutil.TempDir(t)
			defer cleanup()

			tt.createGitRepo(tmpDir)

			c := &GitCommit{}
			name, err := c.GenerateFullyQualifiedImageName(tmpDir, tt.opts)

			testutil.CheckErrorAndDeepEqual(t, tt.shouldErr, err, tt.expectedName, name)
		})
	}
}

func TestGenerateFullyQualifiedImageNameFromSubDirectory(t *testing.T) {
	tmpDir, cleanup := testutil.TempDir(t)
	defer cleanup()

	gitInit(t, tmpDir).
		mkdir("sub/sub").
		commit(t, "initial")

	opts := &TagOptions{ImageName: "test"}
	c := &GitCommit{}

	name1, err := c.GenerateFullyQualifiedImageName(tmpDir, opts)
	failNowIfError(t, err)

	subDir := filepath.Join(tmpDir, "sub", "sub")
	name2, err := c.GenerateFullyQualifiedImageName(subDir, opts)
	failNowIfError(t, err)

	if name1 != name2 || name1 != "test:6e0beab" {
		t.Errorf("Invalid names found: %s and %s", name1, name2)
	}
}

// gitRepo deals with test git repositories
type gitRepo struct {
	dir      string
	workTree *git.Worktree
}

func gitInit(t *testing.T, dir string) *gitRepo {
	repo, err := git.PlainInit(dir, false)
	failNowIfError(t, err)

	w, err := repo.Worktree()
	failNowIfError(t, err)

	return &gitRepo{
		dir:      dir,
		workTree: w,
	}
}

func (g *gitRepo) mkdir(folder string) *gitRepo {
	os.MkdirAll(filepath.Join(g.dir, folder), os.ModePerm)
	return g
}

func (g *gitRepo) write(t *testing.T, file string, content []byte) *gitRepo {
	err := ioutil.WriteFile(filepath.Join(g.dir, file), content, 0644)
	failNowIfError(t, err)
	return g
}

func (g *gitRepo) add(t *testing.T, file string) *gitRepo {
	_, err := g.workTree.Add(file)
	failNowIfError(t, err)
	return g
}

func (g *gitRepo) commit(t *testing.T, msg string) *gitRepo {
	now, err := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Feb 3, 2013 at 7:54pm (PST)")
	failNowIfError(t, err)

	_, err = g.workTree.Commit(msg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "John Doe",
			Email: "john@doe.org",
			When:  now,
		},
	})
	failNowIfError(t, err)

	return g
}

func failNowIfError(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}
