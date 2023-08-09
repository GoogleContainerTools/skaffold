// SPDX-FileCopyrightText: 2019 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

//go:build gojay
// +build gojay

package protocol

import (
	"github.com/francoispqt/gojay"
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CompletionParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKey(keyTextDocument, &v.TextDocument)
	enc.ObjectKey(keyPosition, &v.Position)
	encodeProgressToken(enc, keyWorkDoneToken, v.WorkDoneToken)
	encodeProgressToken(enc, keyPartialResultToken, v.PartialResultToken)
	enc.ObjectKeyOmitEmpty(keyContext, v.Context)
}

// IsNil returns wether the structure is nil value or not.
func (v *CompletionParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *CompletionParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyTextDocument:
		return dec.Object(&v.TextDocument)
	case keyPosition:
		return dec.Object(&v.Position)
	case keyWorkDoneToken:
		return decodeProgressToken(dec, k, keyWorkDoneToken, v.WorkDoneToken)
	case keyPartialResultToken:
		return decodeProgressToken(dec, k, keyPartialResultToken, v.PartialResultToken)
	case keyContext:
		if v.Context == nil {
			v.Context = &CompletionContext{}
		}
		return dec.Object(v.Context)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *CompletionParams) NKeys() int { return 5 }

// compile time check whether the CompletionParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CompletionParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*CompletionParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CompletionContext) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKeyOmitEmpty(keyTriggerCharacter, v.TriggerCharacter)
	enc.Float64Key(keyTriggerKind, float64(v.TriggerKind))
}

// IsNil returns wether the structure is nil value or not.
func (v *CompletionContext) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *CompletionContext) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyTriggerCharacter:
		return dec.String(&v.TriggerCharacter)
	case keyTriggerKind:
		return dec.Float64((*float64)(&v.TriggerKind))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *CompletionContext) NKeys() int { return 2 }

// compile time check whether the CompletionContext implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CompletionContext)(nil)
	_ gojay.UnmarshalerJSONObject = (*CompletionContext)(nil)
)

// CompletionItems represents a slice of CompletionItem.
type CompletionItems []CompletionItem

// compile time check whether the CompletionItems implements a gojay.MarshalerJSONArray and gojay.UnmarshalerJSONArray interfaces.
var (
	_ gojay.MarshalerJSONArray   = (*CompletionItems)(nil)
	_ gojay.UnmarshalerJSONArray = (*CompletionItems)(nil)
)

// MarshalJSONArray implements gojay.MarshalerJSONArray.
func (v CompletionItems) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range v {
		enc.ObjectOmitEmpty(&v[i])
	}
}

// IsNil implements gojay.MarshalerJSONArray.
func (v CompletionItems) IsNil() bool { return len(v) == 0 }

// UnmarshalJSONArray implements gojay.UnmarshalerJSONArray.
func (v *CompletionItems) UnmarshalJSONArray(dec *gojay.Decoder) error {
	t := CompletionItem{}
	if err := dec.Object(&t); err != nil {
		return err
	}
	*v = append(*v, t)
	return nil
}

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CompletionList) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKey(keyIsIncomplete, v.IsIncomplete)
	enc.ArrayKey(keyItems, (*CompletionItems)(&v.Items))
}

// IsNil returns wether the structure is nil value or not.
func (v *CompletionList) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *CompletionList) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyIsIncomplete:
		return dec.Bool(&v.IsIncomplete)
	case keyItems:
		return dec.Array((*CompletionItems)(&v.Items))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *CompletionList) NKeys() int { return 2 }

// compile time check whether the CompletionList implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CompletionList)(nil)
	_ gojay.UnmarshalerJSONObject = (*CompletionList)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *InsertReplaceEdit) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyNewText, v.NewText)
	enc.ObjectKey(keyInsert, &v.Insert)
	enc.ObjectKey(keyReplace, &v.Replace)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *InsertReplaceEdit) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *InsertReplaceEdit) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyNewText:
		return dec.String(&v.NewText)
	case keyInsert:
		return dec.Object(&v.Insert)
	case keyReplace:
		return dec.Object(&v.Replace)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *InsertReplaceEdit) NKeys() int { return 3 }

// compile time check whether the InsertReplaceEdit implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*InsertReplaceEdit)(nil)
	_ gojay.UnmarshalerJSONObject = (*InsertReplaceEdit)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CompletionItem) MarshalJSONObject(enc *gojay.Encoder) {
	enc.AddArrayKeyOmitEmpty(keyAdditionalTextEdits, (*TextEdits)(&v.AdditionalTextEdits))
	enc.ObjectKeyOmitEmpty(keyCommand, v.Command)
	enc.AddArrayKeyOmitEmpty(keyCommitCharacters, (*Strings)(&v.CommitCharacters))
	enc.ArrayKeyOmitEmpty(keyTags, (*CompletionItemTags)(&v.Tags))
	enc.AddInterfaceKeyOmitEmpty(keyData, v.Data)
	enc.BoolKeyOmitEmpty(keyDeprecated, v.Deprecated)
	enc.StringKeyOmitEmpty(keyDetail, v.Detail)
	enc.AddInterfaceKeyOmitEmpty(keyDocumentation, v.Documentation)
	enc.StringKeyOmitEmpty(keyFilterText, v.FilterText)
	enc.StringKeyOmitEmpty(keyInsertText, v.InsertText)
	enc.Float64KeyOmitEmpty(keyInsertTextFormat, float64(v.InsertTextFormat))
	enc.Float64KeyOmitEmpty(keyInsertTextMode, float64(v.InsertTextMode))
	enc.Float64KeyOmitEmpty(keyKind, float64(v.Kind))
	enc.StringKeyOmitEmpty(keyLabel, v.Label)
	enc.BoolKeyOmitEmpty(keyPreselect, v.Preselect)
	enc.StringKeyOmitEmpty(keySortText, v.SortText)
	enc.ObjectKeyOmitEmpty(keyTextEdit, v.TextEdit)
}

