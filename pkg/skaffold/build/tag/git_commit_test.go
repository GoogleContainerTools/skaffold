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
	"strings"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

// These tests do not run on windows
// See: https://github.com/src-d/go-git/issues/378
func TestGitCommit_GenerateTag(t *testing.T) {
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
			variantTags:            "eefe1b9",
			variantCommitSha:       "eefe1b9c44eb0aa87199c9a079f2d48d8eb8baed",
			variantAbbrevCommitSha: "eefe1b9",
			variantTreeSha:         "3bed02ca656e336307e4eb4d80080d7221cba62c",
			variantAbbrevTreeSha:   "3bed02c",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					write("source.go", "code").
					add("source.go").
					commit("initial")
			},
		},
		{
			description:            "clean worktree with tag containing a slash",
			variantTags:            "v_2",
			variantCommitSha:       "aea33bcc86b5af8c8570ff45d8a643202d63c808",
			variantAbbrevCommitSha: "aea33bc",
			variantTreeSha:         "bc69d50cda6897a6f2054e64b9059f038dc6fb0e",
			variantAbbrevTreeSha:   "bc69d50",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					write("source.go", "code").
					add("source.go").
					commit("initial").
					tag("v/1").
					write("other.go", "other").
					add("other.go").
					commit("second commit").
					tag("v/2")
			},
		},
		{
			description:            "clean worktree with tags",
			variantTags:            "v2",
			variantCommitSha:       "aea33bcc86b5af8c8570ff45d8a643202d63c808",
			variantAbbrevCommitSha: "aea33bc",
			variantTreeSha:         "bc69d50cda6897a6f2054e64b9059f038dc6fb0e",
			variantAbbrevTreeSha:   "bc69d50",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					write("source.go", "code").
					add("source.go").
					commit("initial").
					tag("v1").
					write("other.go", "other").
					add("other.go").
					commit("second commit").
					tag("v2")
			},
		},
		{
			description:            "treeSha only considers current tree content",
			variantTags:            "v1",
			variantCommitSha:       "b2f7a7d62794237ac293eb07c6bcae3736b96231",
			variantAbbrevCommitSha: "b2f7a7d",
			variantTreeSha:         "3bed02ca656e336307e4eb4d80080d7221cba62c",
			variantAbbrevTreeSha:   "3bed02c",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					write("source.go", "other code").
					add("source.go").
					commit("initial").
					write("source.go", "code").
					add("source.go").
					commit("updated code").
					tag("v1")
			},
		},
		{
			description:            "dirty worktree without tag",
			variantTags:            "eefe1b9-dirty",
			variantCommitSha:       "eefe1b9c44eb0aa87199c9a079f2d48d8eb8baed-dirty",
			variantAbbrevCommitSha: "eefe1b9-dirty",
			variantTreeSha:         "3bed02ca656e336307e4eb4d80080d7221cba62c-dirty",
			variantAbbrevTreeSha:   "3bed02c-dirty",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					write("source.go", "code").
					add("source.go").
					commit("initial").
					write("source.go", "updated code")
			},
		},
		{
			description:            "dirty worktree with tag",
			variantTags:            "v1-dirty",
			variantCommitSha:       "eefe1b9c44eb0aa87199c9a079f2d48d8eb8baed-dirty",
			variantAbbrevCommitSha: "eefe1b9-dirty",
			variantTreeSha:         "3bed02ca656e336307e4eb4d80080d7221cba62c-dirty",
			variantAbbrevTreeSha:   "3bed02c-dirty",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					write("source.go", "code").
					add("source.go").
					commit("initial").
					tag("v1").
					write("source.go", "updated code")
			},
		},
		{
			description:            "untracked",
			variantTags:            "eefe1b9-dirty",
			variantCommitSha:       "eefe1b9c44eb0aa87199c9a079f2d48d8eb8baed-dirty",
			variantAbbrevCommitSha: "eefe1b9-dirty",
			variantTreeSha:         "3bed02ca656e336307e4eb4d80080d7221cba62c-dirty",
			variantAbbrevTreeSha:   "3bed02c-dirty",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					write("source.go", "code").
					add("source.go").
					commit("initial").
					write("new.go", "new code")
			},
		},
		{
			description:            "tag plus one commit",
			variantTags:            "v1-1-g3cec6b9",
			variantCommitSha:       "3cec6b950895704a8a69b610a199b242a3bd370f",
			variantAbbrevCommitSha: "3cec6b9",
			variantTreeSha:         "81eea360f7f81bc5c187498a8d6c4337e0361374",
			variantAbbrevTreeSha:   "81eea36",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					write("source.go", "code").
					add("source.go").
					commit("initial").
					tag("v1").
					write("source.go", "updated code").
					add("source.go").
					commit("changes")
			},
		},
		{
			description:            "deleted file",
			variantTags:            "279d53f-dirty",
			variantCommitSha:       "279d53fcc3ae34503aec382a49a41f6db6de9a66-dirty",
			variantAbbrevCommitSha: "279d53f-dirty",
			variantTreeSha:         "039c20a072ceb72fb72d5883315df91659bb8ae4-dirty",
			variantAbbrevTreeSha:   "039c20a-dirty",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					write("source1.go", "code1").
					write("source2.go", "code2").
					add("source1.go", "source2.go").
					commit("initial").
					delete("source1.go")
			},
		},
		{
			description:            "rename",
			variantTags:            "eefe1b9-dirty",
			variantCommitSha:       "eefe1b9c44eb0aa87199c9a079f2d48d8eb8baed-dirty",
			variantAbbrevCommitSha: "eefe1b9-dirty",
			variantTreeSha:         "3bed02ca656e336307e4eb4d80080d7221cba62c-dirty",
			variantAbbrevTreeSha:   "3bed02c-dirty",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					write("source.go", "code").
					add("source.go").
					commit("initial").
					rename("source.go", "source2.go")
			},
		},
		{
			description:            "clean artifact1 in tagged repo",
			variantTags:            "v1",
			variantCommitSha:       "b610928dc27484cc56990bc77622aab0dbd67131",
			variantAbbrevCommitSha: "b610928",
			variantTreeSha:         "3bed02ca656e336307e4eb4d80080d7221cba62c",
			variantAbbrevTreeSha:   "3bed02c",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					mkdir("artifact1").write("artifact1/source.go", "code").
					mkdir("artifact2").write("artifact2/source.go", "other code").
					add("artifact1/source.go", "artifact2/source.go").
					commit("initial").tag("v1")
			},
			subDir: "artifact1/",
		},
		{
			description:            "clean artifact2 in tagged repo",
			variantTags:            "v1",
			variantCommitSha:       "b610928dc27484cc56990bc77622aab0dbd67131",
			variantAbbrevCommitSha: "b610928",
			variantTreeSha:         "36651c832d8bf5ca1e84c6dc23bb8678fa51cf3e",
			variantAbbrevTreeSha:   "36651c8",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					mkdir("artifact1").write("artifact1/source.go", "code").
					mkdir("artifact2").write("artifact2/source.go", "other code").
					add("artifact1/source.go", "artifact2/source.go").
					commit("initial").tag("v1")
			},
			subDir: "artifact2",
		},
		{
			description:            "clean artifact in dirty repo",
			variantTags:            "v1",
			variantCommitSha:       "b610928dc27484cc56990bc77622aab0dbd67131",
			variantAbbrevCommitSha: "b610928",
			variantTreeSha:         "3bed02ca656e336307e4eb4d80080d7221cba62c",
			variantAbbrevTreeSha:   "3bed02c",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					mkdir("artifact1").write("artifact1/source.go", "code").
					mkdir("artifact2").write("artifact2/source.go", "other code").
					add("artifact1/source.go", "artifact2/source.go").
					commit("initial").tag("v1").
					write("artifact2/source.go", "updated code")
			},
			subDir: "artifact1",
		},
		{
			description:            "updated artifact in dirty repo",
			variantTags:            "v1-dirty",
			variantCommitSha:       "b610928dc27484cc56990bc77622aab0dbd67131-dirty",
			variantAbbrevCommitSha: "b610928-dirty",
			variantTreeSha:         "36651c832d8bf5ca1e84c6dc23bb8678fa51cf3e-dirty",
			variantAbbrevTreeSha:   "36651c8-dirty",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					mkdir("artifact1").write("artifact1/source.go", "code").
					mkdir("artifact2").write("artifact2/source.go", "other code").
					add("artifact1/source.go", "artifact2/source.go").
					commit("initial").tag("v1").
					write("artifact2/source.go", "updated code")
			},
			subDir: "artifact2",
		},
		{
			description:            "additional commit in other artifact",
			variantTags:            "0d16f59",
			variantCommitSha:       "0d16f59900bd63dd39425d6085d3f1333b66804f",
			variantAbbrevCommitSha: "0d16f59",
			variantTreeSha:         "3bed02ca656e336307e4eb4d80080d7221cba62c",
			variantAbbrevTreeSha:   "3bed02c",
			createGitRepo: func(dir string) {
				gitInit(t, dir).
					mkdir("artifact1").write("artifact1/source.go", "code").
					mkdir("artifact2").write("artifact2/source.go", "other code").
					add("artifact1/source.go", "artifact2/source.go").
					commit("initial").
					write("artifact2/source.go", "updated code").
					add("artifact2/source.go").
					commit("update artifact2")
			},
			subDir: "artifact1",
		},
		{
			description: "non git repo",
			createGitRepo: func(dir string) {
				ioutil.WriteFile(filepath.Join(dir, "source.go"), []byte("code"), os.ModePerm)
			},
			shouldErr: true,
		},
		{
			description: "git repo with no commit",
			createGitRepo: func(dir string) {
				gitInit(t, dir)
			},
			shouldErr: true,
		},
	}

	for _, test := range tests {
		test := test
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Parallel()

			tmpDir := t.NewTempDir()
			test.createGitRepo(tmpDir.Root())
			workspace := tmpDir.Path(test.subDir)

			for variant, expectedTag := range map[string]string{
				"Tags":            test.variantTags,
				"CommitSha":       test.variantCommitSha,
				"AbbrevCommitSha": test.variantAbbrevCommitSha,
				"TreeSha":         test.variantTreeSha,
				"AbbrevTreeSha":   test.variantAbbrevTreeSha,
			} {
				tagger, err := NewGitCommit("", variant)
				t.CheckNoError(err)

				tag, err := tagger.GenerateTag(workspace, "test")

				t.CheckErrorAndDeepEqual(test.shouldErr, err, expectedTag, tag)
			}
		})
	}
}

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
					write("source.go", "code").
					add("source.go").
					commit("initial")
			},
		},
		{
			description: "non git repo",
			createGitRepo: func(dir string) {
				ioutil.WriteFile(filepath.Join(dir, "source.go"), []byte("code"), os.ModePerm)
			},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		test := test
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir()
			test.createGitRepo(tmpDir.Root())
			workspace := tmpDir.Path(test.subDir)

			for variant, expectedTag := range map[string]string{
				"Tags":            test.variantTags,
				"CommitSha":       test.variantCommitSha,
				"AbbrevCommitSha": test.variantAbbrevCommitSha,
				"TreeSha":         test.variantTreeSha,
				"AbbrevTreeSha":   test.variantAbbrevTreeSha,
			} {
				tagger, err := NewGitCommit("", variant)
				t.CheckNoError(err)

				tag, err := GenerateFullyQualifiedImageName(tagger, workspace, "test")

				t.CheckErrorAndDeepEqual(test.shouldErr, err, expectedTag, tag)
			}
		})
	}
}

