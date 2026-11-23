// Copyright 2026 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

import (
	"strconv"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

// DiagnosticTags stores the common zero-or-one DiagnosticTag case without a
// backing-array allocation. The public representation intentionally differs
// from []DiagnosticTag on hot Diagnostic payloads; callers that need slice
// semantics can use Slice or Set.
type DiagnosticTags struct {
	rest  []DiagnosticTag
	n     int
	first DiagnosticTag
}

// NewDiagnosticTags returns a compact DiagnosticTags value initialized from
// tags.
func NewDiagnosticTags(tags ...DiagnosticTag) DiagnosticTags {
	var out DiagnosticTags
	out.Set(tags)
	return out
}

// IsZero reports whether there are no tags, driving the generated omitzero
// behavior for Diagnostic.tags.
func (t DiagnosticTags) IsZero() bool { return t.n == 0 }

// Len reports the number of diagnostic tags.
func (t DiagnosticTags) Len() int { return t.n }

// Slice returns the diagnostic tags as a new slice owned by the caller.
func (t DiagnosticTags) Slice() []DiagnosticTag {
	switch t.n {
	case 0:
		return nil
	case 1:
		return []DiagnosticTag{t.first}
	default:
		out := make([]DiagnosticTag, t.n)
		out[0] = t.first
		copy(out[1:], t.rest[:t.n-1])
		return out
	}
}

// Set replaces the tags with a copy of tags.
func (t *DiagnosticTags) Set(tags []DiagnosticTag) {
	t.Clear()
	for _, tag := range tags {
		t.append(tag)
	}
}

// Clear removes all diagnostic tags.
func (t *DiagnosticTags) Clear() {
	t.first = 0
	t.rest = nil
	t.n = 0
}

// MarshalJSONTo implements json.MarshalerTo.
func (t DiagnosticTags) MarshalJSONTo(enc *jsontext.Encoder) error {
	return json.MarshalEncode(enc, t.Slice())
}

// UnmarshalJSONFrom implements json.UnmarshalerFrom, decoding in place on the
// caller's decoder instead of constructing a fresh one per field.
func (t *DiagnosticTags) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return decodeDiagnosticTagsFrom(dec, t)
}

func (t *DiagnosticTags) append(tag DiagnosticTag) {
	if t.n == 0 {
		t.first = tag
		t.n = 1
		return
	}
	t.rest = append(t.rest, tag)
	t.n++
}

func decodeStringLikeFrom[T ~string](dec *jsontext.Decoder, out *T) error {
	switch dec.PeekKind() {
	case 'n':
		var zero T
		*out = zero
		_, err := dec.ReadToken()
		return err
	case '"':
		tok, err := dec.ReadToken()
		if err != nil {
			return err
		}
		*out = T(tok.String())
		return nil
	default:
		return json.UnmarshalDecode(dec, out, json.WithUnmarshalers(unionUnmarshalers))
	}
}

func decodeOptionalStringFrom(dec *jsontext.Decoder, out *Optional[string]) error {
	switch dec.PeekKind() {
	case 'n':
		_, err := dec.ReadToken()
		if err != nil {
			return err
		}
		out.Clear()
		return nil
	case '"':
		tok, err := dec.ReadToken()
		if err != nil {
			return err
		}
		out.Set(tok.String())
		return nil
	default:
		return json.UnmarshalDecode(dec, out, json.WithUnmarshalers(unionUnmarshalers))
	}
}

func decodeOptionalBoolFrom(dec *jsontext.Decoder, out *Optional[bool]) error {
	switch dec.PeekKind() {
	case 'n':
		_, err := dec.ReadToken()
		if err != nil {
			return err
		}
		out.Clear()
		return nil
	case 't', 'f':
		tok, err := dec.ReadToken()
		if err != nil {
			return err
		}
		out.Set(tok.Bool())
		return nil
	default:
		return json.UnmarshalDecode(dec, out, json.WithUnmarshalers(unionUnmarshalers))
	}
}

func decodeOptionalInt32From(dec *jsontext.Decoder, out *Optional[int32]) error {
	switch dec.PeekKind() {
	case 'n':
		_, err := dec.ReadToken()
		if err != nil {
			return err
		}
		out.Clear()
		return nil
	case '0':
		var v int32
		if err := decodeInt32From(dec, &v); err != nil {
			return err
		}
		out.Set(v)
		return nil
	default:
		return json.UnmarshalDecode(dec, out, json.WithUnmarshalers(unionUnmarshalers))
	}
}

func decodeInt32From[T ~int32](dec *jsontext.Decoder, out *T) error {
	switch dec.PeekKind() {
	case 'n':
		var zero T
		*out = zero
		_, err := dec.ReadToken()
		return err
	case '0':
		tok, err := dec.ReadToken()
		if err != nil {
			return err
		}
		n, err := tok.Int()
		if err != nil {
			return err
		}
		if n < -1<<31 || n > 1<<31-1 {
			return strconv.ErrRange
		}
		*out = T(n)
		return nil
	default:
		return json.UnmarshalDecode(dec, out, json.WithUnmarshalers(unionUnmarshalers))
	}
}

