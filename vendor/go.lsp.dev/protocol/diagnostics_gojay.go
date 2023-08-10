// SPDX-FileCopyrightText: 2019 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

//go:build gojay
// +build gojay

package protocol

import (
	"github.com/francoispqt/gojay"
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *Diagnostic) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKey(keyRange, &v.Range)
	enc.Float64KeyOmitEmpty(keySeverity, float64(v.Severity))
	enc.AddInterfaceKeyOmitEmpty(keyCode, v.Code)
	enc.ObjectKeyOmitEmpty(keyCodeDescription, v.CodeDescription)
	enc.StringKeyOmitEmpty(keySource, v.Source)
	enc.StringKey(keyMessage, v.Message)
	enc.ArrayKeyOmitEmpty(keyTags, DiagnosticTags(v.Tags))
	enc.ArrayKeyOmitEmpty(keyRelatedInformation, DiagnosticRelatedInformations(v.RelatedInformation))
	enc.AddInterfaceKeyOmitEmpty(keyData, v.Data)
}

// IsNil returns wether the structure is nil value or not.
func (v *Diagnostic) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *Diagnostic) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyRange:
		return dec.Object(&v.Range)
	case keySeverity:
		return dec.Float64((*float64)(&v.Severity))
	case keyCode:
		return dec.Interface(&v.Code)
	case keyCodeDescription:
		if v.CodeDescription == nil {
			v.CodeDescription = &CodeDescription{}
		}
		return dec.Object(v.CodeDescription)
	case keySource:
		return dec.String(&v.Source)
	case keyMessage:
		return dec.String(&v.Message)
	case keyTags:
		return dec.Array((*DiagnosticTags)(&v.Tags))
	case keyRelatedInformation:
		values := DiagnosticRelatedInformations{}
		err := dec.Array(&values)
		if err == nil && len(values) > 0 {
			v.RelatedInformation = []DiagnosticRelatedInformation(values)
		}
		return err
	case keyData:
		return dec.Interface(&v.Data)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *Diagnostic) NKeys() int { return 9 }

// compile time check whether the Diagnostic implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*Diagnostic)(nil)
	_ gojay.UnmarshalerJSONObject = (*Diagnostic)(nil)
)

// DiagnosticTags represents a slice of DiagnosticTag.
type DiagnosticTags []DiagnosticTag

// MarshalJSONArray implements gojay.MarshalerJSONArray.
func (v DiagnosticTags) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range v {
		enc.Float64(float64(v[i]))
	}
}

// IsNil implements gojay.MarshalerJSONArray.
func (v DiagnosticTags) IsNil() bool { return len(v) == 0 }

// UnmarshalJSONArray implements gojay.UnmarshalerJSONArray.
func (v *DiagnosticTags) UnmarshalJSONArray(dec *gojay.Decoder) error {
	var value DiagnosticTag
	if err := dec.Float64((*float64)(&value)); err != nil {
		return err
	}
	*v = append(*v, value)
	return nil
}

// compile time check whether the CodeActionKinds implements a gojay.MarshalerJSONArray and gojay.UnmarshalerJSONArray interfaces.
var (
	_ gojay.MarshalerJSONArray   = (*DiagnosticTags)(nil)
	_ gojay.UnmarshalerJSONArray = (*DiagnosticTags)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DiagnosticRelatedInformation) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKey(keyLocation, &v.Location)
	enc.StringKey(keyMessage, v.Message)
}

// IsNil returns wether the structure is nil value or not.
func (v *DiagnosticRelatedInformation) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *DiagnosticRelatedInformation) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyLocation:
		return dec.Object(&v.Location)
	case keyMessage:
		return dec.String(&v.Message)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DiagnosticRelatedInformation) NKeys() int { return 2 }

// compile time check whether the DiagnosticRelatedInformation implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DiagnosticRelatedInformation)(nil)
	_ gojay.UnmarshalerJSONObject = (*DiagnosticRelatedInformation)(nil)
)

// Diagnostics represents a slice of Diagnostics.
type Diagnostics []Diagnostic

// compile time check whether the Diagnostics implements a gojay.MarshalerJSONArray and gojay.UnmarshalerJSONArray interfaces.
var (
	_ gojay.MarshalerJSONArray   = (*Diagnostics)(nil)
	_ gojay.UnmarshalerJSONArray = (*Diagnostics)(nil)
)

// MarshalJSONArray implements gojay.MarshalerJSONArray.
func (v Diagnostics) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range v {
		enc.Object(&v[i])
	}
}

// UnmarshalJSONArray implements gojay.UnmarshalerJSONArray.
func (v *Diagnostics) UnmarshalJSONArray(dec *gojay.Decoder) error {
	value := Diagnostic{}
	if err := dec.Object(&value); err != nil {
		return err
	}
	*v = append(*v, value)
	return nil
}

// IsNil implements gojay.MarshalerJSONArray.
func (v Diagnostics) IsNil() bool { return len(v) == 0 }

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *PublishDiagnosticsParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyURI, string(v.URI))
	enc.Uint32KeyOmitEmpty(keyVersion, v.Version)
	enc.ArrayKey(keyDiagnostics, Diagnostics(v.Diagnostics))
}

// IsNil returns wether the structure is nil value or not.
func (v *PublishDiagnosticsParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *PublishDiagnosticsParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyURI:
		return dec.String((*string)(&v.URI))
	case keyVersion:
		return dec.Uint32(&v.Version)
	case keyDiagnostics:
		value := Diagnostics{}
		err := dec.Array(&value)
		if err == nil && len(value) > 0 {
			v.Diagnostics = []Diagnostic(value)
		}
		return err
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *PublishDiagnosticsParams) NKeys() int { return 3 }

// compile time check whether the PublishDiagnosticsParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*PublishDiagnosticsParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*PublishDiagnosticsParams)(nil)
)
