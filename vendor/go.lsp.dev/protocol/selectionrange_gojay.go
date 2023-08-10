// SPDX-FileCopyrightText: 2021 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

//go:build gojay
// +build gojay

package protocol

import (
	"github.com/francoispqt/gojay"
)

// SelectionRangeProviderOptions selection range provider options interface.
type SelectionRangeProviderOptions interface {
	Value() interface{}
}

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *SelectionRange) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKey(keyRange, &v.Range)
	enc.ObjectKeyOmitEmpty(keyParent, v.Parent)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *SelectionRange) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *SelectionRange) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyRange:
		return dec.Object(&v.Range)
	case keyParent:
		return dec.Object(v.Parent)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *SelectionRange) NKeys() int { return 2 }

// compile time check whether the SelectionRange implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*SelectionRange)(nil)
	_ gojay.UnmarshalerJSONObject = (*SelectionRange)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *EnableSelectionRange) MarshalJSONObject(enc *gojay.Encoder) {
	enc.Bool(v.Value().(bool))
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *EnableSelectionRange) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *EnableSelectionRange) UnmarshalJSONObject(dec *gojay.Decoder, _ string) error {
	return dec.Bool((*bool)(v))
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *EnableSelectionRange) NKeys() int { return 0 }

// compile time check whether the EnableSelectionRange implements a SelectionRangeProviderOptions, gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ SelectionRangeProviderOptions = (*EnableSelectionRange)(nil)
	_ gojay.MarshalerJSONObject     = (*EnableSelectionRange)(nil)
	_ gojay.UnmarshalerJSONObject   = (*EnableSelectionRange)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *SelectionRangeOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyWorkDoneProgress, v.WorkDoneProgress)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *SelectionRangeOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *SelectionRangeOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyWorkDoneProgress {
		return dec.Bool(&v.WorkDoneProgress)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *SelectionRangeOptions) NKeys() int { return 1 }

// compile time check whether the SelectionRangeOptions implements a SelectionRangeProviderOptions, gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ SelectionRangeProviderOptions = (*SelectionRangeOptions)(nil)
	_ gojay.MarshalerJSONObject     = (*SelectionRangeOptions)(nil)
	_ gojay.UnmarshalerJSONObject   = (*SelectionRangeOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *SelectionRangeRegistrationOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyWorkDoneProgress, v.WorkDoneProgress)
	enc.ArrayKey(keyDocumentSelector, &v.DocumentSelector)
	enc.StringKeyOmitEmpty(keyID, v.ID)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *SelectionRangeRegistrationOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *SelectionRangeRegistrationOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyWorkDoneProgress:
		return dec.Bool(&v.WorkDoneProgress)
	case keyDocumentSelector:
		return dec.Array(&v.DocumentSelector)
	case keyID:
		return dec.String(&v.ID)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *SelectionRangeRegistrationOptions) NKeys() int { return 3 }

// compile time check whether the SelectionRangeRegistrationOptions implements a SelectionRangeProviderOptions, gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ SelectionRangeProviderOptions = (*SelectionRangeRegistrationOptions)(nil)
	_ gojay.MarshalerJSONObject     = (*SelectionRangeRegistrationOptions)(nil)
	_ gojay.UnmarshalerJSONObject   = (*SelectionRangeRegistrationOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *SelectionRangeParams) MarshalJSONObject(enc *gojay.Encoder) {
	encodeProgressToken(enc, keyWorkDoneToken, v.WorkDoneToken)
	encodeProgressToken(enc, keyPartialResultToken, v.PartialResultToken)
	enc.ObjectKey(keyTextDocument, &v.TextDocument)
	enc.ArrayKey(keyPositions, (*Positions)(&v.Positions))
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *SelectionRangeParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *SelectionRangeParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyWorkDoneToken:
		return decodeProgressToken(dec, k, keyWorkDoneToken, v.WorkDoneToken)
	case keyPartialResultToken:
		return decodeProgressToken(dec, k, keyPartialResultToken, v.PartialResultToken)
	case keyTextDocument:
		return dec.Object(&v.TextDocument)
	case keyPositions:
		return dec.Array((*Positions)(&v.Positions))
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *SelectionRangeParams) NKeys() int { return 4 }

// compile time check whether the SelectionRangeParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*SelectionRangeParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*SelectionRangeParams)(nil)
)