func decodeUint32From[T ~uint32](dec *jsontext.Decoder, out *T) error {
	switch dec.PeekKind() {
	case 'n':
		var zero T
		*out = zero
		_, err := dec.ReadToken()
		return err
	case '0':
		tok, err := dec.ReadToken()
		if err != nil {
			return err
		}
		n, err := tok.Uint()
		if err != nil {
			return err
		}
		if n > uint64(^uint32(0)) {
			return strconv.ErrRange
		}
		*out = T(n)
		return nil
	default:
		return json.UnmarshalDecode(dec, out, json.WithUnmarshalers(unionUnmarshalers))
	}
}

//nolint:gocritic // ptrToRefParam: out is an out-parameter; the interface slot is assigned in place.
func decodeInlayHintTooltipFrom(dec *jsontext.Decoder, out *InlayHintTooltip) error {
	switch dec.PeekKind() {
	case 'n':
		_, err := dec.ReadToken()
		if err != nil {
			return err
		}
		*out = nil
		return nil
	case '"':
		tok, err := dec.ReadToken()
		if err != nil {
			return err
		}
		*out = String(tok.String())
		return nil
	default:
		return json.UnmarshalDecode(dec, out, json.WithUnmarshalers(unionUnmarshalers))
	}
}

//nolint:gocritic // ptrToRefParam: out is an out-parameter; the interface slot is assigned in place.
func decodeProgressTokenFrom(dec *jsontext.Decoder, out *ProgressToken) error {
	switch dec.PeekKind() {
	case 'n':
		_, err := dec.ReadToken()
		if err != nil {
			return err
		}
		*out = nil
		return nil
	case '"':
		var v String
		if err := decodeStringLikeFrom(dec, &v); err != nil {
			return err
		}
		*out = v
		return nil
	case '0':
		var v Integer
		if err := decodeInt32From(dec, &v); err != nil {
			return err
		}
		*out = v
		return nil
	default:
		return json.UnmarshalDecode(dec, out, json.WithUnmarshalers(unionUnmarshalers))
	}
}

func decodeDiagnosticTagsFrom(dec *jsontext.Decoder, out *DiagnosticTags) error {
	switch dec.PeekKind() {
	case 'n':
		_, err := dec.ReadToken()
		if err != nil {
			return err
		}
		out.Clear()
		return nil
	case '[':
	default:
		var tags []DiagnosticTag
		if err := json.UnmarshalDecode(dec, &tags, json.WithUnmarshalers(unionUnmarshalers)); err != nil {
			return err
		}
		out.Set(tags)
		return nil
	}
	if _, err := dec.ReadToken(); err != nil {
		return err
	}
	out.Clear()
	for dec.PeekKind() != ']' {
		var tag DiagnosticTag
		if err := decodeUint32From(dec, &tag); err != nil {
			return err
		}
		out.append(tag)
	}
	_, err := dec.ReadToken()
	return err
}

func decodePositionFrom(dec *jsontext.Decoder, out *Position) error {
	switch dec.PeekKind() {
	case 'n':
		*out = Position{}
		_, err := dec.ReadToken()
		return err
	case '{':
	default:
		return json.UnmarshalDecode(dec, out, json.WithUnmarshalers(unionUnmarshalers))
	}
	if _, err := dec.ReadToken(); err != nil {
		return err
	}
	for dec.PeekKind() != '}' {
		key, err := dec.ReadToken()
		if err != nil {
			return err
		}
		switch key.String() {
		case "line":
			if err := decodeUint32From(dec, &out.Line); err != nil {
				return err
			}
		case "character":
			if err := decodeUint32From(dec, &out.Character); err != nil {
				return err
			}
		default:
			if err := dec.SkipValue(); err != nil {
				return err
			}
		}
	}
	_, err := dec.ReadToken()
	return err
}

func decodeRangeFrom(dec *jsontext.Decoder, out *Range) error {
	switch dec.PeekKind() {
	case 'n':
		*out = Range{}
		_, err := dec.ReadToken()
		return err
	case '{':
	default:
		return json.UnmarshalDecode(dec, out, json.WithUnmarshalers(unionUnmarshalers))
	}
	if _, err := dec.ReadToken(); err != nil {
		return err
	}
	for dec.PeekKind() != '}' {
		key, err := dec.ReadToken()
		if err != nil {
			return err
		}
		switch key.String() {
		case "start":
			if err := decodePositionFrom(dec, &out.Start); err != nil {
				return err
			}
		case "end":
			if err := decodePositionFrom(dec, &out.End); err != nil {
				return err
			}
		default:
			if err := dec.SkipValue(); err != nil {
				return err
			}
		}
	}
	_, err := dec.ReadToken()
	return err
}

