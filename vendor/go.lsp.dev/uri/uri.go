// Copyright 2026 The Go Language Server Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package uri

import "strings"

const (
	schemeFile  = "file"
	schemeHTTP  = "http"
	schemeHTTPS = "https"
)

const (
	fileURIAbsolutePrefix = schemeFile + ":///"
	fileURIPathStart      = len(schemeFile + "://")
)

// URI is an immutable, comparable, canonical vscode-uri-compatible URI string.
//
// Constructor functions store the canonical encoded representation, equivalent to
// vscode-uri's URI.parse(input).toString() result for representable Go input.
// Decoded component accessors are derived from that canonical representation,
// so parse-history-only casing such as uppercase drive letters or authority
// hosts is intentionally not retained.
type URI string

// Components contains decoded URI components.
type Components struct {
	Scheme    string
	Authority string
	Path      string
	Query     string
	Fragment  string
}

// Parse parses s with vscode-uri non-strict semantics.
func Parse(s string) (URI, error) {
	return parse(s, false)
}

// ParseStrict parses s with vscode-uri strict semantics.
func ParseStrict(s string) (URI, error) {
	return parse(s, true)
}

// MustParse parses s and panics if parsing fails.
func MustParse(s string) URI {
	u, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

// From constructs a URI from decoded components.
//
//nolint:gocritic // Components is the public value-type API shape.
func From(c Components) (URI, error) {
	u, err := newURI(&c, false, "from", "")
	if err != nil {
		return "", err
	}
	components := u.Components()
	if err := validateComponents(&components, true, "from", ""); err != nil {
		return "", err
	}
	return u, nil
}

// String returns the canonical encoded URI string.
func (u URI) String() string {
	return string(u)
}

// StringNoEncoding returns a URI string with vscode-uri toString(true) semantics.
func (u URI) StringNoEncoding() string {
	components := u.Components()
	return formatComponents(&components, true)
}

// Scheme returns the URI scheme.
func (u URI) Scheme() string {
	raw := splitRaw(string(u))
	return raw.scheme
}

// Authority returns the decoded canonical URI authority.
//
// Authority host casing follows the canonical URI string and is therefore
// lowercased, matching URI.parse(input).toString() reparsed by vscode-uri rather
// than vscode-uri's original parse object.
func (u URI) Authority() string {
	raw := splitRaw(string(u))
	return percentDecode(raw.authority)
}

// Path returns the decoded canonical URI path.
//
// Windows drive-letter casing follows the canonical URI string. For example,
// parsing file:///C:/x and reparsing the canonical string both expose /c:/x.
func (u URI) Path() string {
	raw := splitRaw(string(u))
	return percentDecode(raw.path)
}

// Query returns the decoded URI query.
func (u URI) Query() string {
	raw := splitRaw(string(u))
	if !raw.hasQuery {
		return ""
	}
	return percentDecode(raw.query)
}

// Fragment returns the decoded URI fragment.
func (u URI) Fragment() string {
	raw := splitRaw(string(u))
	if !raw.hasFragment {
		return ""
	}
	return percentDecode(raw.fragment)
}

// Components returns all decoded URI components.
func (u URI) Components() Components {
	raw := splitRaw(string(u))
	query := ""
	if raw.hasQuery {
		query = percentDecode(raw.query)
	}
	fragment := ""
	if raw.hasFragment {
		fragment = percentDecode(raw.fragment)
	}
	return Components{
		Scheme:    raw.scheme,
		Authority: percentDecode(raw.authority),
		Path:      percentDecode(raw.path),
		Query:     query,
		Fragment:  fragment,
	}
}

// IsFile reports whether u has the exact file scheme.
func (u URI) IsFile() bool {
	raw := splitRaw(string(u))
	return raw.scheme == schemeFile
}

// IsZero reports whether u is the zero URI value.
func (u URI) IsZero() bool {
	return u == ""
}

func parse(s string, strict bool) (URI, error) {
	if u, ok := parseCanonicalFileFast(s); ok {
		return u, nil
	}

	raw := splitRaw(s)
	if u, ok, err := parseCanonicalFast(s, &raw, strict); ok || err != nil {
		return u, err
	}
	c := Components{
		Scheme:    raw.scheme,
		Authority: decodeComponent(raw.authority),
		Path:      decodeComponent(raw.path),
		Query:     decodeComponent(raw.query),
		Fragment:  decodeComponent(raw.fragment),
	}
	return newURI(&c, strict, "parse", s)
}

func newURI(c *Components, strict bool, op, input string) (URI, error) {
	components := *c
	components.Scheme = schemeFix(components.Scheme, strict)
	components.Path = referenceResolution(components.Scheme, components.Path)
	if err := validateComponents(&components, strict, op, input); err != nil {
		return "", err
	}
	return URI(formatComponents(&components, false)), nil
}

func parseCanonicalFast(s string, raw *rawParts, strict bool) (URI, bool, error) {
	if err := validateRawScheme(s, raw, strict); err != nil {
		return "", false, err
	}
	if raw.scheme == "" {
		return "", false, nil
	}
	if !rawPartsAreCanonical(s, raw) {
		return "", false, nil
	}
	c := Components{
		Scheme:    raw.scheme,
		Authority: raw.authority,
		Path:      raw.path,
		Query:     raw.query,
		Fragment:  raw.fragment,
	}
	if err := validateComponents(&c, strict, "parse", s); err != nil {
		return "", false, err
	}
	return URI(s), true, nil
}

func validateRawScheme(s string, raw *rawParts, strict bool) error {
	if raw.scheme == "" {
		if strict {
			return uriError("parse", s, ErrMissingScheme)
		}
		return nil
	}
	if !validScheme(raw.scheme) {
		return uriError("parse", s, ErrInvalidScheme)
	}
	return nil
}

func rawPartsAreCanonical(s string, raw *rawParts) bool {
	if raw.hasQuery && raw.query == "" || raw.hasFragment && raw.fragment == "" {
		return false
	}
	if raw.authority != "" && !isCanonicalAuthority(raw.authority) {
		return false
	}
	if !isCanonicalPath(raw.path) || !isCanonicalComponent(raw.query) || !isCanonicalComponent(raw.fragment) {
		return false
	}
	if !referenceAlreadyResolved(raw.scheme, raw.path) {
		return false
	}
	return authorityDelimiterIsCanonical(s, raw)
}

func authorityDelimiterIsCanonical(s string, raw *rawParts) bool {
	hasAuthorityDelimiter := len(s) >= len(raw.scheme)+3 && s[len(raw.scheme)+1] == '/' && s[len(raw.scheme)+2] == '/'
	if raw.scheme == schemeFile {
		return strings.HasPrefix(s, schemeFile+"://")
	}
	return raw.authority != "" || !hasAuthorityDelimiter
}

func parseCanonicalFileFast(s string) (URI, bool) {
	if len(s) < len(fileURIAbsolutePrefix) || s[:len(fileURIAbsolutePrefix)] != fileURIAbsolutePrefix {
		return "", false
	}
	if len(s) > fileURIPathStart+1 && s[fileURIPathStart+1] == '/' {
		return "", false
	}
	for i := fileURIPathStart + 1; i < len(s); i++ {
		if !canPassFast(s[i], true, false) {
			return "", false
		}
	}
	return URI(s), true
}

func isCanonicalAuthority(authority string) bool {
	before, after, ok := strings.Cut(authority, "@")
	hostport := authority
	if ok {
		userinfo := before
		colon := strings.LastIndexByte(userinfo, ':')
		if colon < 0 {
			if !isCanonicalComponent(userinfo) {
				return false
			}
		} else if !isCanonicalComponent(userinfo[:colon]) || !isCanonicalAuthorityPass(userinfo[colon+1:]) {
			return false
		}
		hostport = after
	}
	if hostport != strings.ToLower(hostport) {
		return false
	}
	colon := strings.LastIndexByte(hostport, ':')
	if colon < 0 {
		return isCanonicalAuthorityPass(hostport)
	}
	return isCanonicalAuthorityPass(hostport[:colon]) && isCanonicalPortTail(hostport[colon:])
}

func isCanonicalAuthorityPass(s string) bool {
	for i := 0; i < len(s); i++ {
		if !canPassFast(s[i], false, true) {
			return false
		}
	}
	return true
}

func isCanonicalPortTail(s string) bool {
	if s == "" || s[0] != ':' {
		return false
	}
	for i := 1; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func isCanonicalPath(path string) bool {
	for i := 0; i < len(path); i++ {
		if !canPassFast(path[i], true, false) {
			return false
		}
	}
	return true
}

func isCanonicalComponent(component string) bool {
	for i := 0; i < len(component); i++ {
		if !canPassFast(component[i], false, false) {
			return false
		}
	}
	return true
}

func referenceAlreadyResolved(scheme, path string) bool {
	switch scheme {
	case schemeHTTPS, schemeHTTP, schemeFile:
		return path != "" && path[0] == '/'
	default:
		return true
	}
}

func schemeFix(scheme string, strict bool) string {
	if scheme == "" && !strict {
		return schemeFile
	}
	return scheme
}

func referenceResolution(scheme, path string) string {
	switch scheme {
	case schemeHTTPS, schemeHTTP, schemeFile:
		if path == "" {
			return "/"
		}
		if path[0] != '/' {
			return "/" + path
		}
	}
	return path
}

func validateComponents(c *Components, strict bool, op, input string) error {
	if c.Scheme == "" && strict {
		return uriError(op, input, ErrMissingScheme)
	}
	if c.Scheme != "" && !validScheme(c.Scheme) {
		return uriError(op, input, ErrInvalidScheme)
	}
	if c.Path != "" {
		if c.Authority != "" {
			if c.Path[0] != '/' {
				return uriError(op, input, ErrAuthorityPath)
			}
		} else if strings.HasPrefix(c.Path, "//") {
			return uriError(op, input, ErrPathAuthority)
		}
	}
	return nil
}

func validScheme(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '+' || c == '-' || c == '.' {
			continue
		}
		return false
	}
	return true
}
