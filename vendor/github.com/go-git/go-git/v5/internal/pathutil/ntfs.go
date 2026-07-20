package pathutil

import "strings"

// IsNTFSDotGit ports upstream Git's is_ntfs_dotgit. It detects path
// components that NTFS would resolve to ".git": the canonical name
// itself and its 8.3 short-name alias "git~1", each followed by any
// number of trailing spaces or periods (which NTFS silently trims)
// and an optional Alternate Data Stream suffix (":<stream>"). The
// bare strings ".git" and "git~1" also match, mirroring upstream.
//
// Reference: upstream Git path.c is_ntfs_dotgit at L1415-L1449
// in tag v2.54.0[1].
//
// [1]: https://github.com/git/git/blob/v2.54.0/path.c#L1415-L1449
func IsNTFSDotGit(part string) bool {
	var i int
	switch {
	case len(part) >= 4 && part[0] == '.' &&
		asciiToLower(part[1]) == 'g' &&
		asciiToLower(part[2]) == 'i' &&
		asciiToLower(part[3]) == 't':
		i = 4
	case len(part) >= 5 &&
		asciiToLower(part[0]) == 'g' &&
		asciiToLower(part[1]) == 'i' &&
		asciiToLower(part[2]) == 't' &&
		part[3] == '~' && part[4] == '1':
		i = 5
	default:
		return false
	}

	for ; i < len(part); i++ {
		c := part[i]
		if c == ':' {
			return true
		}
		if c != '.' && c != ' ' {
			return false
		}
	}
	return true
}

// WindowsValidPath reports whether part is a valid Windows / NTFS
// path component for the worktree filesystem abstraction. It rejects
// NTFS-disguised variants of `.git` and `git~1` (trailing spaces,
// periods, Alternate Data Streams) and Windows reserved device
// names. Bare `.git` and `git~1` are allowed at this layer; the
// caller decides whether they are permissible at the current path
// position.
func WindowsValidPath(part string) bool {
	if IsNTFSDotGit(part) && !IsDotGitName(part) {
		return false
	}
	return !isWindowsReservedName(part)
}

// windowsReservedNames lists the Windows reserved device names.
// A path component is reserved if its base name (ignoring trailing
// spaces, extensions, and NTFS Alternate Data Streams) matches one of
// these case-insensitively.
//
// See upstream Git compat/mingw.c is_valid_win32_path().
var windowsReservedNames = []string{
	"CON", "PRN", "AUX", "NUL",
	"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
	"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
	"CONIN$", "CONOUT$",
}

func isWindowsReservedName(part string) bool {
	for _, name := range windowsReservedNames {
		if len(part) < len(name) {
			continue
		}
		if !strings.EqualFold(part[:len(name)], name) {
			continue
		}
		// Exact match or followed by space, dot, colon (ADS), or separator.
		if len(part) == len(name) {
			return true
		}
		switch part[len(name)] {
		case ' ', '.', ':':
			return true
		}
	}
	return false
}

// IsNTFSDot ports upstream Git's is_ntfs_dot_generic. It detects NTFS
// path-component variants of a dotfile name that attackers can use to
// bypass case-insensitive comparisons against the canonical name on
// Windows. The dotgit parameter is the lowercase name without the
// leading dot (e.g. "gitmodules"); shortnamePrefix is the canonical
// 6-character NTFS short-name prefix used as a fall-back match
// (e.g. "gi7eba" for ".gitmodules").
//
// Reference: upstream Git path.c is_ntfs_dot_generic at L1451-L1507
// in tag v2.54.0[1].
//
// [1]: https://github.com/git/git/blob/v2.54.0/path.c#L1451-L1507
func IsNTFSDot(name, dotgit, shortnamePrefix string) bool {
	// onlySpacesAndPeriods returns true when the suffix from start
	// onwards consists only of trailing spaces and periods, possibly
	// terminated by a NTFS Alternate Data Stream colon. Mirrors the
	// only_spaces_and_periods label in upstream's is_ntfs_dot_generic.
	onlySpacesAndPeriods := func(start int) bool {
		for i := start; i < len(name); i++ {
			c := name[i]
			if c == ':' {
				return true
			}
			if c != ' ' && c != '.' {
				return false
			}
		}
		return true
	}

	// Pattern 1: ".<dotgit>" prefix + trailing spaces / periods / ADS.
	if len(name) >= len(dotgit)+1 && name[0] == '.' &&
		strings.EqualFold(name[1:1+len(dotgit)], dotgit) {
		if onlySpacesAndPeriods(len(dotgit) + 1) {
			return true
		}
	}

	// Pattern 2: standard NTFS short name <dotgit[:6]>~[1-4].
	if len(dotgit) >= 6 && len(name) >= 8 &&
		strings.EqualFold(name[:6], dotgit[:6]) &&
		name[6] == '~' && name[7] >= '1' && name[7] <= '4' {
		if onlySpacesAndPeriods(8) {
			return true
		}
	}

	// Pattern 3: fall-back NTFS short name keyed by shortnamePrefix.
	if len(shortnamePrefix) < 6 || len(name) < 8 {
		return false
	}
	sawTilde := false
	i := 0
	for i < 8 {
		c := name[i]
		switch {
		case sawTilde:
			if c < '0' || c > '9' {
				return false
			}
		case c == '~':
			i++
			if i >= len(name) || name[i] < '1' || name[i] > '9' {
				return false
			}
			sawTilde = true
		case i >= 6:
			return false
		case c&0x80 != 0:
			return false
		default:
			if asciiToLower(c) != shortnamePrefix[i] {
				return false
			}
		}
		i++
	}
	return onlySpacesAndPeriods(8)
}

// IsNTFSDotGitmodules reports whether part is an NTFS-equivalent of
// ".gitmodules" — the file name (or any of its variants that NTFS
// would resolve to it) that attackers can use to plant submodule
// configuration disguised as a symlink. The 6-character canonical
// short-name prefix "gi7eba" mirrors upstream Git's is_ntfs_dotgitmodules.
func IsNTFSDotGitmodules(part string) bool {
	return IsNTFSDot(part, "gitmodules", "gi7eba")
}

func asciiToLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + ('a' - 'A')
	}
	return c
}
