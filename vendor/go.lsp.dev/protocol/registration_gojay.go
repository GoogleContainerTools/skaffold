// SPDX-FileCopyrightText: 2019 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

//go:build gojay
// +build gojay

package protocol

import "github.com/francoispqt/gojay"

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *Registration) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyID, v.ID)
	enc.StringKey(keyMethod, v.Method)

	switch registerOptions := v.RegisterOptions.(type) {
	case []interface{}:
		enc.ArrayKeyOmitEmpty(keyRegisterOptions, Interfaces(registerOptions))
	case map[string]string:
		enc.ObjectKeyOmitEmpty(keyRegisterOptions, StringStringMap(registerOptions))
	case map[string]interface{}:
		enc.ObjectKeyOmitEmpty(keyRegisterOptions, StringInterfaceMap(registerOptions))
	default:
		enc.AddInterfaceKeyOmitEmpty(keyRegisterOptions, registerOptions)
	}
}

// IsNil returns wether the structure is nil value or not.
func (v *Registration) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *Registration) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyID:
		return dec.String(&v.ID)
	case keyMethod:
		return dec.String(&v.Method)
	case keyRegisterOptions:
		return dec.Interface(&v.RegisterOptions)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *Registration) NKeys() int { return 3 }

// compile time check whether the Registration implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*Registration)(nil)
	_ gojay.UnmarshalerJSONObject = (*Registration)(nil)
)

// Registrations represents a slice of Registration.
type Registrations []Registration

// compile time check whether the Registrations implements a gojay.MarshalerJSONArray and gojay.UnmarshalerJSONArray interfaces.
var (
	_ gojay.MarshalerJSONArray   = (*Registrations)(nil)
	_ gojay.UnmarshalerJSONArray = (*Registrations)(nil)
)

// MarshalJSONArray implements gojay.MarshalerJSONArray.
func (v Registrations) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range v {
		enc.ObjectOmitEmpty(&v[i])
	}
}

// IsNil implements gojay.MarshalerJSONArray.
func (v Registrations) IsNil() bool { return len(v) == 0 }

// UnmarshalJSONArray implements gojay.UnmarshalerJSONArray.
func (v *Registrations) UnmarshalJSONArray(dec *gojay.Decoder) error {
	t := Registration{}
	if err := dec.Object(&t); err != nil {
		return err
	}
	*v = append(*v, t)
	return nil
}

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *RegistrationParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKey(keyRegistrations, (*Registrations)(&v.Registrations))
}

// IsNil returns wether the structure is nil value or not.
func (v *RegistrationParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *RegistrationParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyRegistrations {
		return dec.Array((*Registrations)(&v.Registrations))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *RegistrationParams) NKeys() int { return 1 }

// compile time check whether the RegistrationParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*RegistrationParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*RegistrationParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *TextDocumentRegistrationOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKey(keyDocumentSelector, &v.DocumentSelector)
}

// IsNil returns wether the structure is nil value or not.
func (v *TextDocumentRegistrationOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *TextDocumentRegistrationOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyDocumentSelector {
		return dec.Array(&v.DocumentSelector)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *TextDocumentRegistrationOptions) NKeys() int { return 1 }

// compile time check whether the TextDocumentRegistrationOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*TextDocumentRegistrationOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*TextDocumentRegistrationOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *Unregistration) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyID, v.ID)
	enc.StringKey(keyMethod, v.Method)
}

// IsNil returns wether the structure is nil value or not.
func (v *Unregistration) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *Unregistration) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyID:
		return dec.String(&v.ID)
	case keyMethod:
		return dec.String(&v.Method)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *Unregistration) NKeys() int { return 2 }

// compile time check whether the Unregistration implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*Unregistration)(nil)
	_ gojay.UnmarshalerJSONObject = (*Unregistration)(nil)
)

// Unregisterations represents a slice of Unregistration.
type Unregisterations []Unregistration

// compile time check whether the Unregisterations implements a gojay.MarshalerJSONArray and gojay.UnmarshalerJSONArray interfaces.
var (
	_ gojay.MarshalerJSONArray   = (*Unregisterations)(nil)
	_ gojay.UnmarshalerJSONArray = (*Unregisterations)(nil)
)

// MarshalJSONArray implements gojay.MarshalerJSONArray.
func (v Unregisterations) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range v {
		enc.ObjectOmitEmpty(&v[i])
	}
}

// IsNil implements gojay.MarshalerJSONArray.
func (v Unregisterations) IsNil() bool { return len(v) == 0 }

// UnmarshalJSONArray implements gojay.UnmarshalerJSONArray.
func (v *Unregisterations) UnmarshalJSONArray(dec *gojay.Decoder) error {
	t := Unregistration{}
	if err := dec.Object(&t); err != nil {
		return err
	}
	*v = append(*v, t)
	return nil
}

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *UnregistrationParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKey(keyUnregisterations, (*Unregisterations)(&v.Unregisterations))
}

// IsNil returns wether the structure is nil value or not.
func (v *UnregistrationParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *UnregistrationParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyUnregisterations {
		return dec.Array((*Unregisterations)(&v.Unregisterations))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *UnregistrationParams) NKeys() int { return 1 }

// compile time check whether the UnregistrationParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*UnregistrationParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*UnregistrationParams)(nil)
)