// NKeys returns the number of keys to unmarshal.
func (v *CompletionItem) NKeys() int { return 17 }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *CompletionItem) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyAdditionalTextEdits:
		return dec.Array((*TextEdits)(&v.AdditionalTextEdits))
	case keyCommand:
		if v.Command == nil {
			v.Command = &Command{}
		}
		return dec.Object(v.Command)
	case keyCommitCharacters:
		return dec.Array((*Strings)(&v.CommitCharacters))
	case keyTags:
		return dec.Array((*CompletionItemTags)(&v.Tags))
	case keyData:
		return dec.Interface(&v.Data)
	case keyDeprecated:
		return dec.Bool(&v.Deprecated)
	case keyDetail:
		return dec.String(&v.Detail)
	case keyDocumentation:
		return dec.Interface(&v.Documentation)
	case keyFilterText:
		return dec.String(&v.FilterText)
	case keyInsertText:
		return dec.String(&v.InsertText)
	case keyInsertTextFormat:
		return dec.Float64((*float64)(&v.InsertTextFormat))
	case keyInsertTextMode:
		return dec.Float64((*float64)(&v.InsertTextMode))
	case keyKind:
		return dec.Float64((*float64)(&v.Kind))
	case keyLabel:
		return dec.String(&v.Label)
	case keyPreselect:
		return dec.Bool(&v.Preselect)
	case keySortText:
		return dec.String(&v.SortText)
	case keyTextEdit:
		if v.TextEdit == nil {
			v.TextEdit = &TextEdit{}
		}
		return dec.Object(v.TextEdit)
	}
	return nil
}

// IsNil returns wether the structure is nil value or not.
func (v *CompletionItem) IsNil() bool { return v == nil }

// compile time check whether the CompletionItem implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CompletionItem)(nil)
	_ gojay.UnmarshalerJSONObject = (*CompletionItem)(nil)
)

// CompletionItemTags represents a slice of CompletionItemTag.
type CompletionItemTags []CompletionItemTag

// MarshalJSONArray implements gojay.MarshalerJSONArray.
func (v CompletionItemTags) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range v {
		enc.Float64(float64(v[i]))
	}
}

// IsNil implements gojay.MarshalerJSONArray.
func (v CompletionItemTags) IsNil() bool { return len(v) == 0 }

// UnmarshalJSONArray implements gojay.UnmarshalerJSONArray.
func (v *CompletionItemTags) UnmarshalJSONArray(dec *gojay.Decoder) error {
	var value CompletionItemTag
	if err := dec.Float64((*float64)(&value)); err != nil {
		return err
	}
	*v = append(*v, value)
	return nil
}

// compile time check whether the CompletionItemTags implements a gojay.MarshalerJSONArray and gojay.UnmarshalerJSONArray interfaces.
var (
	_ gojay.MarshalerJSONArray   = (*CompletionItemTags)(nil)
	_ gojay.UnmarshalerJSONArray = (*CompletionItemTags)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CompletionRegistrationOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKey(keyDocumentSelector, &v.DocumentSelector)
	enc.AddArrayKeyOmitEmpty(keyTriggerCharacters, (*Strings)(&v.TriggerCharacters))
	enc.BoolKeyOmitEmpty(keyResolveProvider, v.ResolveProvider)
}

// IsNil returns wether the structure is nil value or not.
func (v *CompletionRegistrationOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *CompletionRegistrationOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDocumentSelector:
		return dec.Array(&v.DocumentSelector)
	case keyTriggerCharacters:
		return dec.Array((*Strings)(&v.TriggerCharacters))
	case keyResolveProvider:
		return dec.Bool(&v.ResolveProvider)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *CompletionRegistrationOptions) NKeys() int { return 3 }

// compile time check whether the CompletionRegistrationOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CompletionRegistrationOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*CompletionRegistrationOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *HoverParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKeyOmitEmpty(keyTextDocument, &v.TextDocument)
	enc.ObjectKeyOmitEmpty(keyPosition, &v.Position)
	encodeProgressToken(enc, keyWorkDoneToken, v.WorkDoneToken)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *HoverParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *HoverParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
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
func (v *HoverParams) NKeys() int { return 3 }

// compile time check whether the HoverParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*HoverParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*HoverParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *Hover) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKey(keyContents, &v.Contents)
	enc.ObjectKeyOmitEmpty(keyRange, v.Range)
}

// IsNil returns wether the structure is nil value or not.
func (v *Hover) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *Hover) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyContents:
		return dec.Object(&v.Contents)
	case keyRange:
		if v.Range == nil {
			v.Range = &Range{}
		}
		return dec.Object(v.Range)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *Hover) NKeys() int { return 2 }

// compile time check whether the Hover implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*Hover)(nil)
	_ gojay.UnmarshalerJSONObject = (*Hover)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *SignatureHelpParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKeyOmitEmpty(keyTextDocument, &v.TextDocument)
	enc.ObjectKeyOmitEmpty(keyPosition, &v.Position)
	encodeProgressToken(enc, keyWorkDoneToken, v.WorkDoneToken)
	enc.ObjectKeyOmitEmpty(keyContext, v.Context)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *SignatureHelpParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *SignatureHelpParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyTextDocument:
		return dec.Object(&v.TextDocument)
	case keyPosition:
		return dec.Object(&v.Position)
	case keyWorkDoneToken:
		return decodeProgressToken(dec, k, keyWorkDoneToken, v.WorkDoneToken)
	case keyContext:
		if v.Context == nil {
			v.Context = &SignatureHelpContext{}
		}
		return dec.Object(v.Context)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *SignatureHelpParams) NKeys() int { return 4 }

