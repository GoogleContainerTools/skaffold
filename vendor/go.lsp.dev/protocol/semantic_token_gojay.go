// SPDX-FileCopyrightText: 2021 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

//go:build gojay
// +build gojay

package protocol

import (
	"github.com/francoispqt/gojay"
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *SemanticTokensParams) MarshalJSONObject(enc *gojay.Encoder) {
	encodeProgressToken(enc, keyWorkDoneToken, v.WorkDoneToken)
	encodeProgressToken(enc, keyPartialResultToken, v.PartialResultToken)
	enc.ObjectKey(keyTextDocument, &v.TextDocument)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *SemanticTokensParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *SemanticTokensParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyWorkDoneToken:
		return decodeProgressToken(dec, k, keyWorkDoneToken, v.WorkDoneToken)
	case keyPartialResultToken:
		return decodeProgressToken(dec, k, keyPartialResultToken, v.PartialResultToken)
	case keyTextDocument:
		return dec.Object(&v.TextDocument)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *SemanticTokensParams) NKeys() int { return 3 }

// compile time check whether the SemanticTokensParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*SemanticTokensParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*SemanticTokensParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *SemanticTokens) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKeyOmitEmpty(keyResultID, v.ResultID)
	enc.ArrayKey(keyData, Uint32s(v.Data))
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *SemanticTokens) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *SemanticTokens) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyResultID:
		return dec.String(&v.ResultID)
	case keyData:
		return dec.Array((*Uint32s)(&v.Data))
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *SemanticTokens) NKeys() int { return 2 }

// compile time check whether the SemanticTokens implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*SemanticTokens)(nil)
	_ gojay.UnmarshalerJSONObject = (*SemanticTokens)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *SemanticTokensPartialResult) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKey(keyData, Uint32s(v.Data))
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *SemanticTokensPartialResult) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *SemanticTokensPartialResult) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyData {
		return dec.Array((*Uint32s)(&v.Data))
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *SemanticTokensPartialResult) NKeys() int { return 1 }

// compile time check whether the SemanticTokensPartialResult implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*SemanticTokensPartialResult)(nil)
	_ gojay.UnmarshalerJSONObject = (*SemanticTokensPartialResult)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *SemanticTokensDeltaParams) MarshalJSONObject(enc *gojay.Encoder) {
	encodeProgressToken(enc, keyWorkDoneToken, v.WorkDoneToken)
	encodeProgressToken(enc, keyPartialResultToken, v.PartialResultToken)
	enc.ObjectKey(keyTextDocument, &v.TextDocument)
	enc.StringKey(keyPreviousResultID, v.PreviousResultID)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *SemanticTokensDeltaParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *SemanticTokensDeltaParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyWorkDoneToken:
		return decodeProgressToken(dec, k, keyWorkDoneToken, v.WorkDoneToken)
	case keyPartialResultToken:
		return decodeProgressToken(dec, k, keyPartialResultToken, v.PartialResultToken)
	case keyTextDocument:
		return dec.Object(&v.TextDocument)
	case keyPreviousResultID:
		return dec.String(&v.PreviousResultID)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *SemanticTokensDeltaParams) NKeys() int { return 4 }

// compile time check whether the SemanticTokensDeltaParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*SemanticTokensDeltaParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*SemanticTokensDeltaParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *SemanticTokensDelta) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKeyOmitEmpty(keyResultID, v.ResultID)
	enc.ArrayKey(keyEdits, SemanticTokensEdits(v.Edits))
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *SemanticTokensDelta) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *SemanticTokensDelta) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyResultID:
		return dec.String(&v.ResultID)
	case keyData:
		return dec.Array((*SemanticTokensEdits)(&v.Edits))
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *SemanticTokensDelta) NKeys() int { return 2 }

// compile time check whether the SemanticTokensDelta implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*SemanticTokensDelta)(nil)
	_ gojay.UnmarshalerJSONObject = (*SemanticTokensDelta)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *SemanticTokensDeltaPartialResult) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKey(keyEdits, SemanticTokensEdits(v.Edits))
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *SemanticTokensDeltaPartialResult) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *SemanticTokensDeltaPartialResult) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyData {
		return dec.Array((*SemanticTokensEdits)(&v.Edits))
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *SemanticTokensDeltaPartialResult) NKeys() int { return 1 }

// compile time check whether the SemanticTokensDeltaPartialResult implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*SemanticTokensDeltaPartialResult)(nil)
	_ gojay.UnmarshalerJSONObject = (*SemanticTokensDeltaPartialResult)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *SemanticTokensEdit) MarshalJSONObject(enc *gojay.Encoder) {
	enc.Uint32Key(keyStart, v.Start)
	enc.Uint32Key(keyDeleteCount, v.DeleteCount)
	enc.ArrayKey(keyData, Uint32s(v.Data))
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *SemanticTokensEdit) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *SemanticTokensEdit) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyStart:
		return dec.Uint32(&v.Start)
	case keyDeleteCount:
		return dec.Uint32(&v.DeleteCount)
	case keyData:
		return dec.Array((*Uint32s)(&v.Data))
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *SemanticTokensEdit) NKeys() int { return 3 }

// compile time check whether the SemanticTokensEdit implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*SemanticTokensEdit)(nil)
	_ gojay.UnmarshalerJSONObject = (*SemanticTokensEdit)(nil)
)

// SemanticTokensEdits represents a slice of SemanticTokensEdit.
type SemanticTokensEdits []SemanticTokensEdit

// compile time check whether the SemanticTokensEdits implements a gojay.MarshalerJSONArray and gojay.UnmarshalerJSONArray interfaces.
var (
	_ gojay.MarshalerJSONArray   = (*SemanticTokensEdits)(nil)
	_ gojay.UnmarshalerJSONArray = (*SemanticTokensEdits)(nil)
)

// MarshalJSONArray implements gojay.MarshalerJSONArray.
func (v SemanticTokensEdits) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range v {
		enc.Object(&v[i])
	}
}

// IsNil implements gojay.MarshalerJSONArray.
func (v SemanticTokensEdits) IsNil() bool { return len(v) == 0 }

// UnmarshalJSONArray implements gojay.UnmarshalerJSONArray.
func (v *SemanticTokensEdits) UnmarshalJSONArray(dec *gojay.Decoder) error {
	t := SemanticTokensEdit{}
	if err := dec.Object(&t); err != nil {
		return err
	}
	*v = append(*v, t)
	return nil
}

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *SemanticTokensRangeParams) MarshalJSONObject(enc *gojay.Encoder) {
	encodeProgressToken(enc, keyWorkDoneToken, v.WorkDoneToken)
	encodeProgressToken(enc, keyPartialResultToken, v.PartialResultToken)
	enc.ObjectKey(keyTextDocument, &v.TextDocument)
	enc.ObjectKey(keyRange, &v.Range)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *SemanticTokensRangeParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *SemanticTokensRangeParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyWorkDoneToken:
		return decodeProgressToken(dec, k, keyWorkDoneToken, v.WorkDoneToken)
	case keyPartialResultToken:
		return decodeProgressToken(dec, k, keyPartialResultToken, v.PartialResultToken)
	case keyTextDocument:
		return dec.Object(&v.TextDocument)
	case keyRange:
		return dec.Object(&v.Range)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *SemanticTokensRangeParams) NKeys() int { return 4 }

// compile time check whether the SemanticTokensRangeParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*SemanticTokensRangeParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*SemanticTokensRangeParams)(nil)
)
