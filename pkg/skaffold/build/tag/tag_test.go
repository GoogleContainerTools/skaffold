/*
Copyright 2020 The Skaffold Authors

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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/warnings"
	"github.com/GoogleContainerTools/skaffold/testutil"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestTagger_GenerateFullyQualifiedImageName(t *testing.T) {
	//This is for testing gitCommit
	createGitRepo := func(dir string) {
		gitInit(t, dir).
			write("source.go", "code").
			add("source.go").
			commit("initial")
	}
	gitCommitExample, _ := NewGitCommit("foo-", "AbbrevCommitSha")

	// This is for testing envTemplate
	envTemplateWithoutIMG, _ := NewEnvTemplateTagger("{{.FOO}}")
	envTemplateWithIMG, _ := NewEnvTemplateTagger("{{.IMAGE_NAME}}:{{.FOO}}")
	env := []string{"FOO=BAR"}

	// This is for testing dateTime
	aLocalTimeStamp := time.Date(2015, 03, 07, 11, 06, 39, 123456789, time.Local)
	dateTimeExample := &dateTimeTagger{
		Format:   "2006-01-02",
		TimeZone: "UTC",
		timeFn:   func() time.Time { return aLocalTimeStamp },
	}
	dateTimeExpected := "2015-03-07"

	tests := []struct {
		description      string
		imageName        string
		tagger           Tagger
		expected         string
		expectedWarnings []string
		shouldErr        bool
	}{
		{
			description: "gitCommit",
			imageName:   "test",
			tagger:      gitCommitExample,
			expected:    "test:foo-eefe1b9",
		},
		{
			description: "sha256 w/o tag",
			imageName:   "test",
			tagger:      &ChecksumTagger{},
			expected:    "test:latest",
		},
		{
			description: "sha256 w/ tag",
			imageName:   "test:tag",
			tagger:      &ChecksumTagger{},
			expected:    "test:tag",
		},
		{
			description: "envTemplate w/o image",
			imageName:   "test",
			tagger:      envTemplateWithoutIMG,
			expected:    "test:BAR",
		},
		{
			description:      "envTemplate w/ image",
			imageName:        "test",
			tagger:           envTemplateWithIMG,
			expected:         "test:BAR",
			expectedWarnings: []string{"{{.IMAGE_NAME}} is deprecated, envTemplate's template should only specify the tag value. See https://skaffold.dev/docs/pipeline-stages/taggers/"},
		},
		{
			description: "dateTime",
			imageName:   "test",
			tagger:      dateTimeExample,
			expected:    "test:" + dateTimeExpected,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			fakeWarner := &warnings.Collect{}
			t.Override(&warnings.Printf, fakeWarner.Warnf)
			t.Override(&util.OSEnviron, func() []string { return env })

			tmpDir := t.NewTempDir()
			createGitRepo(tmpDir.Root())
			workspace := tmpDir.Path("")

			tag, err := GenerateFullyQualifiedImageName(test.tagger, workspace, test.imageName)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, tag)
			t.CheckDeepEqual(test.expectedWarnings, fakeWarner.Warnings)
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
