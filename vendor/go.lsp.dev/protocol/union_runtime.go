// Copyright 2024 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

import (
	"bytes"

	"github.com/go-json-experiment/json/jsontext"
)

// This file implements union-arm discrimination over a raw [jsontext.Value]
// without materializing the object into a map. A [jsontext.Value] is already a
// fully-resident []byte, so a single linear scan of its top-level structure
// answers every dispatch predicate the generated decoders ask — "does the
// object carry these required keys" and "are all its keys within this known
// set" — in one allocation-free pass. The previous implementation decoded the
// value into a map[string]jsontext.Value per predicate call (and a second set
// map inside objectKeysKnown), so a single arm guard reparsed the same object
// twice and a multi-arm union reparsed it many times. Replacing that with the
// scanner below removes the per-predicate allocation entirely.

// isJSONSpace reports whether c is a JSON insignificant-whitespace byte.
func isJSONSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// skipSpace advances i past any JSON whitespace in raw.
func skipSpace(raw []byte, i int) int {
	for i < len(raw) && isJSONSpace(raw[i]) {
		i++
	}
	return i
}

// scanString consumes a JSON string starting at raw[i] (raw[i] must be '"') and
// returns the index just past the closing quote and whether a closing quote was
// found. Escaped characters, including an escaped quote, are handled so the
// closing quote is located correctly. On an unterminated string it returns
// len(raw), false.
func scanString(raw []byte, i int) (end int, ok bool) {
	i++ // opening quote
	for i < len(raw) {
		i = dvScanQuoteBackslash(raw, i)
		if i >= len(raw) {
			break
		}
		if raw[i] == '"' {
			return i + 1, true
		}
		i += 2 // skip the escape and the escaped byte
	}
	return len(raw), false
}

// skipString consumes a JSON string and returns the index just past it,
// tolerating an unterminated string by returning len(raw). Used where only the
// end position matters (value skipping).
func skipString(raw []byte, i int) int {
	end, _ := scanString(raw, i)
	return end
}

// skipValue consumes one JSON value starting at raw[i] (after optional
// whitespace) and returns the index just past it. Objects and arrays are
// skipped by depth tracking that is string-aware, so braces or brackets inside
// string values do not affect the depth. Scalars run to the next structural
// delimiter. On malformed input it returns an index at or before len(raw),
// never panicking.
func skipValue(raw []byte, i int) int {
	i = skipSpace(raw, i)
	if i >= len(raw) {
		return i
	}
	switch raw[i] {
	case '{', '[':
		depth := 0
		for i < len(raw) {
			switch raw[i] {
			case '"':
				i = skipString(raw, i)
				continue
			case '{', '[':
				depth++
			case '}', ']':
				depth--
				if depth == 0 {
					return i + 1
				}
			}
			i++
		}
		return i
	case '"':
		return skipString(raw, i)
	default:
		for i < len(raw) && raw[i] != ',' && raw[i] != '}' && raw[i] != ']' {
			i++
		}
		return i
	}
}

// objectKeys invokes fn for each top-level member key of the JSON object raw,
// passing the key as a raw (still JSON-quoted-escaped) byte slice aliasing raw.
// It stops early if fn returns false. It reports whether raw is a well-formed
// top-level object; a non-object value yields ok=false and no callbacks.
//
// The key slice passed to fn aliases raw and is the still-escaped bytes between
// the surrounding quotes. Callers comparing it against a Go string literal must
// use [keyEquals], which decodes JSON escapes on the slow path so an escaped
// spelling of a member name still matches.
func objectKeys(raw []byte, fn func(key []byte) bool) (ok bool) {
	i := skipSpace(raw, 0)
	if i >= len(raw) || raw[i] != '{' {
		return false
	}
	i++
	for {
		i = skipSpace(raw, i)
		if i >= len(raw) {
			return false // unterminated object
		}
		if raw[i] == '}' {
			return true
		}
		if raw[i] != '"' {
			return false // malformed: expected a string key
		}
		ks := i + 1
		end, term := scanString(raw, i)
		if !term {
			return false // unterminated key string
		}
		key := raw[ks : end-1] // bytes between the quotes
		i = skipSpace(raw, end)
		if i >= len(raw) || raw[i] != ':' {
			return false
		}
		i++
		i = skipValue(raw, i)
		if !fn(key) {
			return true
		}
		i = skipSpace(raw, i)
		if i < len(raw) && raw[i] == ',' {
			i++
		}
	}
}

