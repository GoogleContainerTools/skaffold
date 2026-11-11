// Copyright 2026 The Go Language Server Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package uri

import "strings"

// JoinPath joins URI path segments using Node path.posix semantics.
func JoinPath(u URI, segments ...string) (URI, error) {
	paths := make([]string, 0, len(segments)+1)
	paths = append(paths, u.Path())
	paths = append(paths, segments...)
	return withPath(u, posixJoin(paths...))
}

// ResolvePath resolves URI path segments using Node path.posix semantics.
func ResolvePath(u URI, segments ...string) (URI, error) {
	base := u.Path()
	slashAdded := false
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
		slashAdded = true
	}
	paths := make([]string, 0, len(segments)+1)
	paths = append(paths, base)
	paths = append(paths, segments...)
	path := posixResolve(paths...)
	if slashAdded && strings.HasPrefix(path, "/") && u.Authority() == "" {
		path = path[1:]
	}
	return withPath(u, path)
}

// Dirname returns a URI with its path replaced by Node path.posix.dirname.
func Dirname(u URI) URI {
	path := u.Path()
	if path == "" || path == "/" {
		return u
	}
	dir := posixDirname(path)
	if dir == "." {
		dir = ""
	}
	return mustWithPath(u, dir)
}

// Basename returns Node path.posix.basename of the URI path.
func Basename(u URI) string {
	return posixBasename(u.Path())
}

// Extname returns Node path.posix.extname of the URI path.
func Extname(u URI) string {
	return posixExtname(u.Path())
}

func withPath(u URI, path string) (URI, error) {
	c := u.Components()
	c.Path = path
	return newURI(&c, false, "with path", u.String())
}

func mustWithPath(u URI, path string) URI {
	v, err := withPath(u, path)
	if err != nil {
		panic(err)
	}
	return v
}
