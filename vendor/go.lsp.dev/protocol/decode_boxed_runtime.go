// Copyright 2026 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

import (
	"unsafe"

	"github.com/go-json-experiment/json/jsontext"
)

type ifaceWords struct {
	tab  unsafe.Pointer
	data unsafe.Pointer
}

var (
	progressTokenStringTab = func() unsafe.Pointer {
		var x ProgressToken = String("")
		return (*ifaceWords)(unsafe.Pointer(&x)).tab
	}()
	inlayHintTooltipStringTab = func() unsafe.Pointer {
		var x InlayHintTooltip = String("")
		return (*ifaceWords)(unsafe.Pointer(&x)).tab
	}()
)

// appendBox appends a zero T to a lazily allocated slab and returns the slot
// address. The slab is sized to capN on first use so per-element union arms
// cost one slab allocation per message instead of one heap object per value,
// and an unused arm costs nothing.
func appendBox[T any](boxes *[]T, capN int) *T {
	if *boxes == nil {
		*boxes = make([]T, 0, max(capN, 1))
	}
	var zero T
	*boxes = append(*boxes, zero)
	return &(*boxes)[len(*boxes)-1]
}

// boxedArm constrains tryBoxArm to slab element types whose pointer form
// carries a byte-walker whole-value decoder.
type boxedArm[T any] interface {
	*T
	unmarshalLSPValue(raw jsontext.Value) error
}

// tryBoxArm decodes raw into a fresh slab slot for one object union arm,
// truncating the slab again when the arm does not accept the value so a
// failed probe leaves no trace.
func tryBoxArm[T any, PT boxedArm[T]](raw jsontext.Value, boxes *[]T, capN int) (PT, bool) {
	n := len(*boxes)
	v := PT(appendBox(boxes, capN))
	if v.unmarshalLSPValue(raw) == nil {
		return v, true
	}
	*boxes = (*boxes)[:n]
	return nil, false
}

func boxedStringProgressToken(p *String) ProgressToken {
	var x ProgressToken
	w := (*ifaceWords)(unsafe.Pointer(&x))
	// The slice backing array owns *p for the lifetime of every returned
	// interface value. Installing the cached itab preserves the public dynamic
	// type as String while avoiding one heap box per scalar union arm.
	w.tab = progressTokenStringTab
	w.data = unsafe.Pointer(p)
	return x
}

func boxedStringInlayHintTooltip(p *String) InlayHintTooltip {
	var x InlayHintTooltip
	w := (*ifaceWords)(unsafe.Pointer(&x))
	// The slice backing array owns *p for the lifetime of every returned
	// interface value. Installing the cached itab preserves the public dynamic
	// type as String while avoiding one heap box per scalar union arm.
	w.tab = inlayHintTooltipStringTab
	w.data = unsafe.Pointer(p)
	return x
}

//nolint:gocritic // ptrToRefParam: val is an out-parameter; the boxed union slot is assigned in place.
func unmarshalProgressTokenValueBoxed(raw jsontext.Value, val *ProgressToken, scalarBoxes *[]String, capN int) error {
	switch raw.Kind() {
	case 'n':
		*val = nil
		return dvNullValue(raw)
	case '0':
		v, err := dvScalarInt32(raw)
		if err != nil {
			return err
		}
		*val = Integer(v)
		return nil
	case '"':
		v, err := dvScalarString(raw)
		if err != nil {
			return err
		}
		p := appendBox(scalarBoxes, capN)
		*p = String(v)
		*val = boxedStringProgressToken(p)
		return nil
	}
	return unmarshalProgressTokenValue(raw, val)
}

//nolint:gocritic // ptrToRefParam: val is an out-parameter; the boxed union slot is assigned in place.
func unmarshalInlayHintTooltipValueBoxed(raw jsontext.Value, val *InlayHintTooltip, scalarBoxes *[]String, markupBoxes *[]MarkupContent, capN int) error {
	switch raw.Kind() {
	case 'n':
		*val = nil
		return dvNullValue(raw)
	case '"':
		v, err := dvScalarString(raw)
		if err != nil {
			return err
		}
		p := appendBox(scalarBoxes, capN)
		*p = String(v)
		*val = boxedStringInlayHintTooltip(p)
		return nil
	case '{':
		// MarkupContent is the only object arm, so no discriminating guard is
		// required; a failed decode truncates the slab and falls back.
		n := len(*markupBoxes)
		v := appendBox(markupBoxes, capN)
		if v.unmarshalLSPValue(raw) == nil {
			*val = v
			return nil
		}
		*markupBoxes = (*markupBoxes)[:n]
	}
	return unmarshalInlayHintTooltipValue(raw, val)
}

