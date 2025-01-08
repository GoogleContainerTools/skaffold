// Copyright 2024 ko Build Authors All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// MIT License
//
// Copyright (c) 2016-2022 Carlos Alexandro Becker
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package git

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Info includes tags and diffs used in some point.
type Info struct {
	Branch      string
	Tag         string
	ShortCommit string
	FullCommit  string
	CommitDate  time.Time
	Dirty       bool
}

// TemplateValue converts this Info into a map for use in golang templates.
func (i Info) TemplateValue() map[string]interface{} {
	treeState := "clean"
	if i.Dirty {
		treeState = "dirty"
	}

	return map[string]interface{}{
		"Branch":          i.Branch,
		"Tag":             i.Tag,
		"ShortCommit":     i.ShortCommit,
		"FullCommit":      i.FullCommit,
		"CommitDate":      i.CommitDate.UTC().Format(time.RFC3339),
		"CommitTimestamp": i.CommitDate.UTC().Unix(),
		"IsDirty":         i.Dirty,
		"IsClean":         !i.Dirty,
		"TreeState":       treeState,
	}
}

// GetInfo returns git information for the given directory
func GetInfo(ctx context.Context, dir string) (Info, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return Info{}, ErrNoGit
	}

	if !isRepo(ctx, dir) {
		return Info{}, ErrNotRepository
	}

	branch, err := getBranch(ctx, dir)
	if err != nil {
		return Info{}, fmt.Errorf("couldn't get current branch: %w", err)
	}
	short, err := getShortCommit(ctx, dir)
	if err != nil {
		return Info{}, fmt.Errorf("couldn't get current commit: %w", err)
	}
	full, err := getFullCommit(ctx, dir)
	if err != nil {
		return Info{}, fmt.Errorf("couldn't get current commit: %w", err)
	}
	date, err := getCommitDate(ctx, dir)
	if err != nil {
		return Info{}, fmt.Errorf("couldn't get commit date: %w", err)
	}

	dirty := checkDirty(ctx, dir)

	// TODO: allow exclusions.
	tag, err := getTag(ctx, dir, []string{})
	if err != nil {
		return Info{
			Branch:      branch,
			FullCommit:  full,
			ShortCommit: short,
			CommitDate:  date,
			Tag:         "v0.0.0",
			Dirty:       dirty != nil,
		}, errors.Join(ErrNoTag, dirty)
	}

	return Info{
		Branch:      branch,
		Tag:         tag,
		FullCommit:  full,
		ShortCommit: short,
		CommitDate:  date,
		Dirty:       dirty != nil,
	}, dirty
}

// isRepo returns true if current folder is a git repository.
func isRepo(ctx context.Context, dir string) bool {
	out, err := run(ctx, runConfig{
		dir:  dir,
		args: []string{"rev-parse", "--is-inside-work-tree"},
	})
	return err == nil && strings.TrimSpace(out) == "true"
}

// checkDirty returns an error if the current git repository is dirty.
func checkDirty(ctx context.Context, dir string) error {
	out, err := run(ctx, runConfig{
		dir:  dir,
		args: []string{"status", "--porcelain"},
	})
	if strings.TrimSpace(out) != "" || err != nil {
		return ErrDirty{status: out}
	}
	return nil
}

func getBranch(ctx context.Context, dir string) (string, error) {
	return clean(run(ctx, runConfig{
		dir:  dir,
		args: []string{"rev-parse", "--abbrev-ref", "HEAD", "--quiet"},
	}))
}

func getCommitDate(ctx context.Context, dir string) (time.Time, error) {
	ct, err := clean(run(ctx, runConfig{
		dir:  dir,
		args: []string{"show", "--format='%ct'", "HEAD", "--quiet"},
	}))
	if err != nil {
		return time.Time{}, err
	}
	if ct == "" {
		return time.Time{}, nil
	}
	i, err := strconv.ParseInt(ct, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	t := time.Unix(i, 0).UTC()
	return t, nil
}

func getShortCommit(ctx context.Context, dir string) (string, error) {
	return clean(run(ctx, runConfig{
		dir:  dir,
		args: []string{"show", "--format=%h", "HEAD", "--quiet"},
	}))
}

func getFullCommit(ctx context.Context, dir string) (string, error) {
	return clean(run(ctx, runConfig{
		dir:  dir,
		args: []string{"show", "--format=%H", "HEAD", "--quiet"},
	}))
}

func getTag(ctx context.Context, dir string, excluding []string) (string, error) {
	// this will get the last tag, even if it wasn't made against the
	// last commit...
	tags, err := cleanAllLines(gitDescribe(ctx, dir, "HEAD", excluding))
	if err != nil {
		return "", err
	}
	tag := filterOut(tags, excluding)
	return tag, err
}

func gitDescribe(ctx context.Context, dir, ref string, excluding []string) (string, error) {
	args := []string{
		"describe",
		"--tags",
		"--abbrev=0",
		ref,
	}
	for _, exclude := range excluding {
		args = append(args, "--exclude="+exclude)
	}
	return clean(run(ctx, runConfig{
		dir:  dir,
		args: args,
	}))
}

func filterOut(tags []string, exclude []string) string {
	if len(exclude) == 0 && len(tags) > 0 {
		return tags[0]
	}
	for _, tag := range tags {
		for _, exl := range exclude {
			if exl != tag {
				return tag
			}
		}
	}
	return ""
}