// compile time check whether the SignatureHelpParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*SignatureHelpParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*SignatureHelpParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *SignatureHelpContext) MarshalJSONObject(enc *gojay.Encoder) {
	enc.Float64Key(keyTriggerKind, float64(v.TriggerKind))
	enc.StringKeyOmitEmpty(keyTriggerCharacter, v.TriggerCharacter)
	enc.BoolKey(keyIsRetrigger, v.IsRetrigger)
	enc.ObjectKeyOmitEmpty(keyActiveSignatureHelp, v.ActiveSignatureHelp)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *SignatureHelpContext) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *SignatureHelpContext) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyTriggerKind:
		return dec.Float64((*float64)(&v.TriggerKind))
	case keyTriggerCharacter:
		return dec.String(&v.TriggerCharacter)
	case keyIsRetrigger:
		return dec.Bool(&v.IsRetrigger)
	case keyActiveSignatureHelp:
		if v.ActiveSignatureHelp == nil {
			v.ActiveSignatureHelp = &SignatureHelp{}
		}
		return dec.Object(v.ActiveSignatureHelp)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *SignatureHelpContext) NKeys() int { return 4 }

// compile time check whether the SignatureHelpContext implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*SignatureHelpContext)(nil)
	_ gojay.UnmarshalerJSONObject = (*SignatureHelpContext)(nil)
)

// SignatureInformations represents a slice of SignatureInformation.
type SignatureInformations []SignatureInformation

// compile time check whether the SignatureInformations implements a gojay.MarshalerJSONArray and gojay.UnmarshalerJSONArray interfaces.
var (
	_ gojay.MarshalerJSONArray   = (*SignatureInformations)(nil)
	_ gojay.UnmarshalerJSONArray = (*SignatureInformations)(nil)
)

// MarshalJSONArray implements gojay.MarshalerJSONArray.
func (v SignatureInformations) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range v {
		enc.ObjectOmitEmpty(&v[i])
	}
}

// IsNil implements gojay.MarshalerJSONArray.
func (v SignatureInformations) IsNil() bool { return len(v) == 0 }

// UnmarshalJSONArray implements gojay.UnmarshalerJSONArray.
func (v *SignatureInformations) UnmarshalJSONArray(dec *gojay.Decoder) error {
	t := SignatureInformation{}
	if err := dec.Object(&t); err != nil {
		return err
	}
	*v = append(*v, t)
	return nil
}

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *SignatureHelp) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKey(keySignatures, (*SignatureInformations)(&v.Signatures))
	enc.Uint32KeyOmitEmpty(keyActiveParameter, v.ActiveParameter)
	enc.Uint32KeyOmitEmpty(keyActiveSignature, v.ActiveSignature)
}

// IsNil returns wether the structure is nil value or not.
func (v *SignatureHelp) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *SignatureHelp) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keySignatures:
		if v.Signatures == nil {
			v.Signatures = []SignatureInformation{}
		}
		return dec.Array((*SignatureInformations)(&v.Signatures))
	case keyActiveParameter:
		return dec.Uint32(&v.ActiveParameter)
	case keyActiveSignature:
		return dec.Uint32(&v.ActiveSignature)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *SignatureHelp) NKeys() int { return 3 }

// compile time check whether the SignatureHelp implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*SignatureHelp)(nil)
	_ gojay.UnmarshalerJSONObject = (*SignatureHelp)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *SignatureInformation) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyLabel, v.Label)
	enc.AddInterfaceKeyOmitEmpty(keyDocumentation, v.Documentation)
	enc.ArrayKeyOmitEmpty(keyParameters, (*ParameterInformations)(&v.Parameters))
	enc.Uint32KeyOmitEmpty(keyActiveParameter, v.ActiveParameter)
}

// IsNil returns wether the structure is nil value or not.
func (v *SignatureInformation) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *SignatureInformation) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyLabel:
		return dec.String(&v.Label)
	case keyDocumentation:
		return dec.Interface(&v.Documentation)
	case keyParameters:
		return dec.Array((*ParameterInformations)(&v.Parameters))
	case keyActiveParameter:
		return dec.Uint32(&v.ActiveParameter)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *SignatureInformation) NKeys() int { return 4 }

// compile time check whether the SignatureInformation implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*SignatureInformation)(nil)
	_ gojay.UnmarshalerJSONObject = (*SignatureInformation)(nil)
)

// ParameterInformations represents a slice of ParameterInformation.
type ParameterInformations []ParameterInformation

// compile time check whether the ParameterInformations implements a gojay.MarshalerJSONArray and gojay.UnmarshalerJSONArray interfaces.
var (
	_ gojay.MarshalerJSONArray   = (*ParameterInformations)(nil)
	_ gojay.UnmarshalerJSONArray = (*ParameterInformations)(nil)
)

// MarshalJSONArray implements gojay.MarshalerJSONArray.
func (v ParameterInformations) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range v {
		enc.ObjectOmitEmpty(&v[i])
	}
}

// IsNil implements gojay.MarshalerJSONArray.
func (v ParameterInformations) IsNil() bool { return len(v) == 0 }

// UnmarshalJSONArray implements gojay.UnmarshalerJSONArray.
func (v *ParameterInformations) UnmarshalJSONArray(dec *gojay.Decoder) error {
	t := ParameterInformation{}
	if err := dec.Object(&t); err != nil {
		return err
	}
	*v = append(*v, t)
	return nil
}

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ParameterInformation) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyLabel, v.Label)
	enc.AddInterfaceKeyOmitEmpty(keyDocumentation, v.Documentation)
}

// IsNil returns wether the structure is nil value or not.
func (v *ParameterInformation) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *ParameterInformation) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyLabel:
		return dec.String(&v.Label)
	case keyDocumentation:
		return dec.Interface(&v.Documentation)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *ParameterInformation) NKeys() int { return 2 }