// unmarshalCompletionItemTextEditValueBoxed decodes the TextEdit |
// InsertReplaceEdit union into per-message slabs, probing arms in the exact
// order of the generated dispatcher (InsertReplaceEdit before TextEdit at
// every tier) so a slice element and a lone field always select the same arm,
// even for spec-violating objects carrying both shapes' keys. The bare-attempt
// tier and every non-object kind delegate to the canonical decoder, which is a
// behavioral superset of this fast path.
//
//nolint:gocritic // ptrToRefParam: val is an out-parameter; the boxed union slot is assigned in place.
func unmarshalCompletionItemTextEditValueBoxed(raw jsontext.Value, val *CompletionItemTextEdit, textEditBoxes *[]TextEdit, insertReplaceBoxes *[]InsertReplaceEdit, capN int) error {
	switch raw.Kind() {
	case '{':
		switch {
		case objectHasAndKnownGuard(raw, []string{"newText", "insert", "replace"}, []string{"newText", "insert", "replace"}):
			if v, ok := tryBoxArm[InsertReplaceEdit](raw, insertReplaceBoxes, capN); ok {
				*val = v
				return nil
			}
		case objectHasAndKnownGuard(raw, []string{"range", "newText"}, []string{"range", "newText"}):
			if v, ok := tryBoxArm[TextEdit](raw, textEditBoxes, capN); ok {
				*val = v
				return nil
			}
		}
		if objectHasKeys(raw, "newText", "insert", "replace") {
			if v, ok := tryBoxArm[InsertReplaceEdit](raw, insertReplaceBoxes, capN); ok {
				*val = v
				return nil
			}
		}
		if objectHasKeys(raw, "range", "newText") {
			if v, ok := tryBoxArm[TextEdit](raw, textEditBoxes, capN); ok {
				*val = v
				return nil
			}
		}
	}
	return unmarshalCompletionItemTextEditValue(raw, val)
}

// unmarshalTextDocumentContentChangeEventValueBoxed decodes the partial |
// whole-document change union into per-message slabs. The partial arm is the
// key superset ({range,text} ⊃ {text}), so it is probed first, mirroring the
// generated dispatcher's superset-wins ordering.
//
//nolint:gocritic // ptrToRefParam: val is an out-parameter; the boxed union slot is assigned in place.
func unmarshalTextDocumentContentChangeEventValueBoxed(raw jsontext.Value, val *TextDocumentContentChangeEvent, partialBoxes *[]TextDocumentContentChangePartial, wholeBoxes *[]TextDocumentContentChangeWholeDocument, capN int) error {
	switch raw.Kind() {
	case '{':
		switch {
		case objectHasAndKnownGuard(raw, []string{"range", "text"}, []string{"range", "rangeLength", "text"}):
			if v, ok := tryBoxArm[TextDocumentContentChangePartial](raw, partialBoxes, capN); ok {
				*val = v
				return nil
			}
		case objectHasAndKnownGuard(raw, []string{"text"}, []string{"text"}):
			if v, ok := tryBoxArm[TextDocumentContentChangeWholeDocument](raw, wholeBoxes, capN); ok {
				*val = v
				return nil
			}
		}
		if objectHasKeys(raw, "range", "text") {
			if v, ok := tryBoxArm[TextDocumentContentChangePartial](raw, partialBoxes, capN); ok {
				*val = v
				return nil
			}
		}
		if objectHasKeys(raw, "text") {
			if v, ok := tryBoxArm[TextDocumentContentChangeWholeDocument](raw, wholeBoxes, capN); ok {
				*val = v
				return nil
			}
		}
	}
	return unmarshalTextDocumentContentChangeEventValue(raw, val)
}

//nolint:gocritic // ptrToRefParam: val is an out-parameter; the boxed union slot is assigned in place.
func unmarshalWorkspaceSymbolLocationValueBoxed(raw jsontext.Value, val *WorkspaceSymbolLocation, locationBoxes *[]Location, locationURIOnlyBoxes *[]LocationUriOnly, capN int) error {
	switch raw.Kind() {
	case 'n':
		*val = nil
		return dvNullValue(raw)
	case '{':
		if objectHasAndKnownGuard(raw, []string{"uri", "range"}, []string{"uri", "range"}) {
			n := len(*locationBoxes)
			v := appendBox(locationBoxes, capN)
			if v.unmarshalLSPValue(raw) == nil {
				*val = v
				return nil
			}
			*locationBoxes = (*locationBoxes)[:n]
		}
		if objectHasAndKnownGuard(raw, []string{"uri"}, []string{"uri"}) {
			n := len(*locationURIOnlyBoxes)
			v := appendBox(locationURIOnlyBoxes, capN)
			if v.unmarshalLSPValue(raw) == nil {
				*val = v
				return nil
			}
			*locationURIOnlyBoxes = (*locationURIOnlyBoxes)[:n]
		}
		if objectHasKeys(raw, "uri", "range") {
			n := len(*locationBoxes)
			v := appendBox(locationBoxes, capN)
			if v.unmarshalLSPValue(raw) == nil {
				*val = v
				return nil
			}
			*locationBoxes = (*locationBoxes)[:n]
		}
		if objectHasKeys(raw, "uri") {
			n := len(*locationURIOnlyBoxes)
			v := appendBox(locationURIOnlyBoxes, capN)
			if v.unmarshalLSPValue(raw) == nil {
				*val = v
				return nil
			}
			*locationURIOnlyBoxes = (*locationURIOnlyBoxes)[:n]
		}
		{
			n := len(*locationBoxes)
			v := appendBox(locationBoxes, capN)
			if v.unmarshalLSPValue(raw) == nil {
				*val = v
				return nil
			}
			*locationBoxes = (*locationBoxes)[:n]
		}
		{
			n := len(*locationURIOnlyBoxes)
			v := appendBox(locationURIOnlyBoxes, capN)
			if v.unmarshalLSPValue(raw) == nil {
				*val = v
				return nil
			}
			*locationURIOnlyBoxes = (*locationURIOnlyBoxes)[:n]
		}
	}
	return unmarshalWorkspaceSymbolLocationValue(raw, val)
}
