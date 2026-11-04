// Copyright 2026 The Go Language Server Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package uri

type rawParts struct {
	scheme    string
	authority string
	path      string
	query     string
	fragment  string

	authorityStart int
	authorityEnd   int
	pathStart      int
	pathEnd        int
	queryEnd       int
	fragmentStart  int
	hasQuery       bool
	hasFragment    bool
}

func splitRaw(s string) rawParts {
	var p rawParts
	pos := splitScheme(s, &p)
	pos = splitAuthority(s, pos, &p)
	pos = splitPath(s, pos, &p)
	splitQueryFragment(s, pos, &p)
	return p
}

func splitScheme(s string, p *rawParts) int {
	if schemeEnd := rawSchemeEnd(s); schemeEnd >= 0 {
		p.scheme = s[:schemeEnd]
		return schemeEnd + 1
	}
	return 0
}

func splitAuthority(s string, pos int, p *rawParts) int {
	if len(s)-pos >= 2 && s[pos] == '/' && s[pos+1] == '/' {
		authStart := pos + 2
		authEnd := authStart
		for authEnd < len(s) && s[authEnd] != '/' && s[authEnd] != '?' && s[authEnd] != '#' {
			authEnd++
		}
		p.authorityStart = authStart
		p.authorityEnd = authEnd
		p.authority = s[authStart:authEnd]
		pos = authEnd
	} else {
		p.authorityStart = pos
		p.authorityEnd = pos
	}
	return pos
}

func splitPath(s string, pos int, p *rawParts) int {
	p.pathStart = pos
	pathEnd := pos
	for pathEnd < len(s) && s[pathEnd] != '?' && s[pathEnd] != '#' {
		pathEnd++
	}
	p.pathEnd = pathEnd
	p.path = s[pos:pathEnd]
	return pathEnd
}

func splitQueryFragment(s string, pos int, p *rawParts) {
	if pos < len(s) && s[pos] == '?' {
		p.hasQuery = true
		queryStart := pos + 1
		queryEnd := queryStart
		for queryEnd < len(s) && s[queryEnd] != '#' {
			queryEnd++
		}
		p.query = s[queryStart:queryEnd]
		p.queryEnd = queryEnd
		pos = queryEnd
	} else {
		p.queryEnd = pos
	}

	if pos < len(s) && s[pos] == '#' {
		p.hasFragment = true
		p.fragmentStart = pos + 1
		p.fragment = s[p.fragmentStart:]
	} else {
		p.fragmentStart = len(s)
	}
}

func rawSchemeEnd(s string) int {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case ':':
			if i == 0 {
				return -1
			}
			return i
		case '/', '?', '#':
			return -1
		}
	}
	return -1
}
