// Copyright 2019 The Go Language Server Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package uri

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"
)

const (
	// FileScheme schema of filesystem path.
	FileScheme = "file"

	// HTTPScheme schema of http.
	HTTPScheme = "http"

	// HTTPSScheme schema of https.
	HTTPSScheme = "https"
)

const (
	hierPart = "://"
)

// URI Uniform Resource Identifier (URI) https://tools.ietf.org/html/rfc3986.
//
// This class is a simple parser which creates the basic component parts
// (http://tools.ietf.org/html/rfc3986#section-3) with minimal validation
// and encoding.
//
//	  foo://example.com:8042/over/there?name=ferret#nose
//	  \_/   \______________/\_________/ \_________/ \__/
//	   |           |            |            |        |
//	scheme     authority       path        query   fragment
//	   |   _____________________|__
//	  / \ /                        \
//	  urn:example:animal:ferret:nose
type URI string

// Filename returns the file path for the given URI.
// It is an error to call this on a URI that is not a valid filename.
func (u URI) Filename() string {
	filename, err := filename(u)
	if err != nil {
		panic(err)
	}

	return filepath.FromSlash(filename)
}

func filename(uri URI) (string, error) {
	u, err := url.ParseRequestURI(string(uri))
	if err != nil {
		return "", fmt.Errorf("failed to parse request URI: %w", err)
	}

	if u.Scheme != FileScheme {
		return "", fmt.Errorf("only file URIs are supported, got %v", u.Scheme)
	}

	if isWindowsDriveURI(u.Path) {
		u.Path = u.Path[1:]
	}

	return u.Path, nil
}

// New parses and creates a new URI from s.
func New(s string) URI {
	if u, err := url.PathUnescape(s); err == nil {
		s = u
	}

	if strings.HasPrefix(s, FileScheme+hierPart) {
		return URI(s)
	}

	return File(s)
}

// File parses and creates a new filesystem URI from path.
func File(path string) URI {
	const goRootPragma = "$GOROOT"
	if len(path) >= len(goRootPragma) && strings.EqualFold(goRootPragma, path[:len(goRootPragma)]) {
		path = runtime.GOROOT() + path[len(goRootPragma):]
	}

	if !isWindowsDrivePath(path) {
		if abs, err := filepath.Abs(path); err == nil {
			path = abs
		}
	}

	if isWindowsDrivePath(path) {
		path = "/" + path
	}

	path = filepath.ToSlash(path)
	u := url.URL{
		Scheme: FileScheme,
		Path:   path,
	}

	return URI(u.String())
}

// Parse parses and creates a new URI from s.
func Parse(s string) (u URI, err error) {
	us, err := url.Parse(s)
	if err != nil {
		return u, fmt.Errorf("url.Parse: %w", err)
	}

	switch us.Scheme {
	case FileScheme:
		ut := url.URL{
			Scheme:  FileScheme,
			Path:    us.Path,
			RawPath: filepath.FromSlash(us.Path),
		}
		u = URI(ut.String())

	case HTTPScheme, HTTPSScheme:
		ut := url.URL{
			Scheme:   us.Scheme,
			Host:     us.Host,
			Path:     us.Path,
			RawQuery: us.Query().Encode(),
			Fragment: us.Fragment,
		}
		u = URI(ut.String())

	default:
		return u, errors.New("unknown scheme")
	}

	return
}

// From returns the new URI from args.
func From(scheme, authority, path, query, fragment string) URI {
	switch scheme {
	case FileScheme:
		u := url.URL{
			Scheme:  FileScheme,
			Path:    path,
			RawPath: filepath.FromSlash(path),
		}
		return URI(u.String())

	case HTTPScheme, HTTPSScheme:
		u := url.URL{
			Scheme:   scheme,
			Host:     authority,
			Path:     path,
			RawQuery: url.QueryEscape(query),
			Fragment: fragment,
		}
		return URI(u.String())

	default:
		panic(fmt.Sprintf("unknown scheme: %s", scheme))
	}
}

// isWindowsDrivePath returns true if the file path is of the form used by Windows.
//
// We check if the path begins with a drive letter, followed by a ":".
func isWindowsDrivePath(path string) bool {
	if len(path) < 4 {
		return false
	}
	return unicode.IsLetter(rune(path[0])) && path[1] == ':'
}

// isWindowsDriveURI returns true if the file URI is of the format used by
// Windows URIs. The url.Parse package does not specially handle Windows paths
// (see https://golang.org/issue/6027). We check if the URI path has
// a drive prefix (e.g. "/C:"). If so, we trim the leading "/".
func isWindowsDriveURI(uri string) bool {
	if len(uri) < 4 {
		return false
	}
	return uri[0] == '/' && unicode.IsLetter(rune(uri[1])) && uri[2] == ':'
}
