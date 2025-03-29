package v02

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/buildpacks/lifecycle/platform/files"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	h "github.com/buildpacks/pack/testhelpers"
)

func TestMetadata(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "Metadata", testMetadata, spec.Sequential(), spec.Report(report.Terminal{}))
}

func testMetadata(t *testing.T, when spec.G, it spec.S) {
	var (
		repoPath string
		repo     *git.Repository
		commits  []plumbing.Hash
	)

	it.Before(func() {
		var err error

		repoPath, err = os.MkdirTemp("", "test-repo")
		h.AssertNil(t, err)

		repo, err = git.PlainInit(repoPath, false)
		h.AssertNil(t, err)

		commits = createCommits(t, repo, repoPath, 5)
	})

	it.After(func() {
		h.AssertNil(t, os.RemoveAll(repoPath))
	})

	when("#GitMetadata", func() {
		it("returns proper metadata format", func() {
			assert := h.NewAssertionManager(t)
			remoteOpts := &config.RemoteConfig{
				Name: "origin",
				URLs: []string{"git@github.com:testorg/testproj.git", "git@github.com:testorg/testproj.git"},
			}
			repo.CreateRemote(remoteOpts)
			createUnannotatedTag(t, repo, commits[len(commits)-1], "testTag")

			output := GitMetadata(repoPath)
			expectedOutput := &files.ProjectSource{
				Type: "git",
				Version: map[string]interface{}{
					"commit":   commits[len(commits)-1].String(),
					"describe": "testTag",
				},
				Metadata: map[string]interface{}{
					"refs": []string{"master", "testTag"},
					"url":  "git@github.com:testorg/testproj.git",
				},
			}
			assert.Equal(output, expectedOutput)
		})

		it("returns nil if error occurs while fetching metadata", func() {
			output := GitMetadata("/git-path-not-found-ok")
			h.AssertNil(t, output)
		})
	})

	when("#generateTagsMap", func() {
		when("repository has no tags", func() {
			it("returns empty map", func() {
				commitTagsMap := generateTagsMap(repo)
				h.AssertEq(t, len(commitTagsMap), 0)
			})
		})

		when("repository has only unannotated tags", func() {
			it("returns correct map if commits only have one tag", func() {
				for i := 0; i < 4; i++ {
					createUnannotatedTag(t, repo, commits[i], "")
				}

				commitTagsMap := generateTagsMap(repo)
				h.AssertEq(t, len(commitTagsMap), 4)
				for i := 0; i < 4; i++ {
					tagsInfo, shouldExist := commitTagsMap[commits[i].String()]
					h.AssertEq(t, shouldExist, true)
					h.AssertNotEq(t, tagsInfo[0].Name, "")
					h.AssertEq(t, tagsInfo[0].Type, "unannotated")
					h.AssertEq(t, tagsInfo[0].Message, "")
				}
				_, shouldNotExist := commitTagsMap[commits[3].String()]
				h.AssertEq(t, shouldNotExist, true)
			})

			it("returns map sorted by ascending tag name if commits have multiple tags", func() {
				for i := 0; i < 4; i++ {
					for j := 0; j <= rand.Intn(10); j++ {
						createUnannotatedTag(t, repo, commits[i], "")
					}
				}

				commitTagsMap := generateTagsMap(repo)
				h.AssertEq(t, len(commitTagsMap), 4)
				for i := 0; i < 4; i++ {
					tagsInfo, shouldExist := commitTagsMap[commits[i].String()]
					h.AssertEq(t, shouldExist, true)

					tagsSortedByName := sort.SliceIsSorted(tagsInfo, func(i, j int) bool {
						return tagsInfo[i].Name < tagsInfo[j].Name
					})
					h.AssertEq(t, tagsSortedByName, true)
				}
			})
		})

		when("repository has only annotated tags", func() {
			it("returns correct map if commits only have one tag", func() {
				for i := 0; i < 4; i++ {
					createAnnotatedTag(t, repo, commits[i], "")
				}

				commitTagsMap := generateTagsMap(repo)
				h.AssertEq(t, len(commitTagsMap), 4)
				for i := 0; i < 4; i++ {
					tagsInfo, shouldExist := commitTagsMap[commits[i].String()]
					h.AssertEq(t, shouldExist, true)
					h.AssertNotEq(t, tagsInfo[0].Name, "")
					h.AssertEq(t, tagsInfo[0].Type, "annotated")
					h.AssertNotEq(t, tagsInfo[0].Message, "")
				}
				_, shouldNotExist := commitTagsMap[commits[3].String()]
				h.AssertEq(t, shouldNotExist, true)
			})

			it("returns map sorted by descending tag creation time if commits have multiple tags", func() {
				for i := 0; i < 4; i++ {
					for j := 0; j <= rand.Intn(10); j++ {
						createAnnotatedTag(t, repo, commits[i], "")
					}
				}

				commitTagsMap := generateTagsMap(repo)
				h.AssertEq(t, len(commitTagsMap), 4)
				for i := 0; i < 4; i++ {
					tagsInfo, shouldExist := commitTagsMap[commits[i].String()]
					h.AssertEq(t, shouldExist, true)

					tagsSortedByTime := sort.SliceIsSorted(tagsInfo, func(i, j int) bool {
						return tagsInfo[i].TagTime.After(tagsInfo[j].TagTime)
					})
					h.AssertEq(t, tagsSortedByTime, true)
				}
				_, shouldNotExist := commitTagsMap[commits[3].String()]
				h.AssertEq(t, shouldNotExist, true)
			})
		})

		when("repository has both annotated and unannotated tags", func() {
			it("returns map where annotated tags exist prior to unnanotated if commits have multiple tags", func() {
				for i := 0; i < 4; i++ {
					for j := 0; j <= rand.Intn(10); j++ {
						createAnnotatedTag(t, repo, commits[i], "")
					}
					for j := 0; j <= rand.Intn(10); j++ {
						createUnannotatedTag(t, repo, commits[i], "")
					}
				}

				commitTagsMap := generateTagsMap(repo)
				h.AssertEq(t, len(commitTagsMap), 4)
				for i := 0; i < 4; i++ {
					tagsInfo, shouldExist := commitTagsMap[commits[i].String()]
					h.AssertEq(t, shouldExist, true)

					tagsSortedByType := sort.SliceIsSorted(tagsInfo, func(i, j int) bool {
						if tagsInfo[i].Type == "annotated" && tagsInfo[j].Type == "unannotated" {
							return true
						}
						return false
					})
					h.AssertEq(t, tagsSortedByType, true)
				}
			})
		})
	})

	when("#generateBranchMap", func() {
		it("returns map with latest commit of the `master` branch", func() {
			branchMap := generateBranchMap(repo)
			h.AssertEq(t, branchMap[commits[len(commits)-1].String()][0], "master")
		})

		it("returns map with latest commit all the branches", func() {
			checkoutBranch(t, repo, "newbranch-1", true)
			newBranchCommits := createCommits(t, repo, repoPath, 3)
			checkoutBranch(t, repo, "master", false)
			checkoutBranch(t, repo, "newbranch-2", true)

			branchMap := generateBranchMap(repo)
			h.AssertEq(t, branchMap[commits[len(commits)-1].String()][0], "master")
			h.AssertEq(t, branchMap[commits[len(commits)-1].String()][1], "newbranch-2")
			h.AssertEq(t, branchMap[newBranchCommits[len(newBranchCommits)-1].String()][0], "newbranch-1")
		})
	})

	when("#parseGitDescribe", func() {
		when("all tags are defined in a single branch", func() {
			when("repository has no tags", func() {
				it("returns latest commit hash", func() {
					commitTagsMap := generateTagsMap(repo)
					headRef, err := repo.Head()
					h.AssertNil(t, err)

					output := parseGitDescribe(repo, headRef, commitTagsMap)
					h.AssertEq(t, output, commits[len(commits)-1].String())
				})
			})

			when("repository has only unannotated tags", func() {
				it("returns first tag encountered from HEAD", func() {
					for i := 0; i < 3; i++ {
						tagName := fmt.Sprintf("v0.%d-lw", i+1)
						createUnannotatedTag(t, repo, commits[i], tagName)
					}

					commitTagsMap := generateTagsMap(repo)
					headRef, err := repo.Head()
					h.AssertNil(t, err)
					output := parseGitDescribe(repo, headRef, commitTagsMap)
					h.AssertEq(t, output, "v0.3-lw")
				})

				it("returns proper tag name for tags containing `/`", func() {
					tagName := "v0.1/testing"
					t.Logf("Checking output for tag name: %s", tagName)
					createUnannotatedTag(t, repo, commits[0], tagName)

					commitTagsMap := generateTagsMap(repo)
					headRef, err := repo.Head()
					h.AssertNil(t, err)
					output := parseGitDescribe(repo, headRef, commitTagsMap)
					h.AssertContains(t, output, "v0.1/testing")
				})
			})

			when("repository has only annotated tags", func() {
				it("returns first tag encountered from HEAD", func() {
					for i := 0; i < 3; i++ {
						tagName := fmt.Sprintf("v0.%d", i+1)
						createAnnotatedTag(t, repo, commits[i], tagName)
					}

					commitTagsMap := generateTagsMap(repo)
					headRef, err := repo.Head()
					h.AssertNil(t, err)
					output := parseGitDescribe(repo, headRef, commitTagsMap)
					h.AssertEq(t, output, "v0.3")
				})
			})

			when("repository has both annotated and unannotated tags", func() {
				when("each commit has only one tag", func() {
					it("returns the first tag encountered from HEAD if unannotated tag comes first", func() {
						createAnnotatedTag(t, repo, commits[0], "ann-tag-at-commit-0")
						createUnannotatedTag(t, repo, commits[1], "unann-tag-at-commit-1")
						createAnnotatedTag(t, repo, commits[2], "ann-tag-at-commit-2")
						createUnannotatedTag(t, repo, commits[3], "unann-tag-at-commit-3")
						createUnannotatedTag(t, repo, commits[4], "unann-tag-at-commit-4")

						commitTagsMap := generateTagsMap(repo)
						headRef, err := repo.Head()
						h.AssertNil(t, err)
						output := parseGitDescribe(repo, headRef, commitTagsMap)
						h.AssertEq(t, output, "unann-tag-at-commit-4")
					})

					it("returns the first tag encountered from HEAD if annotated tag comes first", func() {
						createAnnotatedTag(t, repo, commits[0], "ann-tag-at-commit-0")
						createUnannotatedTag(t, repo, commits[1], "unann-tag-at-commit-1")
						createAnnotatedTag(t, repo, commits[2], "ann-tag-at-commit-2")
						createAnnotatedTag(t, repo, commits[3], "ann-tag-at-commit-3")

						commitTagsMap := generateTagsMap(repo)
						headRef, err := repo.Head()
						h.AssertNil(t, err)
						output := parseGitDescribe(repo, headRef, commitTagsMap)
						h.AssertEq(t, output, "ann-tag-at-commit-3")
					})

					it("returns the tag at HEAD if annotated tag exists at HEAD", func() {
						createAnnotatedTag(t, repo, commits[4], "ann-tag-at-HEAD")

						commitTagsMap := generateTagsMap(repo)
						headRef, err := repo.Head()
						h.AssertNil(t, err)
						output := parseGitDescribe(repo, headRef, commitTagsMap)
						h.AssertEq(t, output, "ann-tag-at-HEAD")
					})

					it("returns the tag at HEAD if unannotated tag exists at HEAD", func() {
						createUnannotatedTag(t, repo, commits[4], "unann-tag-at-HEAD")

						commitTagsMap := generateTagsMap(repo)
						headRef, err := repo.Head()
						h.AssertNil(t, err)
						output := parseGitDescribe(repo, headRef, commitTagsMap)
						h.AssertEq(t, output, "unann-tag-at-HEAD")
					})
				})

				when("commits have multiple tags", func() {
					it("returns most recently created tag if a commit has multiple annotated tags", func() {
						createAnnotatedTag(t, repo, commits[1], "ann-tag-1-at-commit-1")
						createAnnotatedTag(t, repo, commits[2], "ann-tag-1-at-commit-2")
						createAnnotatedTag(t, repo, commits[2], "ann-tag-2-at-commit-2")
						createAnnotatedTag(t, repo, commits[2], "ann-tag-3-at-commit-2")

						commitTagsMap := generateTagsMap(repo)
						headRef, err := repo.Head()
						h.AssertNil(t, err)

						output := parseGitDescribe(repo, headRef, commitTagsMap)
						tagsAtCommit := commitTagsMap[commits[2].String()]
						h.AssertEq(t, output, tagsAtCommit[0].Name)
						for i := 1; i < len(tagsAtCommit); i++ {
							h.AssertEq(t, tagsAtCommit[i].TagTime.Before(tagsAtCommit[0].TagTime), true)
						}
					})

					it("returns the tag name that comes first when sorted alphabetically if a commit has multiple unannotated tags", func() {
						createUnannotatedTag(t, repo, commits[1], "ann-tag-1-at-commit-1")
						createUnannotatedTag(t, repo, commits[2], "v0.000002-lw")
						createUnannotatedTag(t, repo, commits[2], "v0.0002-lw")
						createUnannotatedTag(t, repo, commits[2], "v1.0002-lw")

						commitTagsMap := generateTagsMap(repo)
						headRef, err := repo.Head()
						h.AssertNil(t, err)

						output := parseGitDescribe(repo, headRef, commitTagsMap)
						h.AssertEq(t, output, "v0.000002-lw")
					})

					it("returns annotated tag is a commit has both annotated and unannotated tags", func() {
						createAnnotatedTag(t, repo, commits[1], "ann-tag-1-at-commit-1")
						createAnnotatedTag(t, repo, commits[2], "ann-tag-1-at-commit-2")
						createUnannotatedTag(t, repo, commits[2], "unann-tag-1-at-commit-2")

						commitTagsMap := generateTagsMap(repo)
						headRef, err := repo.Head()
						h.AssertNil(t, err)

						output := parseGitDescribe(repo, headRef, commitTagsMap)
						h.AssertEq(t, output, "ann-tag-1-at-commit-2")
					})
				})
			})
		})

		when("tags are defined in multiple branches", func() {
			when("tag is defined in the latest commit of `master` branch and HEAD is at a different branch", func() {
				it("returns the tag if HEAD, master and different branch is at tags", func() {
					checkoutBranch(t, repo, "new-branch", true)
					createAnnotatedTag(t, repo, commits[len(commits)-1], "ann-tag-at-HEAD")

					headRef, err := repo.Head()
					h.AssertNil(t, err)
					commitTagsMap := generateTagsMap(repo)
					output := parseGitDescribe(repo, headRef, commitTagsMap)
					h.AssertEq(t, output, "ann-tag-at-HEAD")
				})

				when("branch is multiple commits ahead of master", func() {
					it("returns git generated version of annotated tag if branch is 2 commits ahead of `master`", func() {
						createAnnotatedTag(t, repo, commits[len(commits)-1], "testTag")
						checkoutBranch(t, repo, "new-branch", true)
						newCommits := createCommits(t, repo, repoPath, 2)

						headRef, err := repo.Head()
						h.AssertNil(t, err)
						commitTagsMap := generateTagsMap(repo)
						output := parseGitDescribe(repo, headRef, commitTagsMap)
						expectedOutput := fmt.Sprintf("testTag-2-g%s", newCommits[len(newCommits)-1].String())
						h.AssertEq(t, output, expectedOutput)
					})

					it("returns git generated version of unannotated tag if branch is 5 commits ahead of `master`", func() {
						createUnannotatedTag(t, repo, commits[len(commits)-1], "testTag")
						checkoutBranch(t, repo, "new-branch", true)
						newCommits := createCommits(t, repo, repoPath, 5)

						headRef, err := repo.Head()
						h.AssertNil(t, err)
						commitTagsMap := generateTagsMap(repo)
						output := parseGitDescribe(repo, headRef, commitTagsMap)
						expectedOutput := fmt.Sprintf("testTag-5-g%s", newCommits[len(newCommits)-1].String())
						h.AssertEq(t, output, expectedOutput)
					})

					it("returns the commit hash if only the diverged tree of `master` branch has a tag", func() {
						checkoutBranch(t, repo, "new-branch", true)
						checkoutBranch(t, repo, "master", false)
						newCommits := createCommits(t, repo, repoPath, 3)
						createUnannotatedTag(t, repo, newCommits[len(newCommits)-1], "testTagAtMaster")
						checkoutBranch(t, repo, "new-branch", false)

						headRef, err := repo.Head()
						h.AssertNil(t, err)
						commitTagsMap := generateTagsMap(repo)
						output := parseGitDescribe(repo, headRef, commitTagsMap)
						expectedOutput := commits[len(commits)-1].String()
						h.AssertEq(t, output, expectedOutput)
					})
				})
			})
		})
	})

	when("#parseGitRefs", func() {
		when("HEAD is not at a tag", func() {
			it("returns branch name if checked out branch is `master`", func() {
				commitTagsMap := generateTagsMap(repo)
				headRef, err := repo.Head()
				h.AssertNil(t, err)
				output := parseGitRefs(repo, headRef, commitTagsMap)
				expectedOutput := []string{"master"}
				h.AssertEq(t, output, expectedOutput)
			})

			it("returns branch name if checked out branch is not `master`", func() {
				checkoutBranch(t, repo, "tests/05-05/test-branch", true)
				createCommits(t, repo, repoPath, 1)

				commitTagsMap := generateTagsMap(repo)
				headRef, err := repo.Head()
				h.AssertNil(t, err)
				output := parseGitRefs(repo, headRef, commitTagsMap)
				expectedOutput := []string{"tests/05-05/test-branch"}
				h.AssertEq(t, output, expectedOutput)
			})
		})

		when("HEAD is at a commit with single tag", func() {
			it("returns annotated tag and branch name", func() {
				createAnnotatedTag(t, repo, commits[len(commits)-1], "test-tag")
				commitTagsMap := generateTagsMap(repo)
				headRef, err := repo.Head()
				h.AssertNil(t, err)
				output := parseGitRefs(repo, headRef, commitTagsMap)
				expectedOutput := []string{"master", "test-tag"}
				h.AssertEq(t, output, expectedOutput)
			})

			it("returns unannotated tag and branch name", func() {
				createUnannotatedTag(t, repo, commits[len(commits)-1], "test-tag")
				commitTagsMap := generateTagsMap(repo)
				headRef, err := repo.Head()
				h.AssertNil(t, err)
				output := parseGitRefs(repo, headRef, commitTagsMap)
				expectedOutput := []string{"master", "test-tag"}
				h.AssertEq(t, output, expectedOutput)
			})
		})

		when("HEAD is at a commit with multiple tags", func() {
			it("returns correct tag names if all tags are unannotated", func() {
				createUnannotatedTag(t, repo, commits[len(commits)-2], "v0.01-testtag-lw")
				createUnannotatedTag(t, repo, commits[len(commits)-1], "v0.02-testtag-lw-1")
				createUnannotatedTag(t, repo, commits[len(commits)-1], "v0.02-testtag-lw-2")
				commitTagsMap := generateTagsMap(repo)
				headRef, err := repo.Head()
				h.AssertNil(t, err)
				output := parseGitRefs(repo, headRef, commitTagsMap)
				expectedOutput := []string{"master", "v0.02-testtag-lw-1", "v0.02-testtag-lw-2"}
				h.AssertEq(t, output, expectedOutput)
			})

			it("returns correct tag names if all tags are annotated", func() {
				createAnnotatedTag(t, repo, commits[len(commits)-2], "v0.01-testtag")
				createAnnotatedTag(t, repo, commits[len(commits)-1], "v0.02-testtag")
				createAnnotatedTag(t, repo, commits[len(commits)-1], "v0.03-testtag")
				commitTagsMap := generateTagsMap(repo)
				headRef, err := repo.Head()
				h.AssertNil(t, err)
				output := parseGitRefs(repo, headRef, commitTagsMap)
				expectedOutput := []string{"master", "v0.02-testtag", "v0.03-testtag"}
				sort.Strings(output)
				sort.Strings(expectedOutput)
				h.AssertEq(t, output, expectedOutput)
			})

			it("returns correct tag names for both tag types", func() {
				createUnannotatedTag(t, repo, commits[len(commits)-3], "v0.001-testtag-lw")
				createAnnotatedTag(t, repo, commits[len(commits)-2], "v0.01-testtag")
				createUnannotatedTag(t, repo, commits[len(commits)-1], "v0.02-testtag-lw-1")
				createUnannotatedTag(t, repo, commits[len(commits)-1], "v0.02-testtag-lw-2")
				createAnnotatedTag(t, repo, commits[len(commits)-1], "v0.02-testtag-1")

				commitTagsMap := generateTagsMap(repo)
				headRef, err := repo.Head()
				h.AssertNil(t, err)
				output := parseGitRefs(repo, headRef, commitTagsMap)
				expectedOutput := []string{"master", "v0.02-testtag-1", "v0.02-testtag-lw-1", "v0.02-testtag-lw-2"}
				h.AssertEq(t, output, expectedOutput)
			})

			it("returns correct tag names for both tag types when branch is not `master`", func() {
				checkoutBranch(t, repo, "test-branch", true)
				createUnannotatedTag(t, repo, commits[len(commits)-3], "v0.001-testtag-lw")
				createAnnotatedTag(t, repo, commits[len(commits)-2], "v0.01-testtag")
				createUnannotatedTag(t, repo, commits[len(commits)-1], "v0.02-testtag-lw-1")
				createUnannotatedTag(t, repo, commits[len(commits)-1], "v0.02-testtag-lw-2")
				createAnnotatedTag(t, repo, commits[len(commits)-1], "v0.02-testtag-1")
				createAnnotatedTag(t, repo, commits[len(commits)-1], "v0.02-testtag-2")

				commitTagsMap := generateTagsMap(repo)
				headRef, err := repo.Head()
				h.AssertNil(t, err)
				output := parseGitRefs(repo, headRef, commitTagsMap)
				expectedOutput := []string{"test-branch", "v0.02-testtag-1", "v0.02-testtag-2", "v0.02-testtag-lw-1", "v0.02-testtag-lw-2"}
				sort.Strings(output)
				sort.Strings(expectedOutput)
				h.AssertEq(t, output, expectedOutput)
			})
		})
	})

	when("#parseGitRemote", func() {
		it("returns fetch url if remote `origin` exists", func() {
			remoteOpts := &config.RemoteConfig{
				Name: "origin",
				URLs: []string{"git@github.com:testorg/testproj.git", "git@github.com:testorg/testproj.git"},
			}
			repo.CreateRemote(remoteOpts)

			output := parseGitRemote(repo)
			h.AssertEq(t, output, "git@github.com:testorg/testproj.git")
		})

		it("returns empty string if no remote exists", func() {
			output := parseGitRemote(repo)
			h.AssertEq(t, output, "")
		})

		it("returns fetch url if fetch and push URLs are different", func() {
			remoteOpts := &config.RemoteConfig{
				Name: "origin",
				URLs: []string{"git@fetch.com:testorg/testproj.git", "git@pushing-p-github.com:testorg/testproj.git"},
			}
			repo.CreateRemote(remoteOpts)

			output := parseGitRemote(repo)
			h.AssertEq(t, output, "git@fetch.com:testorg/testproj.git")
		})
	})

	when("#getRefName", func() {
		it("return proper ref for refs with `/`", func() {
			output := getRefName("refs/tags/this/is/a/tag/with/slashes")
			h.AssertEq(t, output, "this/is/a/tag/with/slashes")
		})
	})
}