// objectKind returns the value of a JSON object's "kind" string member, used as
// a discriminator when disambiguating union arms. It returns false if raw is
// not an object, has no "kind" member, or the member is not a JSON string. A
// JSON-escaped value (e.g. "create") is decoded to its true string, so it
// matches the same discriminator a full JSON decode would.
func objectKind(raw jsontext.Value) (string, bool) {
	var kind string
	var found bool
	objectMember(raw, "kind", func(val []byte) {
		if s, ok := unquoteJSONString(val); ok {
			kind = s
			found = true
		}
	})
	return kind, found
}

// objectMember invokes fn with the raw value bytes of each top-level member
// named want, in wire order. The value bytes alias raw. It is a no-op if raw
// is not an object or has no such member.
//
// Callers that assign inside fn naturally implement the same last-wins
// duplicate-member policy as the package's relaxed wireOptions.
func objectMember(raw []byte, want string, fn func(val []byte)) {
	i := skipSpace(raw, 0)
	if i >= len(raw) || raw[i] != '{' {
		return
	}
	i++
	for {
		i = skipSpace(raw, i)
		if i >= len(raw) || raw[i] == '}' {
			return
		}
		if raw[i] != '"' {
			return
		}
		ks := i + 1
		end, term := scanString(raw, i)
		if !term {
			return // unterminated key string
		}
		key := raw[ks : end-1] // bytes between the quotes
		i = skipSpace(raw, end)
		if i >= len(raw) || raw[i] != ':' {
			return
		}
		i++
		vs := skipSpace(raw, i)
		ve := skipValue(raw, vs)
		if keyEquals(key, want) {
			fn(raw[vs:ve])
		}
		i = ve
		i = skipSpace(raw, i)
		if i < len(raw) && raw[i] == ',' {
			i++
		}
	}
}

// unquoteJSONString returns the decoded string content of a JSON string value.
// It reports false if val is not a quoted JSON string. An escape-free string
// takes a zero-allocation fast path (the common case for LSP discriminators);
// a string containing a JSON escape (e.g. "kind") is decoded so the result
// is the true string value, matching what a full JSON decode would produce.
func unquoteJSONString(val []byte) (string, bool) {
	if len(val) < 2 || val[0] != '"' || val[len(val)-1] != '"' {
		return "", false
	}
	body := val[1 : len(val)-1]
	if bytes.IndexByte(body, '\\') < 0 {
		return string(body), true // fast path: no escapes
	}
	dst, err := jsontext.AppendUnquote(nil, val)
	if err != nil {
		return "", false
	}
	return string(dst), true
}

// keyEquals reports whether the raw JSON object-member key bytes (the content
// between the surrounding quotes, as produced by [objectKeys]/[objectMember])
// equal the Go string literal want. An escape-free key compares byte-for-byte
// with no allocation; a key containing a JSON escape is decoded first so that an
// escaped spelling of a member name (valid JSON, e.g. "range" for "range")
// matches its literal, exactly as a map-based decode would.
func keyEquals(key []byte, want string) bool {
	if bytes.IndexByte(key, '\\') < 0 {
		return string(key) == want // fast path: no escapes
	}
	quoted := make([]byte, 0, len(key)+2)
	quoted = append(quoted, '"')
	quoted = append(quoted, key...)
	quoted = append(quoted, '"')
	dst, err := jsontext.AppendUnquote(nil, quoted)
	if err != nil {
		return false
	}
	return string(dst) == want
}

// objectHasKeys reports whether the JSON object raw contains every given
// top-level key. It returns false if raw is not an object.
func objectHasKeys(raw jsontext.Value, keys ...string) bool {
	if len(keys) == 0 {
		// Map-era semantics: objectHasKeys(raw) with no required keys was
		// "objectFields(raw) != nil", i.e. raw is an object.
		return isObject(raw)
	}
	var seen uint64 // bitset over keys (len(keys) is small)
	want := len(keys)
	ok := objectKeys(raw, func(key []byte) bool {
		for j, k := range keys {
			if seen&(1<<uint(j)) == 0 && keyEquals(key, k) {
				seen |= 1 << uint(j)
				want--
				if want == 0 {
					return false // all found; stop early
				}
			}
		}
		return true
	})
	return ok && want == 0
}

