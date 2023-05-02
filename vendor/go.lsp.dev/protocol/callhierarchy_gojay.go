// SPDX-FileCopyrightText: 2021 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

//go:build gojay
// +build gojay

package protocol

import (
	"github.com/francoispqt/gojay"
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CallHierarchy) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyDynamicRegistration, v.DynamicRegistration)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *CallHierarchy) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *CallHierarchy) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyDynamicRegistration {
		return dec.Bool(&v.DynamicRegistration)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *CallHierarchy) NKeys() int { return 1 }

// compile time check whether the CallHierarchy implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CallHierarchy)(nil)
	_ gojay.UnmarshalerJSONObject = (*CallHierarchy)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CallHierarchyPrepareParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKeyOmitEmpty(keyTextDocument, &v.TextDocument)
	enc.ObjectKeyOmitEmpty(keyPosition, &v.Position)
	encodeProgressToken(enc, keyWorkDoneToken, v.WorkDoneToken)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *CallHierarchyPrepareParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *CallHierarchyPrepareParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyTextDocument:
		return dec.Object(&v.TextDocument)
	case keyPosition:
		return dec.Object(&v.Position)
	case keyWorkDoneToken:
		return decodeProgressToken(dec, k, keyWorkDoneToken, v.WorkDoneToken)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *CallHierarchyPrepareParams) NKeys() int { return 3 }

// compile time check whether the CallHierarchyPrepareParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CallHierarchyPrepareParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*CallHierarchyPrepareParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CallHierarchyItem) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyName, v.Name)
	enc.Float64Key(keyKind, float64(v.Kind))
	enc.ArrayKeyOmitEmpty(keyTags, SymbolTags(v.Tags))
	enc.StringKeyOmitEmpty(keyDetail, v.Detail)
	enc.StringKey(keyURI, string(v.URI))
	enc.ObjectKey(keyRange, &v.Range)
	enc.ObjectKey(keySelectionRange, &v.SelectionRange)
	enc.AddInterfaceKey(keyData, v.Data)
}

// IsNil returns wether the structure is nil value or not.
func (v *CallHierarchyItem) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *CallHierarchyItem) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyName:
		return dec.String(&v.Name)
	case keyKind:
		return dec.Float64((*float64)(&v.Kind))
	case keyTags:
		return dec.Array((*SymbolTags)(&v.Tags))
	case keyDetail:
		return dec.String(&v.Detail)
	case keyURI:
		return dec.String((*string)(&v.URI))
	case keyRange:
		return dec.Object(&v.Range)
	case keySelectionRange:
		return dec.Object(&v.SelectionRange)
	case keyData:
		return dec.Interface(&v.Data)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *CallHierarchyItem) NKeys() int { return 8 }

// compile time check whether the CallHierarchyItem implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CallHierarchyItem)(nil)
	_ gojay.UnmarshalerJSONObject = (*CallHierarchyItem)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CallHierarchyIncomingCallsParams) MarshalJSONObject(enc *gojay.Encoder) {
	encodeProgressToken(enc, keyWorkDoneToken, v.WorkDoneToken)
	encodeProgressToken(enc, keyPartialResultToken, v.PartialResultToken)
	enc.ObjectKey(keyItem, &v.Item)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *CallHierarchyIncomingCallsParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *CallHierarchyIncomingCallsParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyWorkDoneToken:
		return decodeProgressToken(dec, k, keyWorkDoneToken, v.WorkDoneToken)
	case keyPartialResultToken:
		return decodeProgressToken(dec, k, keyPartialResultToken, v.PartialResultToken)
	case keyItem:
		return dec.Object(&v.Item)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *CallHierarchyIncomingCallsParams) NKeys() int { return 3 }

// compile time check whether the CallHierarchyIncomingCallsParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CallHierarchyIncomingCallsParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*CallHierarchyIncomingCallsParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CallHierarchyIncomingCall) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKey(keyFrom, &v.From)
	enc.ArrayKey(keyFromRanges, Ranges(v.FromRanges))
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *CallHierarchyIncomingCall) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *CallHierarchyIncomingCall) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyFrom:
		return dec.Object(&v.From)
	case keyFromRanges:
		return dec.Array((*Ranges)(&v.FromRanges))
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *CallHierarchyIncomingCall) NKeys() int { return 2 }

// compile time check whether the CallHierarchyIncomingCall implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CallHierarchyIncomingCall)(nil)
	_ gojay.UnmarshalerJSONObject = (*CallHierarchyIncomingCall)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CallHierarchyOutgoingCallsParams) MarshalJSONObject(enc *gojay.Encoder) {
	encodeProgressToken(enc, keyWorkDoneToken, v.WorkDoneToken)
	encodeProgressToken(enc, keyPartialResultToken, v.PartialResultToken)
	enc.ObjectKey(keyItem, &v.Item)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *CallHierarchyOutgoingCallsParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *CallHierarchyOutgoingCallsParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyWorkDoneToken:
		return decodeProgressToken(dec, k, keyWorkDoneToken, v.WorkDoneToken)
	case keyPartialResultToken:
		return decodeProgressToken(dec, k, keyPartialResultToken, v.PartialResultToken)
	case keyItem:
		return dec.Object(&v.Item)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *CallHierarchyOutgoingCallsParams) NKeys() int { return 3 }

// compile time check whether the CallHierarchyOutgoingCallsParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CallHierarchyOutgoingCallsParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*CallHierarchyOutgoingCallsParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CallHierarchyOutgoingCall) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKey(keyTo, &v.To)
	enc.ArrayKey(keyFromRanges, Ranges(v.FromRanges))
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *CallHierarchyOutgoingCall) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *CallHierarchyOutgoingCall) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyTo:
		return dec.Object(&v.To)
	case keyFromRanges:
		return dec.Array((*Ranges)(&v.FromRanges))
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *CallHierarchyOutgoingCall) NKeys() int { return 2 }

// compile time check whether the CallHierarchyOutgoingCall implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CallHierarchyOutgoingCall)(nil)
	_ gojay.UnmarshalerJSONObject = (*CallHierarchyOutgoingCall)(nil)
)
