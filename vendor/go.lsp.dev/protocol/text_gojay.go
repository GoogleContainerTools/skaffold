// SPDX-FileCopyrightText: 2019 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

//go:build gojay
// +build gojay

package protocol

import "github.com/francoispqt/gojay"

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DidOpenTextDocumentParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKey(keyTextDocument, &v.TextDocument)
}

// IsNil returns wether the structure is nil value or not.
func (v *DidOpenTextDocumentParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *DidOpenTextDocumentParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyTextDocument {
		return dec.Object(&v.TextDocument)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DidOpenTextDocumentParams) NKeys() int { return 1 }

// compile time check whether the DidOpenTextDocumentParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DidOpenTextDocumentParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*DidOpenTextDocumentParams)(nil)
)

// TextDocumentContentChangeEvents represents a slice of TextDocumentContentChangeEvent.
type TextDocumentContentChangeEvents []TextDocumentContentChangeEvent

// compile time check whether the TextDocumentContentChangeEvents implements a gojay.MarshalerJSONArray and gojay.UnmarshalerJSONArray interfaces.
var (
	_ gojay.MarshalerJSONArray   = (*TextDocumentContentChangeEvents)(nil)
	_ gojay.UnmarshalerJSONArray = (*TextDocumentContentChangeEvents)(nil)
)

// MarshalJSONArray implements gojay.MarshalerJSONArray.
func (v TextDocumentContentChangeEvents) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range v {
		enc.ObjectOmitEmpty(&v[i])
	}
}

// IsNil implements gojay.MarshalerJSONArray.
func (v TextDocumentContentChangeEvents) IsNil() bool { return len(v) == 0 }

// UnmarshalJSONArray implements gojay.UnmarshalerJSONArray.
func (v *TextDocumentContentChangeEvents) UnmarshalJSONArray(dec *gojay.Decoder) error {
	t := TextDocumentContentChangeEvent{}
	if err := dec.Object(&t); err != nil {
		return err
	}
	*v = append(*v, t)
	return nil
}

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DidChangeTextDocumentParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKey(keyTextDocument, &v.TextDocument)
	enc.ArrayKey(keyContentChanges, (*TextDocumentContentChangeEvents)(&v.ContentChanges))
}

// IsNil returns wether the structure is nil value or not.
func (v *DidChangeTextDocumentParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *DidChangeTextDocumentParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyTextDocument:
		return dec.Object(&v.TextDocument)
	case keyContentChanges:
		return dec.Array((*TextDocumentContentChangeEvents)(&v.ContentChanges))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DidChangeTextDocumentParams) NKeys() int { return 2 }

// compile time check whether the DidChangeTextDocumentParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DidChangeTextDocumentParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*DidChangeTextDocumentParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *TextDocumentContentChangeEvent) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKeyOmitEmpty(keyRange, &v.Range)
	enc.Uint32KeyOmitEmpty(keyRangeLength, v.RangeLength)
	enc.StringKey(keyText, v.Text)
}

// IsNil returns wether the structure is nil value or not.
func (v *TextDocumentContentChangeEvent) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *TextDocumentContentChangeEvent) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyRange:
		return dec.Object(&v.Range)
	case keyRangeLength:
		return dec.Uint32(&v.RangeLength)
	case keyText:
		return dec.String(&v.Text)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *TextDocumentContentChangeEvent) NKeys() int { return 3 }

// compile time check whether the TextDocumentContentChangeEvent implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*TextDocumentContentChangeEvent)(nil)
	_ gojay.UnmarshalerJSONObject = (*TextDocumentContentChangeEvent)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *TextDocumentChangeRegistrationOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKey(keyDocumentSelector, &v.DocumentSelector)
	enc.Float64Key(keySyncKind, float64(v.SyncKind))
}

// IsNil returns wether the structure is nil value or not.
func (v *TextDocumentChangeRegistrationOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *TextDocumentChangeRegistrationOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDocumentSelector:
		return dec.Array(&v.DocumentSelector)
	case keySyncKind:
		return dec.Float64((*float64)(&v.SyncKind))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *TextDocumentChangeRegistrationOptions) NKeys() int { return 2 }

// compile time check whether the TextDocumentChangeRegistrationOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*TextDocumentChangeRegistrationOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*TextDocumentChangeRegistrationOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *WillSaveTextDocumentParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKey(keyTextDocument, &v.TextDocument)
	enc.Float64KeyOmitEmpty(keyReason, float64(v.Reason))
}

// IsNil returns wether the structure is nil value or not.
func (v *WillSaveTextDocumentParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *WillSaveTextDocumentParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyTextDocument:
		return dec.Object(&v.TextDocument)
	case keyReason:
		return dec.Float64((*float64)(&v.Reason))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *WillSaveTextDocumentParams) NKeys() int { return 2 }

// compile time check whether the WillSaveTextDocumentParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*WillSaveTextDocumentParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*WillSaveTextDocumentParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DidSaveTextDocumentParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKeyOmitEmpty(keyText, v.Text)
	enc.ObjectKey(keyTextDocument, &v.TextDocument)
}

// IsNil returns wether the structure is nil value or not.
func (v *DidSaveTextDocumentParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *DidSaveTextDocumentParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyText:
		return dec.String(&v.Text)
	case keyTextDocument:
		return dec.Object(&v.TextDocument)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DidSaveTextDocumentParams) NKeys() int { return 2 }

// compile time check whether the DidSaveTextDocumentParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DidSaveTextDocumentParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*DidSaveTextDocumentParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *TextDocumentSaveRegistrationOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKey(keyDocumentSelector, &v.DocumentSelector)
	enc.BoolKeyOmitEmpty(keyIncludeText, v.IncludeText)
}

// IsNil returns wether the structure is nil value or not.
func (v *TextDocumentSaveRegistrationOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *TextDocumentSaveRegistrationOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDocumentSelector:
		return dec.Array(&v.DocumentSelector)
	case keyIncludeText:
		return dec.Bool(&v.IncludeText)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *TextDocumentSaveRegistrationOptions) NKeys() int { return 2 }

// compile time check whether the TextDocumentSaveRegistrationOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*TextDocumentSaveRegistrationOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*TextDocumentSaveRegistrationOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DidCloseTextDocumentParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKey(keyTextDocument, &v.TextDocument)
}

// IsNil returns wether the structure is nil value or not.
func (v *DidCloseTextDocumentParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *DidCloseTextDocumentParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyTextDocument {
		return dec.Object(&v.TextDocument)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DidCloseTextDocumentParams) NKeys() int { return 1 }

// compile time check whether the DidCloseTextDocumentParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DidCloseTextDocumentParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*DidCloseTextDocumentParams)(nil)
)