// objectHasAndKnown reports, in a single scan, whether the JSON object raw
// contains every key in required AND whether every one of its top-level keys is
// in known. It is the fused form of objectHasKeys(raw, required...) &&
// objectKeysKnown(raw, known...), emitted by the generator's widest-first
// dispatch tier so an arm guard scans the object once instead of twice.
//
// required is assumed to be a subset of known (the generator always passes the
// arm's required keys as required and its full key set as known), so a missing
// required key is also reported via has=false.
func objectHasAndKnown(raw jsontext.Value, required, known []string) (has, allKnown bool) {
	var seen uint64 // bitset over required (small)
	want := len(required)
	allKnown = true
	ok := objectKeys(raw, func(key []byte) bool {
		inKnown := false
		for _, k := range known {
			if keyEquals(key, k) {
				inKnown = true
				break
			}
		}
		if !inKnown {
			allKnown = false
			return false // a foreign key disqualifies the arm; stop early
		}
		for j, r := range required {
			if seen&(1<<uint(j)) == 0 && keyEquals(key, r) {
				seen |= 1 << uint(j)
				want--
			}
		}
		return true
	})
	if !ok {
		return false, false
	}
	return want == 0, allKnown
}

// objectHasAndKnownGuard is the single-boolean arm-guard form of
// objectHasAndKnown: it reports whether raw has every required key AND all its
// keys are known. The generated dispatch code calls this in an if-guard.
func objectHasAndKnownGuard(raw jsontext.Value, required, known []string) bool {
	has, allKnown := objectHasAndKnown(raw, required, known)
	return has && allKnown
}

// objectKeysKnown reports whether every top-level key of the JSON object raw is
// present in the given set of known field names. It is used to ensure a union
// arm is not selected when the payload carries a field that arm cannot hold.
func objectKeysKnown(raw jsontext.Value, known ...string) bool {
	allKnown := true
	ok := objectKeys(raw, func(key []byte) bool {
		for _, k := range known {
			if keyEquals(key, k) {
				return true // known; continue
			}
		}
		allKnown = false
		return false // unknown key found; stop early
	})
	return ok && allKnown
}

// isObject reports whether raw is a JSON object.
func isObject(raw []byte) bool {
	i := skipSpace(raw, 0)
	return i < len(raw) && raw[i] == '{'
}

// arrayFirst invokes fn with the raw bytes of the first element of the JSON
// array raw and reports whether raw is a non-empty array. The element bytes
// alias raw.
func arrayFirst(raw []byte, fn func(elem []byte)) bool {
	i := skipSpace(raw, 0)
	if i >= len(raw) || raw[i] != '[' {
		return false
	}
	i++
	i = skipSpace(raw, i)
	if i >= len(raw) || raw[i] == ']' {
		return false // empty array
	}
	es := i
	ee := skipValue(raw, es)
	fn(raw[es:ee])
	return true
}

// arrayFirstHasKeys reports whether raw is a non-empty JSON array whose first
// element is an object containing every given key.
func arrayFirstHasKeys(raw jsontext.Value, keys ...string) bool {
	var result bool
	present := arrayFirst(raw, func(elem []byte) {
		result = objectHasKeys(elem, keys...)
	})
	return present && result
}

// arrayFirstKeysKnown reports whether raw is a non-empty JSON array whose first
// element is an object all of whose keys are in the given known set.
func arrayFirstKeysKnown(raw jsontext.Value, known ...string) bool {
	var result bool
	present := arrayFirst(raw, func(elem []byte) {
		result = objectKeysKnown(elem, known...)
	})
	return present && result
}

// arrayFirstHasAndKnown is the fused form of arrayFirstHasKeys(raw, required...)
// && arrayFirstKeysKnown(raw, known...): it reports whether raw is a non-empty
// array whose first element is an object that has every required key and whose
// keys are all in known, scanning that element once.
func arrayFirstHasAndKnown(raw jsontext.Value, required, known []string) bool {
	var has, allKnown bool
	present := arrayFirst(raw, func(elem []byte) {
		has, allKnown = objectHasAndKnown(elem, required, known)
	})
	return present && has && allKnown
}
