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
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/golang/glog"
	git "gopkg.in/src-d/go-git.v4"
)

var (
	gitRegexp = regexp.MustCompile(`^\.git$`)

	gitRepoPathPrefixes = map[string]string{
		"github.com":            "blob/master/",
		"bitbucket.org":         "src/master/",
		"go.googlesource.com":   "+/refs/heads/master/",
		"code.googlesource.com": "+/refs/heads/master/",
	}
)

// GitFileURL returns the URL of a file stored in a Git repository.
// It uses the URL of the specified Git remote repository to construct this URL.
// It supports repositories hosted on github.com, bitbucket.org and googlesource.com.
func GitFileURL(filePath string, remote string) (*url.URL, error) {
	dotGitPath, err := findUpwards(filepath.Dir(filePath), gitRegexp, srcDirRegexps)
	if err != nil {
		return nil, err
	}
	relFilePath, err := filepath.Rel(filepath.Dir(dotGitPath), filePath)
	if err != nil {
		return nil, err
	}
	repoURL, err := gitRemoteURL(dotGitPath, remote)
	if err != nil {
		return nil, err
	}
	repoURL.Host = strings.TrimSuffix(repoURL.Host, ".")
	repoURL.Path = strings.TrimSuffix(repoURL.Path, ".git")
	if prefix, ok := gitRepoPathPrefixes[repoURL.Host]; ok {
		repoURL.Path = path.Join(repoURL.Path, prefix, filepath.ToSlash(relFilePath))
	} else {
		return nil, fmt.Errorf("unrecognised Git repository host: %q", repoURL)
	}

	return repoURL, nil
}

func gitRemoteURL(repoPath string, remoteName string) (*url.URL, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, err
	}
	remote, err := repo.Remote(remoteName)
	if err != nil {
		return nil, err
	}
	for _, urlStr := range remote.Config().URLs {
		u, err := url.Parse(urlStr)
		if err != nil {
			glog.Warningf("Error parsing %q as URL from remote %q in Git repo at %q: %s", urlStr, remoteName, repoPath, err)
			continue
		}
		return u, nil
	}
	return nil, fmt.Errorf("the Git remote %q does not have a valid URL", remoteName)
}