// compile time check whether the ParameterInformation implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ParameterInformation)(nil)
	_ gojay.UnmarshalerJSONObject = (*ParameterInformation)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *SignatureHelpRegistrationOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKey(keyDocumentSelector, &v.DocumentSelector)
	enc.ArrayKeyOmitEmpty(keyTriggerCharacters, (*Strings)(&v.TriggerCharacters))
}

// IsNil returns wether the structure is nil value or not.
func (v *SignatureHelpRegistrationOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *SignatureHelpRegistrationOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDocumentSelector:
		return dec.Array(&v.DocumentSelector)
	case keyTriggerCharacters:
		return dec.Array((*Strings)(&v.TriggerCharacters))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *SignatureHelpRegistrationOptions) NKeys() int { return 2 }

// compile time check whether the SignatureHelpRegistrationOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*SignatureHelpRegistrationOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*SignatureHelpRegistrationOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ReferenceContext) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKey(keyIncludeDeclaration, v.IncludeDeclaration)
}

// IsNil returns wether the structure is nil value or not.
func (v *ReferenceContext) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *ReferenceContext) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyIncludeDeclaration {
		return dec.Bool(&v.IncludeDeclaration)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *ReferenceContext) NKeys() int { return 1 }

// compile time check whether the ReferenceContext implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ReferenceContext)(nil)
	_ gojay.UnmarshalerJSONObject = (*ReferenceContext)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ReferenceParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKeyOmitEmpty(keyTextDocument, &v.TextDocument)
	enc.ObjectKeyOmitEmpty(keyPosition, &v.Position)
	encodeProgressToken(enc, keyWorkDoneToken, v.WorkDoneToken)
	encodeProgressToken(enc, keyPartialResultToken, v.PartialResultToken)
	enc.ObjectKeyOmitEmpty(keyContext, &v.Context)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *ReferenceParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *ReferenceParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyTextDocument:
		return dec.Object(&v.TextDocument)
	case keyPosition:
		return dec.Object(&v.Position)
	case keyWorkDoneToken:
		return decodeProgressToken(dec, k, keyWorkDoneToken, v.WorkDoneToken)
	case keyPartialResultToken:
		return decodeProgressToken(dec, k, keyPartialResultToken, v.PartialResultToken)
	case keyContext:
		return dec.Object(&v.Context)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *ReferenceParams) NKeys() int { return 5 }

// compile time check whether the ReferenceParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ReferenceParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*ReferenceParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DocumentHighlight) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKey(keyRange, &v.Range)
	enc.Float64KeyOmitEmpty(keyKind, float64(v.Kind))
}

// IsNil returns wether the structure is nil value or not.
func (v *DocumentHighlight) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *DocumentHighlight) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyRange:
		return dec.Object(&v.Range)
	case keyKind:
		return dec.Float64((*float64)(&v.Kind))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DocumentHighlight) NKeys() int { return 2 }

// compile time check whether the DocumentHighlight implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DocumentHighlight)(nil)
	_ gojay.UnmarshalerJSONObject = (*DocumentHighlight)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DocumentSymbolParams) MarshalJSONObject(enc *gojay.Encoder) {
	encodeProgressToken(enc, keyWorkDoneToken, v.WorkDoneToken)
	encodeProgressToken(enc, keyPartialResultToken, v.PartialResultToken)
	enc.ObjectKey(keyTextDocument, &v.TextDocument)
}

// IsNil returns wether the structure is nil value or not.
func (v *DocumentSymbolParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *DocumentSymbolParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
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

// NKeys returns the number of keys to unmarshal.
func (v *DocumentSymbolParams) NKeys() int { return 3 }

// compile time check whether the DocumentSymbolParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DocumentSymbolParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*DocumentSymbolParams)(nil)
)

// DocumentSymbols represents a slice of DocumentSymbol.
type DocumentSymbols []DocumentSymbol

// compile time check whether the DocumentSymbols implements a gojay.MarshalerJSONArray and gojay.UnmarshalerJSONArray interfaces.
var (
	_ gojay.MarshalerJSONArray   = (*DocumentSymbols)(nil)
	_ gojay.UnmarshalerJSONArray = (*DocumentSymbols)(nil)
)

// MarshalJSONArray implements gojay.MarshalerJSONArray.
func (v DocumentSymbols) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range v {
		enc.ObjectOmitEmpty(&v[i])
	}
}

// IsNil implements gojay.MarshalerJSONArray.
func (v DocumentSymbols) IsNil() bool { return len(v) == 0 }

// UnmarshalJSONArray implements gojay.UnmarshalerJSONArray.
func (v *DocumentSymbols) UnmarshalJSONArray(dec *gojay.Decoder) error {
	t := DocumentSymbol{}
	if err := dec.Object(&t); err != nil {
		return err
	}
	*v = append(*v, t)
	return nil
}

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DocumentSymbol) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyName, v.Name)
	enc.StringKeyOmitEmpty(keyDetail, v.Detail)
	enc.Float64Key(keyKind, float64(v.Kind))
	enc.ArrayKeyOmitEmpty(keyTags, (*SymbolTags)(&v.Tags))
	enc.BoolKeyOmitEmpty(keyDeprecated, v.Deprecated)
	enc.ObjectKey(keyRange, &v.Range)
	enc.ObjectKey(keySelectionRange, &v.SelectionRange)
	enc.ArrayKeyOmitEmpty(keyChildren, (*DocumentSymbols)(&v.Children))
}

