// SPDX-FileCopyrightText: 2021 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

//go:build gojay
// +build gojay

package protocol

import (
	"github.com/francoispqt/gojay"
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *WorkDoneProgressBegin) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyKind, string(v.Kind))
	enc.StringKey(keyTitle, v.Title)
	enc.BoolKeyOmitEmpty(keyCancellable, v.Cancellable)
	enc.StringKeyOmitEmpty(keyMessage, v.Message)
	enc.Uint32KeyOmitEmpty(keyPercentage, v.Percentage)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *WorkDoneProgressBegin) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *WorkDoneProgressBegin) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyKind:
		return dec.String((*string)(&v.Kind))
	case keyTitle:
		return dec.String(&v.Title)
	case keyCancellable:
		return dec.Bool(&v.Cancellable)
	case keyMessage:
		return dec.String(&v.Message)
	case keyPercentage:
		return dec.Uint32(&v.Percentage)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *WorkDoneProgressBegin) NKeys() int { return 5 }

// compile time check whether the WorkDoneProgressBegin implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*WorkDoneProgressBegin)(nil)
	_ gojay.UnmarshalerJSONObject = (*WorkDoneProgressBegin)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *WorkDoneProgressReport) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyKind, string(v.Kind))
	enc.BoolKeyOmitEmpty(keyCancellable, v.Cancellable)
	enc.StringKeyOmitEmpty(keyMessage, v.Message)
	enc.Uint32KeyOmitEmpty(keyPercentage, v.Percentage)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *WorkDoneProgressReport) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *WorkDoneProgressReport) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyKind:
		return dec.String((*string)(&v.Kind))
	case keyCancellable:
		return dec.Bool(&v.Cancellable)
	case keyMessage:
		return dec.String(&v.Message)
	case keyPercentage:
		return dec.Uint32(&v.Percentage)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *WorkDoneProgressReport) NKeys() int { return 4 }

// compile time check whether the WorkDoneProgressReport implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*WorkDoneProgressReport)(nil)
	_ gojay.UnmarshalerJSONObject = (*WorkDoneProgressReport)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *WorkDoneProgressEnd) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyKind, string(v.Kind))
	enc.StringKeyOmitEmpty(keyMessage, v.Message)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *WorkDoneProgressEnd) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *WorkDoneProgressEnd) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyKind:
		return dec.String((*string)(&v.Kind))
	case keyMessage:
		return dec.String(&v.Message)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *WorkDoneProgressEnd) NKeys() int { return 2 }

// compile time check whether the WorkDoneProgressEnd implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*WorkDoneProgressEnd)(nil)
	_ gojay.UnmarshalerJSONObject = (*WorkDoneProgressEnd)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *WorkDoneProgressParams) MarshalJSONObject(enc *gojay.Encoder) {
	encodeProgressToken(enc, keyWorkDoneToken, v.WorkDoneToken)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *WorkDoneProgressParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *WorkDoneProgressParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	return decodeProgressToken(dec, k, keyWorkDoneToken, v.WorkDoneToken)
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *WorkDoneProgressParams) NKeys() int { return 1 }

// compile time check whether the WorkDoneProgressParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*WorkDoneProgressParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*WorkDoneProgressParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *PartialResultParams) MarshalJSONObject(enc *gojay.Encoder) {
	encodeProgressToken(enc, keyPartialResultToken, v.PartialResultToken)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *PartialResultParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *PartialResultParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	return decodeProgressToken(dec, k, keyPartialResultToken, v.PartialResultToken)
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *PartialResultParams) NKeys() int { return 1 }

// compile time check whether the PartialResultParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*PartialResultParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*PartialResultParams)(nil)
)
