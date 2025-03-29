package v02

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/buildpacks/lifecycle/platform/files"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

type TagInfo struct {
	Name    string
	Message string
	Type    string
	TagHash string
	TagTime time.Time
}

func GitMetadata(appPath string) *files.ProjectSource {
	repo, err := git.PlainOpen(appPath)
	if err != nil {
		return nil
	}
	headRef, err := repo.Head()
	if err != nil {
		return nil
	}
	commitTagMap := generateTagsMap(repo)

	describe := parseGitDescribe(repo, headRef, commitTagMap)
	refs := parseGitRefs(repo, headRef, commitTagMap)
	remote := parseGitRemote(repo)

	projectSource := &files.ProjectSource{
		Type: "git",
		Version: map[string]interface{}{
			"commit":   headRef.Hash().String(),
			"describe": describe,
		},
		Metadata: map[string]interface{}{
			"refs": refs,
			"url":  remote,
		},
	}
	return projectSource
}

func generateTagsMap(repo *git.Repository) map[string][]TagInfo {
	commitTagMap := make(map[string][]TagInfo)
	tags, err := repo.Tags()
	if err != nil {
		return commitTagMap
	}

	tags.ForEach(func(ref *plumbing.Reference) error {
		tagObj, err := repo.TagObject(ref.Hash())
		switch err {
		case nil:
			commitTagMap[tagObj.Target.String()] = append(
				commitTagMap[tagObj.Target.String()],
				TagInfo{Name: tagObj.Name, Message: tagObj.Message, Type: "annotated", TagHash: ref.Hash().String(), TagTime: tagObj.Tagger.When},
			)
		case plumbing.ErrObjectNotFound:
			commitTagMap[ref.Hash().String()] = append(
				commitTagMap[ref.Hash().String()],
				TagInfo{Name: getRefName(ref.Name().String()), Message: "", Type: "unannotated", TagHash: ref.Hash().String(), TagTime: time.Now()},
			)
		default:
			return err
		}
		return nil
	})

	for _, tagRefs := range commitTagMap {
		sort.Slice(tagRefs, func(i, j int) bool {
			if tagRefs[i].Type == "annotated" && tagRefs[j].Type == "annotated" {
				return tagRefs[i].TagTime.After(tagRefs[j].TagTime)
			}
			if tagRefs[i].Type == "unannotated" && tagRefs[j].Type == "unannotated" {
				return tagRefs[i].Name < tagRefs[j].Name
			}
			if tagRefs[i].Type == "annotated" && tagRefs[j].Type == "unannotated" {
				return true
			}
			return false
		})
	}
	return commitTagMap
}

func generateBranchMap(repo *git.Repository) map[string][]string {
	commitBranchMap := make(map[string][]string)
	branches, err := repo.Branches()
	if err != nil {
		return commitBranchMap
	}
	branches.ForEach(func(ref *plumbing.Reference) error {
		commitBranchMap[ref.Hash().String()] = append(commitBranchMap[ref.Hash().String()], getRefName(ref.Name().String()))
		return nil
	})
	return commitBranchMap
}

// `git describe --tags --always`
func parseGitDescribe(repo *git.Repository, headRef *plumbing.Reference, commitTagMap map[string][]TagInfo) string {
	logOpts := &git.LogOptions{
		From:  headRef.Hash(),
		Order: git.LogOrderCommitterTime,
	}
	commits, err := repo.Log(logOpts)
	if err != nil {
		return ""
	}

	latestTag := headRef.Hash().String()
	commitsFromHEAD := 0
	commitBranchMap := generateBranchMap(repo)
	branchAtHEAD := getRefName(headRef.String())
	currentBranch := branchAtHEAD
	for {
		commitInfo, err := commits.Next()
		if err != nil {
			break
		}

		if branchesAtCommit, exists := commitBranchMap[commitInfo.Hash.String()]; exists {
			currentBranch = branchesAtCommit[0]
		}
		if refs, exists := commitTagMap[commitInfo.Hash.String()]; exists {
			if branchAtHEAD != currentBranch && commitsFromHEAD != 0 {
				// https://git-scm.com/docs/git-describe#_examples
				latestTag = fmt.Sprintf("%s-%d-g%s", refs[0].Name, commitsFromHEAD, headRef.Hash().String())
			} else {
				latestTag = refs[0].Name
			}
			break
		}
		commitsFromHEAD += 1
	}
	return latestTag
}

func parseGitRefs(repo *git.Repository, headRef *plumbing.Reference, commitTagMap map[string][]TagInfo) []string {
	var parsedRefs []string
	parsedRefs = append(parsedRefs, getRefName(headRef.Name().String()))
	if refs, exists := commitTagMap[headRef.Hash().String()]; exists {
		for _, ref := range refs {
			parsedRefs = append(parsedRefs, ref.Name)
		}
	}
	return parsedRefs
}

func parseGitRemote(repo *git.Repository) string {
	remotes, err := repo.Remotes()
	if err != nil || len(remotes) == 0 {
		return ""
	}

	for _, remote := range remotes {
		if remote.Config().Name == "origin" {
			return remote.Config().URLs[0]
		}
	}
	return remotes[0].Config().URLs[0]
}

// Parse ref name from refs/tags/<ref_name>
func getRefName(ref string) string {
	if refSplit := strings.SplitN(ref, "/", 3); len(refSplit) == 3 {
		return refSplit[2]
	}
	return ""
}
