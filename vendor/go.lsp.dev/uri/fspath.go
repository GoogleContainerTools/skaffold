// Copyright 2026 The Go Language Server Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package uri

import (
	"runtime"
	"strings"
)

// Platform selects filesystem path behavior for URI.file and uriToFsPath.
type Platform uint8

const (
	// PlatformPOSIX uses slash-separated POSIX path semantics.
	PlatformPOSIX Platform = iota
	// PlatformWindows uses Windows slash conversion and drive behavior.
	PlatformWindows
)

// File constructs a file URI for path on the host platform.
func File(path string) URI {
	return FileFor(defaultPlatform(), path)
}

// FileFor constructs a file URI for path using platform-specific vscode-uri semantics.
func FileFor(platform Platform, path string) URI {
	if platform == PlatformPOSIX {
		if u, ok := fileForCleanPOSIX(path); ok {
			return u
		}
	}

	if platform == PlatformWindows {
		path = strings.ReplaceAll(path, "\\", "/")
	}

	authority := ""
	if len(path) >= 2 && path[0] == '/' && path[1] == '/' {
		idx := strings.IndexByte(path[2:], '/')
		if idx < 0 {
			authority = path[2:]
			path = "/"
		} else {
			idx += 2
			authority = path[2:idx]
			path = path[idx:]
			if path == "" {
				path = "/"
			}
		}
	}

	components := Components{Scheme: schemeFile, Authority: authority, Path: path}
	u, err := newURI(&components, false, "file", path)
	if err != nil {
		panic(err)
	}
	return u
}

func fileForCleanPOSIX(path string) (URI, bool) {
	if path == "" || path[0] != '/' {
		return "", false
	}
	if len(path) > 1 && path[1] == '/' {
		return "", false
	}
	for i := 0; i < len(path); i++ {
		if !canPassFast(path[i], true, false) {
			return "", false
		}
	}
	return URI(schemeFile + "://" + path), true
}

// FsPath returns the filesystem path for u on the host platform.
func (u URI) FsPath() string {
	return FsPathFor(u, defaultPlatform(), false)
}

// FsPathFor returns the filesystem path for u using vscode-uri uriToFsPath
// semantics over u's canonical component view.
//
// Because URI values do not retain parse-history-only casing, UNC authorities
// and drive letters use the canonical casing exposed by Authority and Path.
func FsPathFor(u URI, platform Platform, keepDriveLetterCasing bool) string {
	if value, ok := fsPathFast(u, platform, keepDriveLetterCasing); ok {
		return value
	}

	scheme := u.Scheme()
	authority := u.Authority()
	path := u.Path()

	var value string
	switch {
	case scheme == schemeFile && authority != "" && len(path) > 1:
		value = "//" + authority + path
	case len(path) >= 3 && path[0] == '/' && isASCIIAlpha(path[1]) && path[2] == ':':
		if keepDriveLetterCasing {
			value = path[1:]
		} else {
			value = string(toLowerASCII(path[1])) + path[2:]
		}
	default:
		value = path
	}

	if platform == PlatformWindows {
		value = strings.ReplaceAll(value, "/", "\\")
	}
	return value
}

func fsPathFast(u URI, platform Platform, keepDriveLetterCasing bool) (string, bool) {
	if platform != PlatformPOSIX || keepDriveLetterCasing {
		return "", false
	}
	s := string(u)
	if len(s) < len(fileURIAbsolutePrefix) || s[:len(fileURIAbsolutePrefix)] != fileURIAbsolutePrefix {
		return "", false
	}
	path := s[fileURIPathStart:]
	if strings.ContainsAny(path, "%?#") {
		return "", false
	}
	if len(path) > 1 && path[1] == '/' {
		return "", false
	}
	if len(path) >= 3 && path[0] == '/' && isASCIIAlpha(path[1]) && path[2] == ':' {
		return "", false
	}
	return path, true
}

func defaultPlatform() Platform {
	if runtime.GOOS == "windows" {
		return PlatformWindows
	}
	return PlatformPOSIX
}

func isASCIIAlpha(c byte) bool {
	return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z'
}

func toLowerASCII(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 'a' - 'A'
	}
	return c
}