func TestGitCommitSubDirectory(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		tmpDir := t.NewTempDir()
		gitInit(t.T, tmpDir.Root()).mkdir("sub/sub").commit("initial")
		workspace := tmpDir.Path("sub/sub")

		tagger, err := NewGitCommit("", "Tags")
		t.CheckNoError(err)
		tag, err := tagger.GenerateTag(workspace, "test")
		t.CheckNoError(err)
		t.CheckDeepEqual("a7b32a6", tag)

		tagger, err = NewGitCommit("", "CommitSha")
		t.CheckNoError(err)
		tag, err = tagger.GenerateTag(workspace, "test")
		t.CheckNoError(err)
		t.CheckDeepEqual("a7b32a69335a6daa51bd89cc1bf30bd31df228ba", tag)

		tagger, err = NewGitCommit("", "AbbrevCommitSha")
		t.CheckNoError(err)
		tag, err = tagger.GenerateTag(workspace, "test")
		t.CheckNoError(err)
		t.CheckDeepEqual("a7b32a6", tag)

		tagger, err = NewGitCommit("", "TreeSha")
		t.CheckNoError(err)
		_, err = tagger.GenerateTag(workspace, "test")
		t.CheckErrorAndDeepEqual(true, err, "a7b32a6", tag)

		tagger, err = NewGitCommit("", "AbbrevTreeSha")
		t.CheckNoError(err)
		_, err = tagger.GenerateTag(workspace, "test")
		t.CheckErrorAndDeepEqual(true, err, "a7b32a6", tag)
	})
}

