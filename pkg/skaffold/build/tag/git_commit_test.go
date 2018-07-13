// +build !windows

/*
Copyright 2018 The Skaffold Authors

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

	"github.com/GoogleContainerTools/skaffold/testutil"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

// These tests do not run on windows
// See: https://github.com/src-d/go-git/issues/378
func TestGitCommit_GenerateFullyQualifiedImageName(t *testing.T) {
	tests := []struct {
		description   string
		expectedName  string
		createGitRepo func(string)
		subDir        string
		shouldErr     bool
	}{
		{
			description:  "success",
			expectedName: "test:eefe1b9",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					write("source.go", []byte("code")).
					add("source.go").
					commit("initial")
			},
		},
		{
			description:  "use tag over commit",
			expectedName: "test:v2",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					write("source.go", []byte("code")).
					add("source.go").
					commit("initial").
					tag("v1").
					write("other.go", []byte("other")).
					add("other.go").
					commit("second commit").
					tag("v2")
			},
		},
		{
			description:  "dirty",
			expectedName: "test:eefe1b9-dirty-abababa",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					write("source.go", []byte("code")).
					add("source.go").
					commit("initial").
					write("source.go", []byte("updated code"))
			},
		},
		{
			description:  "ignore tag when dirty",
			expectedName: "test:eefe1b9-dirty-abababa",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					write("source.go", []byte("code")).
					add("source.go").
					commit("initial").
					tag("v1").
					write("source.go", []byte("updated code"))
			},
		},
		{
			description:  "untracked",
			expectedName: "test:eefe1b9-dirty-abababa",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					write("source.go", []byte("code")).
					add("source.go").
					commit("initial").
					write("new.go", []byte("new code"))
			},
		},
		{
			description:  "don't use tag if not exact match",
			expectedName: "test:3cec6b9",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					write("source.go", []byte("code")).
					add("source.go").
					commit("initial").
					tag("v1").
					write("source.go", []byte("updated code")).
					add("source.go").
					commit("changes")
			},
		},
		{
			description:  "deleted file",
			expectedName: "test:279d53f-dirty-abababa",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					write("source1.go", []byte("code1")).
					write("source2.go", []byte("code2")).
					add("source1.go", "source2.go").
					commit("initial").
					delete("source1.go")
			},
		},
		{
			description:  "rename",
			expectedName: "test:eefe1b9-dirty-abababa",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					write("source.go", []byte("code")).
					add("source.go").
					commit("initial").
					rename("source.go", "source2.go")
			},
		},
		{
			description:  "sub directory",
			expectedName: "test:a7b32a6",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					mkdir("sub/sub").
					commit("initial")
			},
			subDir: "sub/sub",
		},
		{
			description:  "clean artifact1 in tagged repo",
			expectedName: "test:v1",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					mkdir("artifact1").write("artifact1/source.go", []byte("code")).
					mkdir("artifact2").write("artifact2/source.go", []byte("code")).
					add("artifact1/source.go", "artifact2/source.go").
					commit("initial").tag("v1")
			},
			subDir: "artifact1",
		},
		{
			description:  "clean artifact2 in tagged repo",
			expectedName: "test:v1",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					mkdir("artifact1").write("artifact1/source.go", []byte("code")).
					mkdir("artifact2").write("artifact2/source.go", []byte("code")).
					add("artifact1/source.go", "artifact2/source.go").
					commit("initial").tag("v1")
			},
			subDir: "artifact2",
		},
		{
			description:  "clean artifact in dirty repo",
			expectedName: "test:v1",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					mkdir("artifact1").write("artifact1/source.go", []byte("code")).
					mkdir("artifact2").write("artifact2/source.go", []byte("code")).
					add("artifact1/source.go", "artifact2/source.go").
					commit("initial").tag("v1").
					write("artifact2/source.go", []byte("updated code"))
			},
			subDir: "artifact1",
		},
		{
			description:  "updated artifact in dirty repo",
			expectedName: "test:0c60cb8-dirty-abababa",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					mkdir("artifact1").write("artifact1/source.go", []byte("code")).
					mkdir("artifact2").write("artifact2/source.go", []byte("code")).
					add("artifact1/source.go", "artifact2/source.go").
					commit("initial").tag("v1").
					write("artifact2/source.go", []byte("updated code"))
			},
			subDir: "artifact2",
		},
		{
			description:  "non git repo",
			expectedName: "test:dirty-abababa",
			createGitRepo: func(dir string) {
				ioutil.WriteFile(filepath.Join(dir, "source.go"), []byte("code"), os.ModePerm)
			},
		},
		{
			description:  "git repo with no commit",
			expectedName: "test:dirty-abababa",
			createGitRepo: func(dir string) {
				gitInit(t, dir)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			tmpDir, cleanup := testutil.TempDir(t)
			defer cleanup()

			tt.createGitRepo(tmpDir)
			workspace := filepath.Join(tmpDir, tt.subDir)

			opts := &Options{
				ImageName: "test",
				Digest:    "sha256:ababababababababababa",
			}

			c := &GitCommit{}
			name, err := c.GenerateFullyQualifiedImageName(workspace, opts)
			testutil.CheckErrorAndDeepEqual(t, tt.shouldErr, err, tt.expectedName, name)

			name, err = generateNameGoGit(workspace, opts)
			testutil.CheckErrorAndDeepEqual(t, tt.shouldErr, err, tt.expectedName, name)

			name, err = generateNameGitShellOut(workspace, opts)
			testutil.CheckErrorAndDeepEqual(t, tt.shouldErr, err, tt.expectedName, name)
		})
	}
}

// gitRepo deals with test git repositories
type gitRepo struct {
	dir      string
	repo     *git.Repository
	workTree *git.Worktree
	t        *testing.T
}

func gitInit(t *testing.T, dir string) *gitRepo {
	repo, err := git.PlainInit(dir, false)
	failNowIfError(t, err)

	w, err := repo.Worktree()
	failNowIfError(t, err)

	return &gitRepo{
		dir:      dir,
		repo:     repo,
		workTree: w,
		t:        t,
	}
}

func (g *gitRepo) mkdir(folder string) *gitRepo {
	err := os.MkdirAll(filepath.Join(g.dir, folder), os.ModePerm)
	failNowIfError(g.t, err)
	return g
}

func (g *gitRepo) write(file string, content []byte) *gitRepo {
	err := ioutil.WriteFile(filepath.Join(g.dir, file), content, os.ModePerm)
	failNowIfError(g.t, err)
	return g
}

func (g *gitRepo) rename(file, to string) *gitRepo {
	err := os.Rename(filepath.Join(g.dir, file), filepath.Join(g.dir, to))
	failNowIfError(g.t, err)
	return g
}

func (g *gitRepo) delete(files ...string) *gitRepo {
	for _, file := range files {
		err := os.Remove(filepath.Join(g.dir, file))
		failNowIfError(g.t, err)
	}
	return g
}

func (g *gitRepo) add(files ...string) *gitRepo {
	for _, file := range files {
		_, err := g.workTree.Add(file)
		failNowIfError(g.t, err)
	}
	return g
}

func (g *gitRepo) commit(msg string) *gitRepo {
	now, err := time.Parse("Jan 2, 2006 at 15:04:05 -0700 MST", "Feb 3, 2013 at 19:54:00 -0700 MST")
	failNowIfError(g.t, err)

	_, err = g.workTree.Commit(msg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "John Doe",
			Email: "john@doe.org",
			When:  now,
		},
	})
	failNowIfError(g.t, err)

	return g
}

func (g *gitRepo) tag(tag string) *gitRepo {
	head, err := g.repo.Head()
	failNowIfError(g.t, err)

	n := plumbing.ReferenceName("refs/tags/" + tag)
	t := plumbing.NewHashReference(n, head.Hash())

	err = g.repo.Storer.SetReference(t)
	failNowIfError(g.t, err)

	return g
}

func failNowIfError(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}
