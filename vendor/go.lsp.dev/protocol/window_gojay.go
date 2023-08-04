// SPDX-FileCopyrightText: 2019 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

//go:build gojay
// +build gojay

package protocol

import (
	"github.com/francoispqt/gojay"
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ShowMessageParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyMessage, v.Message)
	enc.Float64Key(keyType, float64(v.Type))
}

// IsNil returns wether the structure is nil value or not.
func (v *ShowMessageParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *ShowMessageParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyMessage:
		return dec.String(&v.Message)
	case keyType:
		return dec.Float64((*float64)(&v.Type))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *ShowMessageParams) NKeys() int { return 2 }

// compile time check whether the ShowMessageParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ShowMessageParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*ShowMessageParams)(nil)
)

// MessageActionItems represents a slice of MessageActionItem.
type MessageActionItems []MessageActionItem

// compile time check whether the MessageActionItems implements a gojay.MarshalerJSONArray and gojay.UnmarshalerJSONArray interfaces.
var (
	_ gojay.MarshalerJSONArray   = (*MessageActionItems)(nil)
	_ gojay.UnmarshalerJSONArray = (*MessageActionItems)(nil)
)

// MarshalJSONArray implements gojay.MarshalerJSONArray.
func (v MessageActionItems) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range v {
		enc.ObjectOmitEmpty(&v[i])
	}
}

// IsNil implements gojay.MarshalerJSONArray.
func (v MessageActionItems) IsNil() bool { return len(v) == 0 }

// UnmarshalJSONArray implements gojay.UnmarshalerJSONArray.
func (v *MessageActionItems) UnmarshalJSONArray(dec *gojay.Decoder) error {
	t := MessageActionItem{}
	if err := dec.Object(&t); err != nil {
		return err
	}
	*v = append(*v, t)
	return nil
}

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ShowMessageRequestParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKey(keyActions, (*MessageActionItems)(&v.Actions))
	enc.StringKey(keyMessage, v.Message)
	enc.Float64Key(keyType, float64(v.Type))
}

// IsNil returns wether the structure is nil value or not.
func (v *ShowMessageRequestParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *ShowMessageRequestParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyActions:
		return dec.Array((*MessageActionItems)(&v.Actions))
	case keyMessage:
		return dec.String(&v.Message)
	case keyType:
		return dec.Float64((*float64)(&v.Type))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *ShowMessageRequestParams) NKeys() int { return 3 }

// compile time check whether the ShowMessageRequestParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ShowMessageRequestParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*ShowMessageRequestParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *MessageActionItem) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyTitle, v.Title)
}

// IsNil returns wether the structure is nil value or not.
func (v *MessageActionItem) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *MessageActionItem) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyTitle {
		return dec.String(&v.Title)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *MessageActionItem) NKeys() int { return 1 }

// compile time check whether the MessageActionItem implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*MessageActionItem)(nil)
	_ gojay.UnmarshalerJSONObject = (*MessageActionItem)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *LogMessageParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyMessage, v.Message)
	enc.Float64Key(keyType, float64(v.Type))
}

// IsNil returns wether the structure is nil value or not.
func (v *LogMessageParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *LogMessageParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyMessage:
		return dec.String(&v.Message)
	case keyType:
		return dec.Float64((*float64)(&v.Type))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *LogMessageParams) NKeys() int { return 2 }

// compile time check whether the LogMessageParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*LogMessageParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*LogMessageParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *WorkDoneProgressCreateParams) MarshalJSONObject(enc *gojay.Encoder) {
	encodeProgressToken(enc, keyToken, &v.Token)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *WorkDoneProgressCreateParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *WorkDoneProgressCreateParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyToken {
		return dec.Interface((*interface{})(&v.Token))
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *WorkDoneProgressCreateParams) NKeys() int { return 1 }

// compile time check whether the WorkDoneProgressCreateParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*WorkDoneProgressCreateParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*WorkDoneProgressCreateParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *WorkDoneProgressCancelParams) MarshalJSONObject(enc *gojay.Encoder) {
	encodeProgressToken(enc, keyToken, &v.Token)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *WorkDoneProgressCancelParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *WorkDoneProgressCancelParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyToken {
		return dec.Interface((*interface{})(&v.Token))
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *WorkDoneProgressCancelParams) NKeys() int { return 1 }

// compile time check whether the WorkDoneProgressCancelParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*WorkDoneProgressCancelParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*WorkDoneProgressCancelParams)(nil)
)