// IsNil returns wether the structure is nil value or not.
func (v *DocumentSymbol) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *DocumentSymbol) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyName:
		return dec.String(&v.Name)
	case keyDetail:
		return dec.String(&v.Detail)
	case keyKind:
		return dec.Float64((*float64)(&v.Kind))
	case keyTags:
		return dec.Array((*SymbolTags)(&v.Tags))
	case keyDeprecated:
		return dec.Bool(&v.Deprecated)
	case keyRange:
		return dec.Object(&v.Range)
	case keySelectionRange:
		return dec.Object(&v.SelectionRange)
	case keyChildren:
		if v.Children == nil {
			v.Children = []DocumentSymbol{}
		}
		return dec.Array((*DocumentSymbols)(&v.Children))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DocumentSymbol) NKeys() int { return 8 }

// compile time check whether the DocumentSymbol implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DocumentSymbol)(nil)
	_ gojay.UnmarshalerJSONObject = (*DocumentSymbol)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DocumentFormattingParams) MarshalJSONObject(enc *gojay.Encoder) {
	encodeProgressToken(enc, keyWorkDoneToken, v.WorkDoneToken)
	enc.ObjectKey(keyOptions, &v.Options)
	enc.ObjectKey(keyTextDocument, &v.TextDocument)
}

// IsNil returns wether the structure is nil value or not.
func (v *DocumentFormattingParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *DocumentFormattingParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyWorkDoneToken:
		return decodeProgressToken(dec, k, keyWorkDoneToken, v.WorkDoneToken)
	case keyOptions:
		return dec.Object(&v.Options)
	case keyTextDocument:
		return dec.Object(&v.TextDocument)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DocumentFormattingParams) NKeys() int { return 3 }

// compile time check whether the DocumentFormattingParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DocumentFormattingParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*DocumentFormattingParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *SymbolInformation) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyName, v.Name)
	enc.Float64Key(keyKind, float64(v.Kind))
	enc.ArrayKeyOmitEmpty(keyTags, (*SymbolTags)(&v.Tags))
	enc.BoolKeyOmitEmpty(keyDeprecated, v.Deprecated)
	enc.ObjectKey(keyLocation, &v.Location)
	enc.StringKeyOmitEmpty(keyContainerName, v.ContainerName)
}

// IsNil returns wether the structure is nil value or not.
func (v *SymbolInformation) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *SymbolInformation) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyName:
		return dec.String(&v.Name)
	case keyKind:
		return dec.Float64((*float64)(&v.Kind))
	case keyTags:
		return dec.Array((*SymbolTags)(&v.Tags))
	case keyDeprecated:
		return dec.Bool(&v.Deprecated)
	case keyLocation:
		return dec.Object(&v.Location)
	case keyContainerName:
		return dec.String(&v.ContainerName)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *SymbolInformation) NKeys() int { return 6 }

// compile time check whether the SymbolInformation implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*SymbolInformation)(nil)
	_ gojay.UnmarshalerJSONObject = (*SymbolInformation)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CodeActionParams) MarshalJSONObject(enc *gojay.Encoder) {
	encodeProgressToken(enc, keyWorkDoneToken, v.WorkDoneToken)
	encodeProgressToken(enc, keyPartialResultToken, v.PartialResultToken)
	enc.ObjectKey(keyTextDocument, &v.TextDocument)
	enc.ObjectKey(keyContext, &v.Context)
	enc.ObjectKey(keyRange, &v.Range)
}

// IsNil returns wether the structure is nil value or not.
func (v *CodeActionParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *CodeActionParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyWorkDoneToken:
		return decodeProgressToken(dec, k, keyWorkDoneToken, v.WorkDoneToken)
	case keyPartialResultToken:
		return decodeProgressToken(dec, k, keyPartialResultToken, v.PartialResultToken)
	case keyTextDocument:
		return dec.Object(&v.TextDocument)
	case keyContext:
		return dec.Object(&v.Context)
	case keyRange:
		return dec.Object(&v.Range)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *CodeActionParams) NKeys() int { return 5 }

// compile time check whether the CodeActionParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CodeActionParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*CodeActionParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CodeActionContext) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKey(keyDiagnostics, Diagnostics(v.Diagnostics))
	enc.ArrayKey(keyOnly, CodeActionKinds(v.Only))
}

// IsNil returns wether the structure is nil value or not.
func (v *CodeActionContext) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *CodeActionContext) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDiagnostics:
		return dec.Array((*Diagnostics)(&v.Diagnostics))
	case keyOnly:
		return dec.Array((*CodeActionKinds)(&v.Only))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *CodeActionContext) NKeys() int { return 2 }

// compile time check whether the CodeActionContext implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CodeActionContext)(nil)
	_ gojay.UnmarshalerJSONObject = (*CodeActionContext)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CodeAction) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyTitle, v.Title)
	enc.StringKeyOmitEmpty(keyKind, string(v.Kind))
	enc.ArrayKeyOmitEmpty(keyDiagnostics, Diagnostics(v.Diagnostics))
	enc.BoolKeyOmitEmpty(keyIsPreferred, v.IsPreferred)
	enc.ObjectKeyOmitEmpty(keyDisabled, v.Disabled)
	enc.ObjectKeyOmitEmpty(keyEdit, v.Edit)
	enc.ObjectKeyOmitEmpty(keyCommand, v.Command)
	enc.AddInterfaceKeyOmitEmpty(keyData, v.Data)
}

// IsNil returns wether the structure is nil value or not.
func (v *CodeAction) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *CodeAction) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyTitle:
		return dec.String(&v.Title)
	case keyKind:
		return dec.String((*string)(&v.Kind))
	case keyDiagnostics:
		return dec.Array((*Diagnostics)(&v.Diagnostics))
	case keyIsPreferred:
		return dec.Bool(&v.IsPreferred)
	case keyDisabled:
		if v.Disabled == nil {
			v.Disabled = &CodeActionDisable{}
		}
		return dec.Object(v.Disabled)
	case keyEdit:
		if v.Edit == nil {
			v.Edit = &WorkspaceEdit{}
		}
		return dec.Object(v.Edit)
	case keyCommand:
		if v.Command == nil {
			v.Command = &Command{}
		}
		return dec.Object(v.Command)
	case keyData:
		return dec.Interface(&v.Data)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *CodeAction) NKeys() int { return 8 }

