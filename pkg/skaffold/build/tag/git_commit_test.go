// +build !windows

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
		description            string
		variantTags            string
		variantCommitSha       string
		variantAbbrevCommitSha string
		variantTreeSha         string
		variantAbbrevTreeSha   string
		createGitRepo          func(string)
		subDir                 string
		shouldErr              bool
	}{
		{
			description:            "clean worktree without tag",
			variantTags:            "test:eefe1b9",
			variantCommitSha:       "test:eefe1b9c44eb0aa87199c9a079f2d48d8eb8baed",
			variantAbbrevCommitSha: "test:eefe1b9",
			variantTreeSha:         "test:3bed02ca656e336307e4eb4d80080d7221cba62c",
			variantAbbrevTreeSha:   "test:3bed02c",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					write("source.go", []byte("code")).
					add("source.go").
					commit("initial")
			},
		},
		{
			description:            "clean worktree with tags",
			variantTags:            "test:v2",
			variantCommitSha:       "test:aea33bcc86b5af8c8570ff45d8a643202d63c808",
			variantAbbrevCommitSha: "test:aea33bc",
			variantTreeSha:         "test:bc69d50cda6897a6f2054e64b9059f038dc6fb0e",
			variantAbbrevTreeSha:   "test:bc69d50",
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
			description:            "treeSha only considers current tree content",
			variantTags:            "test:v1",
			variantCommitSha:       "test:b2f7a7d62794237ac293eb07c6bcae3736b96231",
			variantAbbrevCommitSha: "test:b2f7a7d",
			variantTreeSha:         "test:3bed02ca656e336307e4eb4d80080d7221cba62c",
			variantAbbrevTreeSha:   "test:3bed02c",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					write("source.go", []byte("other code")).
					add("source.go").
					commit("initial").
					write("source.go", []byte("code")).
					add("source.go").
					commit("updated code").
					tag("v1")
			},
		},
		{
			description:            "dirty worktree without tag",
			variantTags:            "test:eefe1b9-dirty",
			variantCommitSha:       "test:eefe1b9c44eb0aa87199c9a079f2d48d8eb8baed-dirty",
			variantAbbrevCommitSha: "test:eefe1b9-dirty",
			variantTreeSha:         "test:3bed02ca656e336307e4eb4d80080d7221cba62c-dirty",
			variantAbbrevTreeSha:   "test:3bed02c-dirty",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					write("source.go", []byte("code")).
					add("source.go").
					commit("initial").
					write("source.go", []byte("updated code"))
			},
		},
		{
			description:            "dirty worktree with tag",
			variantTags:            "test:v1-dirty",
			variantCommitSha:       "test:eefe1b9c44eb0aa87199c9a079f2d48d8eb8baed-dirty",
			variantAbbrevCommitSha: "test:eefe1b9-dirty",
			variantTreeSha:         "test:3bed02ca656e336307e4eb4d80080d7221cba62c-dirty",
			variantAbbrevTreeSha:   "test:3bed02c-dirty",
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
			description:            "untracked",
			variantTags:            "test:eefe1b9-dirty",
			variantCommitSha:       "test:eefe1b9c44eb0aa87199c9a079f2d48d8eb8baed-dirty",
			variantAbbrevCommitSha: "test:eefe1b9-dirty",
			variantTreeSha:         "test:3bed02ca656e336307e4eb4d80080d7221cba62c-dirty",
			variantAbbrevTreeSha:   "test:3bed02c-dirty",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					write("source.go", []byte("code")).
					add("source.go").
					commit("initial").
					write("new.go", []byte("new code"))
			},
		},
		{
			description:            "tag plus one commit",
			variantTags:            "test:v1-1-g3cec6b9",
			variantCommitSha:       "test:3cec6b950895704a8a69b610a199b242a3bd370f",
			variantAbbrevCommitSha: "test:3cec6b9",
			variantTreeSha:         "test:81eea360f7f81bc5c187498a8d6c4337e0361374",
			variantAbbrevTreeSha:   "test:81eea36",
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
			description:            "deleted file",
			variantTags:            "test:279d53f-dirty",
			variantCommitSha:       "test:279d53fcc3ae34503aec382a49a41f6db6de9a66-dirty",
			variantAbbrevCommitSha: "test:279d53f-dirty",
			variantTreeSha:         "test:039c20a072ceb72fb72d5883315df91659bb8ae4-dirty",
			variantAbbrevTreeSha:   "test:039c20a-dirty",
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
			description:            "rename",
			variantTags:            "test:eefe1b9-dirty",
			variantCommitSha:       "test:eefe1b9c44eb0aa87199c9a079f2d48d8eb8baed-dirty",
			variantAbbrevCommitSha: "test:eefe1b9-dirty",
			variantTreeSha:         "test:3bed02ca656e336307e4eb4d80080d7221cba62c-dirty",
			variantAbbrevTreeSha:   "test:3bed02c-dirty",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					write("source.go", []byte("code")).
					add("source.go").
					commit("initial").
					rename("source.go", "source2.go")
			},
		},
		{
			description:            "sub directory",
			variantTags:            "test:a7b32a6",
			variantCommitSha:       "test:a7b32a69335a6daa51bd89cc1bf30bd31df228ba",
			variantAbbrevCommitSha: "test:a7b32a6",
			variantTreeSha:         "test:dirty",
			variantAbbrevTreeSha:   "test:dirty",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					mkdir("sub/sub").
					commit("initial")
			},
			subDir: "sub/sub",
		},
		{
			description:            "clean artifact1 in tagged repo",
			variantTags:            "test:v1",
			variantCommitSha:       "test:b610928dc27484cc56990bc77622aab0dbd67131",
			variantAbbrevCommitSha: "test:b610928",
			variantTreeSha:         "test:3bed02ca656e336307e4eb4d80080d7221cba62c",
			variantAbbrevTreeSha:   "test:3bed02c",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					mkdir("artifact1").write("artifact1/source.go", []byte("code")).
					mkdir("artifact2").write("artifact2/source.go", []byte("other code")).
					add("artifact1/source.go", "artifact2/source.go").
					commit("initial").tag("v1")
			},
			subDir: "artifact1/",
		},
		{
			description:            "clean artifact2 in tagged repo",
			variantTags:            "test:v1",
			variantCommitSha:       "test:b610928dc27484cc56990bc77622aab0dbd67131",
			variantAbbrevCommitSha: "test:b610928",
			variantTreeSha:         "test:36651c832d8bf5ca1e84c6dc23bb8678fa51cf3e",
			variantAbbrevTreeSha:   "test:36651c8",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					mkdir("artifact1").write("artifact1/source.go", []byte("code")).
					mkdir("artifact2").write("artifact2/source.go", []byte("other code")).
					add("artifact1/source.go", "artifact2/source.go").
					commit("initial").tag("v1")
			},
			subDir: "artifact2",
		},
		{
			description:            "clean artifact in dirty repo",
			variantTags:            "test:v1",
			variantCommitSha:       "test:b610928dc27484cc56990bc77622aab0dbd67131",
			variantAbbrevCommitSha: "test:b610928",
			variantTreeSha:         "test:3bed02ca656e336307e4eb4d80080d7221cba62c",
			variantAbbrevTreeSha:   "test:3bed02c",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					mkdir("artifact1").write("artifact1/source.go", []byte("code")).
					mkdir("artifact2").write("artifact2/source.go", []byte("other code")).
					add("artifact1/source.go", "artifact2/source.go").
					commit("initial").tag("v1").
					write("artifact2/source.go", []byte("updated code"))
			},
			subDir: "artifact1",
		},
		{
			description:            "updated artifact in dirty repo",
			variantTags:            "test:v1-dirty",
			variantCommitSha:       "test:b610928dc27484cc56990bc77622aab0dbd67131-dirty",
			variantAbbrevCommitSha: "test:b610928-dirty",
			variantTreeSha:         "test:36651c832d8bf5ca1e84c6dc23bb8678fa51cf3e-dirty",
			variantAbbrevTreeSha:   "test:36651c8-dirty",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					mkdir("artifact1").write("artifact1/source.go", []byte("code")).
					mkdir("artifact2").write("artifact2/source.go", []byte("other code")).
					add("artifact1/source.go", "artifact2/source.go").
					commit("initial").tag("v1").
					write("artifact2/source.go", []byte("updated code"))
			},
			subDir: "artifact2",
		},
		{
			description:            "additional commit in other artifact",
			variantTags:            "test:0d16f59",
			variantCommitSha:       "test:0d16f59900bd63dd39425d6085d3f1333b66804f",
			variantAbbrevCommitSha: "test:0d16f59",
			variantTreeSha:         "test:3bed02ca656e336307e4eb4d80080d7221cba62c",
			variantAbbrevTreeSha:   "test:3bed02c",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					mkdir("artifact1").write("artifact1/source.go", []byte("code")).
					mkdir("artifact2").write("artifact2/source.go", []byte("other code")).
					add("artifact1/source.go", "artifact2/source.go").
					commit("initial").
					write("artifact2/source.go", []byte("updated code")).
					add("artifact2/source.go").
					commit("update artifact2")
			},
			subDir: "artifact1",
		},
		{
			description:            "non git repo",
			variantTags:            "test:dirty",
			variantCommitSha:       "test:dirty",
			variantAbbrevCommitSha: "test:dirty",
			variantTreeSha:         "test:dirty",
			variantAbbrevTreeSha:   "test:dirty",
			createGitRepo: func(dir string) {
				ioutil.WriteFile(filepath.Join(dir, "source.go"), []byte("code"), os.ModePerm)
			},
		},
		{
			description:            "git repo with no commit",
			variantTags:            "test:dirty",
			variantCommitSha:       "test:dirty",
			variantAbbrevCommitSha: "test:dirty",
			variantTreeSha:         "test:dirty",
			variantAbbrevTreeSha:   "test:dirty",
			createGitRepo: func(dir string) {
				gitInit(t, dir)
			},
		},
	}

	tTags, err := NewGitCommit("Tags")
	testutil.CheckError(t, false, err)

	tCommit, err := NewGitCommit("CommitSha")
	testutil.CheckError(t, false, err)

	tAbbrevC, err := NewGitCommit("AbbrevCommitSha")
	testutil.CheckError(t, false, err)

	tTree, err := NewGitCommit("TreeSha")
	testutil.CheckError(t, false, err)

	tAbbrevT, err := NewGitCommit("AbbrevTreeSha")
	testutil.CheckError(t, false, err)

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir()
			test.createGitRepo(tmpDir.Root())
			workspace := tmpDir.Path(test.subDir)

			name, err := tTags.GenerateFullyQualifiedImageName(workspace, "test")
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.variantTags, name)

			name, err = tCommit.GenerateFullyQualifiedImageName(workspace, "test")
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.variantCommitSha, name)

			name, err = tAbbrevC.GenerateFullyQualifiedImageName(workspace, "test")
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.variantAbbrevCommitSha, name)

			name, err = tTree.GenerateFullyQualifiedImageName(workspace, "test")
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.variantTreeSha, name)

			name, err = tAbbrevT.GenerateFullyQualifiedImageName(workspace, "test")
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.variantAbbrevTreeSha, name)
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
