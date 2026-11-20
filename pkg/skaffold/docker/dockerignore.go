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

package docker

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/moby/patternmatcher"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/walk"
)

// NewDockerIgnorePredicate creates a walk.Predicate that checks if directory entries
// should be ignored. For an ignored directory, it also returns `filepath.SkipDir`
// unless a negated pattern (!...) can match one of the directory's descendants.
//
// The predicate is not safe for concurrent use.
func NewDockerIgnorePredicate(workspace string, excludes []string) (walk.Predicate, error) {
	ignore, err := newDockerIgnore(excludes)
	if err != nil {
		return nil, fmt.Errorf("invalid exclude patterns: %w", err)
	}

	return func(path string, info walk.Dirent) (bool, error) {
		relPath, err := filepath.Rel(workspace, path)
		if err != nil {
			return false, err
		}

		if !info.IsDir() {
			return ignore.fileIgnored(relPath)
		}

		dir, err := ignore.dir(relPath)
		if err != nil {
			return false, err
		}
		if dir.ignored && ignore.canSkip(relPath, dir) {
			return false, filepath.SkipDir
		}
		return dir.ignored, nil
	}, nil
}

// dockerIgnore decides which paths a list of .dockerignore patterns ignores.
// It gives the same answers as patternmatcher.MatchesOrParentMatches: the last
// pattern that matches a path, or one of its parent directories, decides.
// Unlike MatchesOrParentMatches, it matches every pattern against every
// directory once, instead of once for each path below that directory.
type dockerIgnore struct {
	patterns []*ignorePattern
	// fallback is set when the patterns cannot be matched one by one.
	// It then decides alone, and only directories without negated patterns are skipped.
	fallback *patternmatcher.PatternMatcher
	// dirs holds the results for the directories seen so far.
	dirs map[string]*ignoreDir
}

type ignorePattern struct {
	// matcher matches this pattern alone, without its negation.
	matcher  *patternmatcher.PatternMatcher
	negated  bool
	segments []string
	// anyDepth is set when the pattern has syntax for which the number of
	// path segments that it matches is not known.
	anyDepth bool
	// noMatch is what the matcher returns for a path that the pattern does not match.
	// Passing it back for a child of that path makes the matcher check only the child.
	noMatch    patternmatcher.MatchInfo
	hasNoMatch bool
}

type ignoreDir struct {
	// matches holds, for each pattern, whether it matches the directory or one of its parents.
	matches []bool
	ignored bool
}

func newDockerIgnore(excludes []string) (*dockerIgnore, error) {
	all, err := patternmatcher.New(excludes)
	if err != nil {
		return nil, err
	}

	d := &dockerIgnore{dirs: make(map[string]*ignoreDir)}
	for _, pattern := range all.Patterns() {
		// String returns the cleaned pattern without the leading "!".
		matcher, err := patternmatcher.New([]string{pattern.String()})
		if err != nil {
			return nil, err
		}
		if single := matcher.Patterns(); len(single) != 1 || single[0].Exclusion() || single[0].String() != pattern.String() {
			// The pattern reads differently on its own, for example "!!file".
			return &dockerIgnore{fallback: all}, nil
		}

		p := &ignorePattern{
			matcher:  matcher,
			negated:  pattern.Exclusion(),
			segments: strings.Split(pattern.String(), string(filepath.Separator)),
			anyDepth: strings.Contains(pattern.String(), "[") || (filepath.Separator != '\\' && strings.Contains(pattern.String(), `\`)),
		}
		d.patterns = append(d.patterns, p)
	}
	return d, nil
}

// fileIgnored returns whether the patterns ignore a path that is not a directory.
func (d *dockerIgnore) fileIgnored(relPath string) (bool, error) {
	if d.fallback != nil {
		return d.fallback.MatchesOrParentMatches(relPath)
	}

	parent, err := d.parent(relPath)
	if err != nil {
		return false, err
	}

	// The last pattern that matches decides.
	for i := len(d.patterns) - 1; i >= 0; i-- {
		matches, err := d.patterns[i].matches(relPath, parent, i)
		if err != nil {
			return false, err
		}
		if matches {
			return !d.patterns[i].negated, nil
		}
	}
	return false, nil
}

// dir returns which patterns match a directory.
func (d *dockerIgnore) dir(relPath string) (*ignoreDir, error) {
	if dir, found := d.dirs[relPath]; found {
		return dir, nil
	}
	if d.fallback != nil {
		ignored, err := d.fallback.MatchesOrParentMatches(relPath)
		return &ignoreDir{ignored: ignored}, err
	}

	parent, err := d.parent(relPath)
	if err != nil {
		return nil, err
	}

	dir := &ignoreDir{matches: make([]bool, len(d.patterns))}
	for i, pattern := range d.patterns {
		matches, err := pattern.matches(relPath, parent, i)
		if err != nil {
			return nil, err
		}
		if matches {
			dir.matches[i] = true
			dir.ignored = !pattern.negated
		}
	}

	d.dirs[relPath] = dir
	return dir, nil
}

// parent returns the directory that contains a path, or nil for a path at the top of the workspace.
func (d *dockerIgnore) parent(relPath string) (*ignoreDir, error) {
	parentPath := filepath.Dir(relPath)
	if parentPath == "." {
		return nil, nil
	}
	return d.dir(parentPath)
}

// canSkip returns whether everything below an ignored directory is ignored too.
// That is the case unless a negated pattern, that does not already match the
// directory or one of its parents, can match one of the directory's descendants.
func (d *dockerIgnore) canSkip(relPath string, dir *ignoreDir) bool {
	if d.fallback != nil {
		return !d.fallback.Exclusions()
	}

	var segments []string
	if relPath != "." {
		segments = strings.Split(relPath, string(filepath.Separator))
	}

	for i, pattern := range d.patterns {
		if pattern.negated && !dir.matches[i] && pattern.mayMatchBelow(segments) {
			return false
		}
	}
	return true
}

// matches returns whether the pattern matches a path or one of its parents.
// i is the index of the pattern in the list of patterns.
func (p *ignorePattern) matches(relPath string, parent *ignoreDir, i int) (bool, error) {
	// Without information about the parents, the matcher checks them itself.
	var parentInfo patternmatcher.MatchInfo
	if parent != nil {
		if parent.matches[i] {
			return true, nil
		}
		if p.hasNoMatch {
			parentInfo = p.noMatch
		}
	}

	matches, info, err := p.matcher.MatchesUsingParentResults(relPath, parentInfo)
	if err != nil {
		return false, err
	}
	if !matches && !p.hasNoMatch {
		p.noMatch, p.hasNoMatch = info, true
	}
	return matches, nil
}

// mayMatchBelow returns whether the pattern can match a path below a directory.
// It may return true for a pattern that cannot, but never false for one that can.
func (p *ignorePattern) mayMatchBelow(dir []string) bool {
	if p.anyDepth {
		return true
	}

	for i, segment := range p.segments {
		// The pattern has more segments than the directory, and the segments before agree with it.
		if i >= len(dir) {
			return true
		}
		// "**" matches any number of segments.
		if strings.Contains(segment, "**") {
			return true
		}
		if !strings.ContainsAny(segment, "*?") && segment != dir[i] {
			return false
		}
	}

	// Without "**", the pattern only matches paths with as many segments as it has.
	return false
}