func encodeProgressTokenTo(enc *jsontext.Encoder, x ProgressToken) error {
	switch v := x.(type) {
	case nil:
		return enc.WriteToken(jsontext.Null)
	case String:
		return enc.WriteToken(jsontext.String(string(v)))
	case *String:
		if v == nil {
			return enc.WriteToken(jsontext.Null)
		}
		return enc.WriteToken(jsontext.String(string(*v)))
	case Integer:
		return enc.WriteToken(jsontext.Int(int64(v)))
	default:
		return json.MarshalEncode(enc, x)
	}
}

func encodeDiagnosticTagsTo(enc *jsontext.Encoder, x DiagnosticTags) error {
	if err := enc.WriteToken(jsontext.BeginArray); err != nil {
		return err
	}
	if x.n > 0 {
		if err := enc.WriteToken(jsontext.Uint(uint64(x.first))); err != nil {
			return err
		}
	}
	for _, tag := range x.rest[:max(x.n-1, 0)] {
		if err := enc.WriteToken(jsontext.Uint(uint64(tag))); err != nil {
			return err
		}
	}
	return enc.WriteToken(jsontext.EndArray)
}

func encodeSymbolTagsTo(enc *jsontext.Encoder, x []SymbolTag) error {
	if err := enc.WriteToken(jsontext.BeginArray); err != nil {
		return err
	}
	for _, tag := range x {
		if err := enc.WriteToken(jsontext.Uint(uint64(tag))); err != nil {
			return err
		}
	}
	return enc.WriteToken(jsontext.EndArray)
}

func encodePositionTo(enc *jsontext.Encoder, x Position) error {
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String("line")); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.Uint(uint64(x.Line))); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String("character")); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.Uint(uint64(x.Character))); err != nil {
		return err
	}
	return enc.WriteToken(jsontext.EndObject)
}

func encodeRangeTo(enc *jsontext.Encoder, x Range) error {
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String("start")); err != nil {
		return err
	}
	if err := encodePositionTo(enc, x.Start); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String("end")); err != nil {
		return err
	}
	if err := encodePositionTo(enc, x.End); err != nil {
		return err
	}
	return enc.WriteToken(jsontext.EndObject)
}

func encodeLocationTo(enc *jsontext.Encoder, x Location) error {
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String("uri")); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String(string(x.URI))); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String("range")); err != nil {
		return err
	}
	if err := encodeRangeTo(enc, x.Range); err != nil {
		return err
	}
	return enc.WriteToken(jsontext.EndObject)
}

func encodeLocationURIOnlyTo(enc *jsontext.Encoder, x LocationUriOnly) error {
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String("uri")); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String(string(x.URI))); err != nil {
		return err
	}
	return enc.WriteToken(jsontext.EndObject)
}

func encodeWorkspaceSymbolLocationTo(enc *jsontext.Encoder, x WorkspaceSymbolLocation) error {
	switch v := x.(type) {
	case nil:
		return enc.WriteToken(jsontext.Null)
	case *Location:
		return encodeLocationTo(enc, *v)
	case *LocationUriOnly:
		return encodeLocationURIOnlyTo(enc, *v)
	default:
		return json.MarshalEncode(enc, x)
	}
}

func encodeBaseSymbolInformationFieldsTo(enc *jsontext.Encoder, x BaseSymbolInformation) error {
	if err := enc.WriteToken(jsontext.String("name")); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String(x.Name)); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String("kind")); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.Uint(uint64(x.Kind))); err != nil {
		return err
	}
	if len(x.Tags) > 0 {
		if err := enc.WriteToken(jsontext.String("tags")); err != nil {
			return err
		}
		if err := encodeSymbolTagsTo(enc, x.Tags); err != nil {
			return err
		}
	}
	if x.ContainerName != nil {
		if err := enc.WriteToken(jsontext.String("containerName")); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.String(*x.ContainerName)); err != nil {
			return err
		}
	}
	return nil
}

func encodeWorkspaceSymbolTo(enc *jsontext.Encoder, x *WorkspaceSymbol) error {
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	if err := encodeBaseSymbolInformationFieldsTo(enc, x.BaseSymbolInformation); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String("location")); err != nil {
		return err
	}
	if err := encodeWorkspaceSymbolLocationTo(enc, x.Location); err != nil {
		return err
	}
	if len(x.Data) > 0 {
		if err := enc.WriteToken(jsontext.String("data")); err != nil {
			return err
		}
		if err := enc.WriteValue(x.Data); err != nil {
			return err
		}
	}
	return enc.WriteToken(jsontext.EndObject)
}

func isZeroCommand(v Command) bool {
	return v.Title == "" && v.Tooltip == nil && v.Command == "" && len(v.Arguments) == 0
}
