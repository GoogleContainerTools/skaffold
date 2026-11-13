// Copyright 2026 The Go Language Server Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package uri

import "strings"

func posixNormalize(p string) string {
	if p == "" {
		return "."
	}
	absolute := p[0] == '/'
	trailing := p[len(p)-1] == '/'
	stack := make([]string, 0, strings.Count(p, "/")+1)
	for part := range strings.SplitSeq(p, "/") {
		switch part {
		case "", ".":
			continue
		case "..":
			if len(stack) > 0 && stack[len(stack)-1] != ".." {
				stack = stack[:len(stack)-1]
			} else if !absolute {
				stack = append(stack, part)
			}
		default:
			stack = append(stack, part)
		}
	}

	result := strings.Join(stack, "/")
	if result == "" {
		if absolute {
			result = "/"
		} else {
			result = "."
		}
	} else if absolute {
		result = "/" + result
	}
	if trailing && result != "/" {
		result += "/"
	}
	return result
}

func posixJoin(paths ...string) string {
	totalLen := 0
	nonEmptyCount := 0
	onlyNonEmpty := ""
	for _, p := range paths {
		if p == "" {
			continue
		}
		totalLen += len(p)
		nonEmptyCount++
		onlyNonEmpty = p
	}
	if nonEmptyCount == 0 {
		return "."
	}
	if nonEmptyCount == 1 {
		return posixNormalize(onlyNonEmpty)
	}

	var joined strings.Builder
	joined.Grow(totalLen + nonEmptyCount - 1)
	for _, p := range paths {
		if p == "" {
			continue
		}
		if joined.Len() > 0 {
			joined.WriteByte('/')
		}
		joined.WriteString(p)
	}
	return posixNormalize(joined.String())
}

func posixResolve(paths ...string) string {
	resolved := ""
	absolute := false
	for i := len(paths) - 1; i >= 0 && !absolute; i-- {
		p := paths[i]
		if p == "" {
			continue
		}
		resolved = p + "/" + resolved
		absolute = p[0] == '/'
	}
	if !absolute {
		resolved = "/" + resolved
	}
	result := posixNormalize(resolved)
	for len(result) > 1 && result[len(result)-1] == '/' {
		result = result[:len(result)-1]
	}
	return result
}

func posixDirname(p string) string {
	if p == "" {
		return "."
	}
	end := len(p) - 1
	for end > 0 && p[end] == '/' {
		end--
	}
	p = p[:end+1]
	idx := strings.LastIndexByte(p, '/')
	if idx < 0 {
		return "."
	}
	if idx == 0 {
		return "/"
	}
	for idx > 0 && p[idx-1] == '/' {
		idx--
	}
	return p[:idx]
}

func posixBasename(p string) string {
	if p == "" {
		return ""
	}
	end := len(p) - 1
	for end >= 0 && p[end] == '/' {
		end--
	}
	if end < 0 {
		return ""
	}
	start := end
	for start >= 0 && p[start] != '/' {
		start--
	}
	return p[start+1 : end+1]
}

func posixExtname(p string) string {
	base := posixBasename(p)
	if base == "" || base == "." || base == ".." {
		return ""
	}
	idx := strings.LastIndexByte(base, '.')
	if idx <= 0 {
		return ""
	}
	return base[idx:]
}