func createCommits(t *testing.T, repo *git.Repository, repoPath string, numberOfCommits int) []plumbing.Hash {
	worktree, err := repo.Worktree()
	h.AssertNil(t, err)

	var commitHashes []plumbing.Hash
	for i := 0; i < numberOfCommits; i++ {
		file, err := os.CreateTemp(repoPath, h.RandString(10))
		h.AssertNil(t, err)
		defer file.Close()

		_, err = worktree.Add(filepath.Base(file.Name()))
		h.AssertNil(t, err)

		commitMsg := fmt.Sprintf("%s %d", "test commit number", i)
		commitOpts := git.CommitOptions{
			All: true,
			Author: &object.Signature{
				Name:  "Test Author",
				Email: "testauthor@test.com",
				When:  time.Now(),
			},
			Committer: &object.Signature{
				Name:  "Test Committer",
				Email: "testcommitter@test.com",
				When:  time.Now(),
			},
		}
		commitHash, err := worktree.Commit(commitMsg, &commitOpts)
		h.AssertNil(t, err)
		commitHashes = append(commitHashes, commitHash)
	}
	return commitHashes
}

func createUnannotatedTag(t *testing.T, repo *git.Repository, commitHash plumbing.Hash, tagName string) {
	if tagName == "" {
		version := rand.Float32()*10 + float32(rand.Intn(20))
		tagName = fmt.Sprintf("v%f-lw", version)
	}
	_, err := repo.CreateTag(tagName, commitHash, nil)
	h.AssertNil(t, err)
}