func TestPrefix(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		tmpDir := t.NewTempDir()
		gitInit(t.T, tmpDir.Root()).commit("initial")
		workspace := tmpDir.Path(".")

		tagger, err := NewGitCommit("tag-", "Tags")
		t.CheckNoError(err)
		tag, err := tagger.GenerateTag(workspace, "test")
		t.CheckNoError(err)
		t.CheckDeepEqual("tag-a7b32a6", tag)

		tagger, err = NewGitCommit("commit-", "CommitSha")
		t.CheckNoError(err)
		tag, err = tagger.GenerateTag(workspace, "test")
		t.CheckNoError(err)
		t.CheckDeepEqual("commit-a7b32a69335a6daa51bd89cc1bf30bd31df228ba", tag)
	})
}

func TestInvalidVariant(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		_, err := NewGitCommit("", "Invalid")

		t.CheckErrorContains("\"Invalid\" is not a valid git tagger variant", err)
	})
}

func TestSanitizeTag(t *testing.T) {
	testutil.Run(t, "valid tags", func(t *testutil.T) {
		t.CheckDeepEqual("abcdefghijklmnopqrstuvwxyz", sanitizeTag("abcdefghijklmnopqrstuvwxyz"))
		t.CheckDeepEqual("ABCDEFGHIJKLMNOPQRSTUVWXYZ", sanitizeTag("ABCDEFGHIJKLMNOPQRSTUVWXYZ"))
		t.CheckDeepEqual("0123456789-_.", sanitizeTag("0123456789-_."))
		t.CheckDeepEqual("_v1", sanitizeTag("_v1"))
	})

	testutil.Run(t, "sanitized tags", func(t *testutil.T) {
		t.CheckDeepEqual("v_1", sanitizeTag("v/1"))
		t.CheckDeepEqual("v____1", sanitizeTag("v%$@!1"))
		t.CheckDeepEqual("__v1", sanitizeTag("--v1"))
		t.CheckDeepEqual("__v1", sanitizeTag("..v1"))
		t.CheckDeepEqual(128, len(sanitizeTag(strings.Repeat("0123456789", 20))))
	})
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

func (g *gitRepo) write(file string, content string) *gitRepo {
	err := ioutil.WriteFile(filepath.Join(g.dir, file), []byte(content), os.ModePerm)
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