// compile time check whether the CodeAction implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CodeAction)(nil)
	_ gojay.UnmarshalerJSONObject = (*CodeAction)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CodeActionDisable) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyReason, v.Reason)
}

// IsNil returns wether the structure is nil value or not.
func (v *CodeActionDisable) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *CodeActionDisable) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyReason {
		return dec.String(&v.Reason)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *CodeActionDisable) NKeys() int { return 1 }

// compile time check whether the CodeActionDisable implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CodeActionDisable)(nil)
	_ gojay.UnmarshalerJSONObject = (*CodeActionDisable)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CodeActionRegistrationOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKey(keyDocumentSelector, &v.DocumentSelector)
	enc.ArrayKeyOmitEmpty(keyCodeActionKinds, CodeActionKinds(v.CodeActionKinds))
}

// IsNil returns wether the structure is nil value or not.
func (v *CodeActionRegistrationOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *CodeActionRegistrationOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDocumentSelector:
		return dec.Array(&v.DocumentSelector)
	case keyCodeActionKinds:
		return dec.Array((*CodeActionKinds)(&v.CodeActionKinds))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *CodeActionRegistrationOptions) NKeys() int { return 2 }

// compile time check whether the CodeActionRegistrationOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CodeActionRegistrationOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*CodeActionRegistrationOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CodeLensParams) MarshalJSONObject(enc *gojay.Encoder) {
	encodeProgressToken(enc, keyWorkDoneToken, v.WorkDoneToken)
	encodeProgressToken(enc, keyPartialResultToken, v.PartialResultToken)
	enc.ObjectKey(keyTextDocument, &v.TextDocument)
}

// IsNil returns wether the structure is nil value or not.
func (v *CodeLensParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *CodeLensParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
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

// NKeys returns the number of keys to unmarshal.
func (v *CodeLensParams) NKeys() int { return 3 }

// compile time check whether the CodeLensParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CodeLensParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*CodeLensParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CodeLens) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKey(keyRange, &v.Range)
	enc.ObjectKeyOmitEmpty(keyCommand, v.Command)
	enc.AddInterfaceKeyOmitEmpty(keyData, v.Data)
}

// IsNil returns wether the structure is nil value or not.
func (v *CodeLens) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *CodeLens) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyRange:
		return dec.Object(&v.Range)
	case keyCommand:
		if v.Command == nil {
			v.Command = &Command{}
		}
		return dec.Object(v.Command)
	case keyData:
		return dec.Interface(&v.Data)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *CodeLens) NKeys() int { return 3 }

// compile time check whether the CodeLens implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CodeLens)(nil)
	_ gojay.UnmarshalerJSONObject = (*CodeLens)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CodeLensRegistrationOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKey(keyDocumentSelector, &v.DocumentSelector)
	enc.BoolKeyOmitEmpty(keyResolveProvider, v.ResolveProvider)
}

// IsNil returns wether the structure is nil value or not.
func (v *CodeLensRegistrationOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *CodeLensRegistrationOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDocumentSelector:
		return dec.Array(&v.DocumentSelector)
	case keyResolveProvider:
		return dec.Bool(&v.ResolveProvider)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *CodeLensRegistrationOptions) NKeys() int { return 2 }

// compile time check whether the CodeLensRegistrationOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CodeLensRegistrationOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*CodeLensRegistrationOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DocumentLinkParams) MarshalJSONObject(enc *gojay.Encoder) {
	encodeProgressToken(enc, keyWorkDoneToken, v.WorkDoneToken)
	encodeProgressToken(enc, keyPartialResultToken, v.PartialResultToken)
	enc.ObjectKey(keyTextDocument, &v.TextDocument)
}

// IsNil returns wether the structure is nil value or not.
func (v *DocumentLinkParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *DocumentLinkParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
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

// NKeys returns the number of keys to unmarshal.
func (v *DocumentLinkParams) NKeys() int { return 3 }

// compile time check whether the DocumentLinkParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DocumentLinkParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*DocumentLinkParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DocumentLink) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKey(keyRange, &v.Range)
	enc.StringKeyOmitEmpty(keyTarget, string(v.Target))
	enc.StringKeyOmitEmpty(keyTooltip, v.Tooltip)
	enc.AddInterfaceKeyOmitEmpty(keyData, v.Data)
}

// IsNil returns wether the structure is nil value or not.
func (v *DocumentLink) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *DocumentLink) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyRange:
		return dec.Object(&v.Range)
	case keyTarget:
		return dec.String((*string)(&v.Target))
	case keyTooltip:
		return dec.String(&v.Tooltip)
	case keyData:
		return dec.Interface(&v.Data)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DocumentLink) NKeys() int { return 4 }

// compile time check whether the DocumentLink implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DocumentLink)(nil)
	_ gojay.UnmarshalerJSONObject = (*DocumentLink)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DocumentColorParams) MarshalJSONObject(enc *gojay.Encoder) {
	encodeProgressToken(enc, keyWorkDoneToken, v.WorkDoneToken)
	encodeProgressToken(enc, keyPartialResultToken, v.PartialResultToken)
	enc.ObjectKey(keyTextDocument, &v.TextDocument)
}

// IsNil returns wether the structure is nil value or not.
func (v *DocumentColorParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *DocumentColorParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
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

// NKeys returns the number of keys to unmarshal.
func (v *DocumentColorParams) NKeys() int { return 3 }

// compile time check whether the DocumentColorParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DocumentColorParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*DocumentColorParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ColorInformation) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKey(keyRange, &v.Range)
	enc.ObjectKey(keyColor, &v.Color)
}