func createAnnotatedTag(t *testing.T, repo *git.Repository, commitHash plumbing.Hash, tagName string) {
	if tagName == "" {
		version := rand.Float32()*10 + float32(rand.Intn(20))
		tagName = fmt.Sprintf("v%f-%s", version, h.RandString(5))
	}
	tagMessage := fmt.Sprintf("This is an annotated tag for version - %s", tagName)
	tagOpts := &git.CreateTagOptions{
		Message: tagMessage,
		Tagger: &object.Signature{
			Name:  "Test Tagger",
			Email: "testtagger@test.com",
			When:  time.Now().Add(time.Hour*time.Duration(rand.Intn(100)) + time.Minute*time.Duration(rand.Intn(100))),
		},
	}
	_, err := repo.CreateTag(tagName, commitHash, tagOpts)
	h.AssertNil(t, err)
}

func checkoutBranch(t *testing.T, repo *git.Repository, branchName string, newBranch bool) {
	worktree, err := repo.Worktree()
	h.AssertNil(t, err)

	var fullBranchName string
	if branchName == "" {
		fullBranchName = "refs/heads/" + h.RandString(10)
	} else {
		fullBranchName = "refs/heads/" + branchName
	}

	checkoutOpts := &git.CheckoutOptions{
		Branch: plumbing.ReferenceName(fullBranchName),
		Create: newBranch,
	}
	err = worktree.Checkout(checkoutOpts)
	h.AssertNil(t, err)
}
