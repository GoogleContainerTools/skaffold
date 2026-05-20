package pathutil

import "strings"

// IsDotGitName reports whether name is `.git` or its 8.3 NTFS short
// alias `git~1`, case-insensitively. Both are forbidden as path
// components (and as submodule names) because they refer to the
// repository's own metadata directory.
//
// File names that do not conform to the 8.3 format (up to eight
// characters for the basename, three for the file extension) are
// associated with a so-called "short name" on NTFS — at least on
// the `C:` drive by default — which means that `git~1/` is a valid
// way to refer to `.git/`.
func IsDotGitName(name string) bool {
	switch strings.ToLower(name) {
	case ".git", "git~1":
		return true
	}
	return false
}