// IsNil returns wether the structure is nil value or not.
func (v *ColorInformation) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *ColorInformation) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyRange:
		return dec.Object(&v.Range)
	case keyColor:
		return dec.Object(&v.Color)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *ColorInformation) NKeys() int { return 2 }

// compile time check whether the ColorInformation implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ColorInformation)(nil)
	_ gojay.UnmarshalerJSONObject = (*ColorInformation)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *Color) MarshalJSONObject(enc *gojay.Encoder) {
	enc.Float64Key(keyAlpha, v.Alpha)
	enc.Float64Key(keyBlue, v.Blue)
	enc.Float64Key(keyGreen, v.Green)
	enc.Float64Key(keyRed, v.Red)
}

// IsNil returns wether the structure is nil value or not.
func (v *Color) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *Color) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyAlpha:
		return dec.Float64(&v.Alpha)
	case keyBlue:
		return dec.Float64(&v.Blue)
	case keyGreen:
		return dec.Float64(&v.Green)
	case keyRed:
		return dec.Float64(&v.Red)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *Color) NKeys() int { return 4 }

// compile time check whether the Color implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*Color)(nil)
	_ gojay.UnmarshalerJSONObject = (*Color)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ColorPresentationParams) MarshalJSONObject(enc *gojay.Encoder) {
	encodeProgressToken(enc, keyWorkDoneToken, v.WorkDoneToken)
	encodeProgressToken(enc, keyPartialResultToken, v.PartialResultToken)
	enc.ObjectKey(keyTextDocument, &v.TextDocument)
	enc.ObjectKey(keyColor, &v.Color)
	enc.ObjectKey(keyRange, &v.Range)
}

// IsNil returns wether the structure is nil value or not.
func (v *ColorPresentationParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *ColorPresentationParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyWorkDoneToken:
		return decodeProgressToken(dec, k, keyWorkDoneToken, v.WorkDoneToken)
	case keyPartialResultToken:
		return decodeProgressToken(dec, k, keyPartialResultToken, v.PartialResultToken)
	case keyTextDocument:
		return dec.Object(&v.TextDocument)
	case keyColor:
		return dec.Object(&v.Color)
	case keyRange:
		return dec.Object(&v.Range)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *ColorPresentationParams) NKeys() int { return 5 }

// compile time check whether the ColorPresentationParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ColorPresentationParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*ColorPresentationParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ColorPresentation) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyLabel, v.Label)
	enc.ObjectKeyOmitEmpty(keyTextEdit, v.TextEdit)
	enc.AddArrayKeyOmitEmpty(keyAdditionalTextEdits, (*TextEdits)(&v.AdditionalTextEdits))
}

// IsNil returns wether the structure is nil value or not.
func (v *ColorPresentation) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *ColorPresentation) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyLabel:
		return dec.String(&v.Label)
	case keyTextEdit:
		if v.TextEdit == nil {
			v.TextEdit = &TextEdit{}
		}
		return dec.Object(v.TextEdit)
	case keyAdditionalTextEdits:
		return dec.Array((*TextEdits)(&v.AdditionalTextEdits))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *ColorPresentation) NKeys() int { return 3 }

// compile time check whether the ColorPresentation implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ColorPresentation)(nil)
	_ gojay.UnmarshalerJSONObject = (*ColorPresentation)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *FormattingOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKey(keyInsertSpaces, v.InsertSpaces)
	enc.Uint32Key(keyTabSize, v.TabSize)
	enc.BoolKeyOmitEmpty(keyTrimTrailingWhitespace, v.TrimTrailingWhitespace)
	enc.BoolKeyOmitEmpty(keyInsertFinalNewline, v.InsertFinalNewline)
	enc.BoolKeyOmitEmpty(keyTrimFinalNewlines, v.TrimFinalNewlines)
	enc.ObjectKeyOmitEmpty(keyKey, (*StringInterfaceMap)(&v.Key))
}

// IsNil returns wether the structure is nil value or not.
func (v *FormattingOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *FormattingOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyInsertSpaces:
		return dec.Bool(&v.InsertSpaces)
	case keyTabSize:
		return dec.Uint32(&v.TabSize)
	case keyTrimTrailingWhitespace:
		return dec.Bool(&v.TrimTrailingWhitespace)
	case keyInsertFinalNewline:
		return dec.Bool(&v.InsertFinalNewline)
	case keyTrimFinalNewlines:
		return dec.Bool(&v.TrimFinalNewlines)
	case keyKey:
		if v.Key == nil {
			v.Key = make(StringInterfaceMap)
		}
		return dec.Object((*StringInterfaceMap)(&v.Key))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *FormattingOptions) NKeys() int { return 6 }

// compile time check whether the FormattingOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*FormattingOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*FormattingOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DocumentRangeFormattingParams) MarshalJSONObject(enc *gojay.Encoder) {
	encodeProgressToken(enc, keyWorkDoneToken, v.WorkDoneToken)
	enc.ObjectKey(keyTextDocument, &v.TextDocument)
	enc.ObjectKey(keyRange, &v.Range)
	enc.ObjectKey(keyOptions, &v.Options)
}

// IsNil returns wether the structure is nil value or not.
func (v *DocumentRangeFormattingParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *DocumentRangeFormattingParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyWorkDoneToken:
		return decodeProgressToken(dec, k, keyWorkDoneToken, v.WorkDoneToken)
	case keyTextDocument:
		return dec.Object(&v.TextDocument)
	case keyRange:
		return dec.Object(&v.Range)
	case keyOptions:
		return dec.Object(&v.Options)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DocumentRangeFormattingParams) NKeys() int { return 4 }

