// Copyright 2026 The Go Language Server Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package uri

import "strings"

func formatComponents(c *Components, skipEncoding bool) string {
	var b strings.Builder
	b.Grow(len(c.Scheme) + len(c.Authority) + len(c.Path) + len(c.Query) + len(c.Fragment) + 8)
	if c.Scheme != "" {
		b.WriteString(c.Scheme)
		b.WriteByte(':')
	}
	if c.Authority != "" || c.Scheme == schemeFile {
		b.WriteString("//")
	}
	if c.Authority != "" {
		writeAuthority(&b, c.Authority, skipEncoding)
	}
	path := formatPathDrive(c.Path)
	if skipEncoding {
		writeComponentMinimal(&b, path)
	} else {
		writeComponentFast(&b, path, true, false)
	}
	if c.Query != "" {
		b.WriteByte('?')
		if skipEncoding {
			writeComponentMinimal(&b, c.Query)
		} else {
			writeComponentFast(&b, c.Query, false, false)
		}
	}
	if c.Fragment != "" {
		b.WriteByte('#')
		if skipEncoding {
			b.WriteString(c.Fragment)
		} else {
			writeComponentFast(&b, c.Fragment, false, false)
		}
	}
	return b.String()
}

func writeAuthority(b *strings.Builder, authority string, skipEncoding bool) {
	at := strings.IndexByte(authority, '@')
	if at >= 0 {
		userinfo := authority[:at]
		colon := strings.LastIndexByte(userinfo, ':')
		if colon < 0 {
			writeAuthorityPart(b, userinfo, false, skipEncoding)
		} else {
			writeAuthorityPart(b, userinfo[:colon], false, skipEncoding)
			b.WriteByte(':')
			writeAuthorityPart(b, userinfo[colon+1:], true, skipEncoding)
		}
		b.WriteByte('@')
		authority = authority[at+1:]
	}

	authority = strings.ToLower(authority)
	colon := strings.LastIndexByte(authority, ':')
	if colon < 0 {
		writeAuthorityPart(b, authority, true, skipEncoding)
		return
	}
	writeAuthorityPart(b, authority[:colon], true, skipEncoding)
	b.WriteString(authority[colon:])
}

func writeAuthorityPart(b *strings.Builder, s string, isAuthority, skipEncoding bool) {
	if skipEncoding {
		writeComponentMinimal(b, s)
		return
	}
	writeComponentFast(b, s, false, isAuthority)
}

func formatPathDrive(path string) string {
	if len(path) >= 3 && path[0] == '/' && path[2] == ':' && isUpperASCII(path[1]) {
		return "/" + string(path[1]+'a'-'A') + ":" + path[3:]
	}
	if len(path) >= 2 && path[1] == ':' && isUpperASCII(path[0]) {
		return string(path[0]+'a'-'A') + ":" + path[2:]
	}
	return path
}

func isUpperASCII(c byte) bool {
	return c >= 'A' && c <= 'Z'
}
