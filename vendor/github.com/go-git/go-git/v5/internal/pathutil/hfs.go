package pathutil

import "unicode"

// hfsIgnoredCodepoints contains Unicode code points that HFS+ ignores
// during path normalization. A path component containing these
// characters between the bytes of ".git" (or ".gitmodules", etc.)
// will be treated as that name by HFS+, so they have to be filtered
// out before comparison.
//
// See upstream Git utf8.c next_hfs_char in tag v2.54.0[1].
//
// [1]: https://github.com/git/git/blob/v2.54.0/utf8.c#L703-L740
var hfsIgnoredCodepoints = map[rune]struct{}{
	0x200c: {}, // ZERO WIDTH NON-JOINER
	0x200d: {}, // ZERO WIDTH JOINER
	0x200e: {}, // LEFT-TO-RIGHT MARK
	0x200f: {}, // RIGHT-TO-LEFT MARK
	0x202a: {}, // LEFT-TO-RIGHT EMBEDDING
	0x202b: {}, // RIGHT-TO-LEFT EMBEDDING
	0x202c: {}, // POP DIRECTIONAL FORMATTING
	0x202d: {}, // LEFT-TO-RIGHT OVERRIDE
	0x202e: {}, // RIGHT-TO-LEFT OVERRIDE
	0x206a: {}, // INHIBIT SYMMETRIC SWAPPING
	0x206b: {}, // ACTIVATE SYMMETRIC SWAPPING
	0x206c: {}, // INHIBIT ARABIC FORM SHAPING
	0x206d: {}, // ACTIVATE ARABIC FORM SHAPING
	0x206e: {}, // NATIONAL DIGIT SHAPES
	0x206f: {}, // NOMINAL DIGIT SHAPES
	0xfeff: {}, // ZERO WIDTH NO-BREAK SPACE
}

// IsHFSDot reports whether part would be treated as ".<needle>" on an
// HFS+ filesystem after stripping ignored Unicode code points and
// folding ASCII to lower case. The needle is the lowercase ASCII
// suffix without the leading dot (e.g. "git", "gitmodules"). It
// mirrors upstream Git's is_hfs_dot_generic and is the building
// block of IsHFSDotGit / IsHFSDotGitmodules.
//
// Reference: upstream Git utf8.c is_hfs_dot_generic at L741-L774 and
// the dotgit family at L784-L809 in tag v2.54.0[1].
//
// [1]: https://github.com/git/git/blob/v2.54.0/utf8.c#L741-L809
func IsHFSDot(part, needle string) bool {
	runes := []rune(part)
	i := 0

	// skip ignored code points, then expect '.'
	for i < len(runes) {
		if _, ok := hfsIgnoredCodepoints[runes[i]]; !ok {
			break
		}
		i++
	}
	if i >= len(runes) || runes[i] != '.' {
		return false
	}
	i++

	// match needle case-insensitively, skipping ignored code points
	for _, expected := range needle {
		for i < len(runes) {
			if _, ok := hfsIgnoredCodepoints[runes[i]]; !ok {
				break
			}
			i++
		}
		if i >= len(runes) {
			return false
		}
		r := runes[i]
		if r > 127 {
			return false
		}
		if unicode.ToLower(r) != expected {
			return false
		}
		i++
	}

	// skip trailing ignored code points
	for i < len(runes) {
		if _, ok := hfsIgnoredCodepoints[runes[i]]; !ok {
			break
		}
		i++
	}

	// must be at end of component
	return i == len(runes)
}

// IsHFSDotGit reports whether part is an HFS+ equivalent of ".git".
func IsHFSDotGit(part string) bool { return IsHFSDot(part, "git") }

// IsHFSDotGitmodules reports whether part is an HFS+ equivalent of
// ".gitmodules", catching attempts to plant the file via Unicode
// code points that HFS+ would strip during normalisation.
func IsHFSDotGitmodules(part string) bool { return IsHFSDot(part, "gitmodules") }