// compile time check whether the DocumentRangeFormattingParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DocumentRangeFormattingParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*DocumentRangeFormattingParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DocumentOnTypeFormattingParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKey(keyTextDocument, &v.TextDocument)
	enc.ObjectKey(keyPosition, &v.Position)
	enc.StringKey(keyCh, v.Ch)
	enc.ObjectKey(keyOptions, &v.Options)
}

// IsNil returns wether the structure is nil value or not.
func (v *DocumentOnTypeFormattingParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *DocumentOnTypeFormattingParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyTextDocument:
		return dec.Object(&v.TextDocument)
	case keyPosition:
		return dec.Object(&v.Position)
	case keyCh:
		return dec.String(&v.Ch)
	case keyOptions:
		return dec.Object(&v.Options)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DocumentOnTypeFormattingParams) NKeys() int { return 4 }

// compile time check whether the DocumentOnTypeFormattingParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DocumentOnTypeFormattingParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*DocumentOnTypeFormattingParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DocumentOnTypeFormattingRegistrationOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKey(keyDocumentSelector, &v.DocumentSelector)
	enc.StringKey(keyFirstTriggerCharacter, v.FirstTriggerCharacter)
	enc.ArrayKeyOmitEmpty(keyMoreTriggerCharacter, (*Strings)(&v.MoreTriggerCharacter))
}

// IsNil returns wether the structure is nil value or not.
func (v *DocumentOnTypeFormattingRegistrationOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *DocumentOnTypeFormattingRegistrationOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDocumentSelector:
		return dec.Array(&v.DocumentSelector)
	case keyFirstTriggerCharacter:
		return dec.String(&v.FirstTriggerCharacter)
	case keyMoreTriggerCharacter:
		return dec.Array((*Strings)(&v.MoreTriggerCharacter))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DocumentOnTypeFormattingRegistrationOptions) NKeys() int { return 3 }

// compile time check whether the DocumentOnTypeFormattingRegistrationOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DocumentOnTypeFormattingRegistrationOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*DocumentOnTypeFormattingRegistrationOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *RenameParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKey(keyTextDocument, &v.TextDocument)
	enc.ObjectKey(keyPosition, &v.Position)
	encodeProgressToken(enc, keyPartialResultToken, v.PartialResultToken)
	enc.StringKey(keyNewName, v.NewName)
}

// IsNil returns wether the structure is nil value or not.
func (v *RenameParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *RenameParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyTextDocument:
		return dec.Object(&v.TextDocument)
	case keyPosition:
		return dec.Object(&v.Position)
	case keyPartialResultToken:
		return decodeProgressToken(dec, k, keyPartialResultToken, v.PartialResultToken)
	case keyNewName:
		return dec.String(&v.NewName)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *RenameParams) NKeys() int { return 4 }

// compile time check whether the RenameParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*RenameParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*RenameParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *RenameRegistrationOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKey(keyDocumentSelector, &v.DocumentSelector)
	enc.BoolKeyOmitEmpty(keyPrepareProvider, v.PrepareProvider)
}

// IsNil returns wether the structure is nil value or not.
func (v *RenameRegistrationOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *RenameRegistrationOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDocumentSelector:
		return dec.Array(&v.DocumentSelector)
	case keyPrepareProvider:
		return dec.Bool(&v.PrepareProvider)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *RenameRegistrationOptions) NKeys() int { return 2 }

// compile time check whether the RenameRegistrationOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*RenameRegistrationOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*RenameRegistrationOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *PrepareRenameParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKey(keyTextDocument, &v.TextDocument)
	enc.ObjectKey(keyPosition, &v.Position)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *PrepareRenameParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *PrepareRenameParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyTextDocument:
		return dec.Object(&v.TextDocument)
	case keyPosition:
		return dec.Object(&v.Position)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *PrepareRenameParams) NKeys() int { return 2 }

// compile time check whether the PrepareRenameParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*PrepareRenameParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*PrepareRenameParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *FoldingRangeParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKey(keyTextDocument, &v.TextDocument)
	enc.ObjectKey(keyPosition, &v.Position)
	encodeProgressToken(enc, keyPartialResultToken, v.PartialResultToken)
}

// IsNil returns wether the structure is nil value or not.
func (v *FoldingRangeParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *FoldingRangeParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyTextDocument:
		return dec.Object(&v.TextDocument)
	case keyPosition:
		return dec.Object(&v.Position)
	case keyPartialResultToken:
		return decodeProgressToken(dec, k, keyPartialResultToken, v.PartialResultToken)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *FoldingRangeParams) NKeys() int { return 3 }

// compile time check whether the FoldingRangeParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*FoldingRangeParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*FoldingRangeParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *FoldingRange) MarshalJSONObject(enc *gojay.Encoder) {
	enc.Uint32Key(keyStartLine, v.StartLine)
	enc.Uint32KeyOmitEmpty(keyStartCharacter, v.StartCharacter)
	enc.Uint32Key(keyEndLine, v.EndLine)
	enc.Uint32KeyOmitEmpty(keyEndCharacter, v.EndCharacter)
	enc.StringKeyOmitEmpty(keyKind, string(v.Kind))
}

// IsNil returns wether the structure is nil value or not.
func (v *FoldingRange) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *FoldingRange) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyStartLine:
		return dec.Uint32(&v.StartLine)
	case keyStartCharacter:
		return dec.Uint32(&v.StartCharacter)
	case keyEndLine:
		return dec.Uint32(&v.EndLine)
	case keyEndCharacter:
		return dec.Uint32(&v.EndCharacter)
	case keyKind:
		return dec.String((*string)(&v.Kind))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *FoldingRange) NKeys() int { return 5 }

// compile time check whether the FoldingRange implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*FoldingRange)(nil)
	_ gojay.UnmarshalerJSONObject = (*FoldingRange)(nil)
)
