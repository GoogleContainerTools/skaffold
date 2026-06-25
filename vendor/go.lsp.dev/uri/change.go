// Copyright 2026 The Go Language Server Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package uri

// Change describes URI component replacements for With.
//
// A nil field keeps the existing component. A non-nil pointer replaces that
// component; a pointer to the empty string clears it, except Scheme follows
// vscode-uri non-strict scheme fixing and becomes "file" when empty.
type Change struct {
	Scheme    *string
	Authority *string
	Path      *string
	Query     *string
	Fragment  *string
}

// With returns a new URI with selected decoded components changed.
func (u URI) With(change Change) (URI, error) {
	c := u.Components()
	before := c
	if change.Scheme != nil {
		c.Scheme = *change.Scheme
	}
	if change.Authority != nil {
		c.Authority = *change.Authority
	}
	if change.Path != nil {
		c.Path = *change.Path
	}
	if change.Query != nil {
		c.Query = *change.Query
	}
	if change.Fragment != nil {
		c.Fragment = *change.Fragment
	}
	if c == before {
		return u, nil
	}
	return newURI(&c, false, "with", u.String())
}

// MarshalText returns the canonical URI string as text.
func (u URI) MarshalText() ([]byte, error) {
	return []byte(u.String()), nil
}

// UnmarshalText parses text as a URI using non-strict vscode-uri semantics.
func (u *URI) UnmarshalText(text []byte) error {
	v, err := Parse(string(text))
	if err != nil {
		return err
	}
	*u = v
	return nil
}
