// SPDX-FileCopyrightText: 2021 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

//go:build gojay
// +build gojay

package protocol

import (
	"github.com/francoispqt/gojay"
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CancelParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.AddInterfaceKey(keyID, v.ID)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *CancelParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *CancelParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyID {
		return dec.Interface(&v.ID)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *CancelParams) NKeys() int { return 1 }

// compile time check whether the CancelParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CancelParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*CancelParams)(nil)
)

// ProgressToken is the progress token provided by the client or server.
//
// @since 3.15.0.
type ProgressToken interface{}

// NewProgressToken returns a new ProgressToken.
//nolint:gocritic
func NewProgressToken(s string) *ProgressToken {
	var iface interface{} = s
	return (*ProgressToken)(&iface)
}

// NewNumberProgressToken returns a new number ProgressToken.
//nolint:gocritic
func NewNumberProgressToken(n int32) *ProgressToken {
	var iface interface{} = n
	return (*ProgressToken)(&iface)
}

//nolint:gocritic
func encodeProgressToken(enc *gojay.Encoder, key string, v *ProgressToken) {
	if v == nil {
		return
	}
	enc.AddInterfaceKey(key, (interface{})(*v))
}

//nolint:gocritic
func decodeProgressToken(dec *gojay.Decoder, k, key string, v *ProgressToken) error {
	if v == nil || k != key {
		return nil
	}
	return dec.Interface((*interface{})(v))
}

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ProgressParams) MarshalJSONObject(enc *gojay.Encoder) {
	encodeProgressToken(enc, keyToken, &v.Token)
	enc.AddInterfaceKey(keyValue, v.Value)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *ProgressParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *ProgressParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyToken:
		return decodeProgressToken(dec, k, keyToken, &v.Token)
	case keyValue:
		return dec.Interface(&v.Value)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *ProgressParams) NKeys() int { return 2 }

// compile time check whether the ProgressParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ProgressParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*ProgressParams)(nil)
)
