// SPDX-FileCopyrightText: 2019 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

//go:build gojay
// +build gojay

package protocol

import (
	"github.com/francoispqt/gojay"
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *Position) MarshalJSONObject(enc *gojay.Encoder) {
	enc.Uint32Key(keyLine, v.Line)
	enc.Uint32Key(keyCharacter, v.Character)
}

// IsNil returns wether the structure is nil value or not.
func (v *Position) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *Position) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyLine:
		return dec.Uint32(&v.Line)
	case keyCharacter:
		return dec.Uint32(&v.Character)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *Position) NKeys() int { return 2 }

// compile time check whether the Position implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*Position)(nil)
	_ gojay.UnmarshalerJSONObject = (*Position)(nil)
)

// Positions represents a slice of Position.
type Positions []Position

// MarshalJSONArray implements gojay.MarshalerJSONArray.
func (v Positions) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range v {
		enc.Object(&v[i])
	}
}

// IsNil implements gojay.MarshalerJSONArray.
func (v Positions) IsNil() bool { return len(v) == 0 }

// UnmarshalJSONArray implements gojay.UnmarshalerJSONArray.
func (v *Positions) UnmarshalJSONArray(dec *gojay.Decoder) error {
	value := Position{}
	if err := dec.Object(&value); err != nil {
		return err
	}
	*v = append(*v, value)
	return nil
}

// compile time check whether the Positions implements a gojay.MarshalerJSONArray and gojay.UnmarshalerJSONArray interfaces.
var (
	_ gojay.MarshalerJSONArray   = (*Positions)(nil)
	_ gojay.UnmarshalerJSONArray = (*Positions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *Range) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKey(keyStart, &v.Start)
	enc.ObjectKey(keyEnd, &v.End)
}

// IsNil returns wether the structure is nil value or not.
func (v *Range) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *Range) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyStart:
		return dec.Object(&v.Start)
	case keyEnd:
		return dec.Object(&v.End)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *Range) NKeys() int { return 2 }

// compile time check whether the Range implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*Range)(nil)
	_ gojay.UnmarshalerJSONObject = (*Range)(nil)
)

// Ranges represents a slice of Range.
type Ranges []Range

// MarshalJSONArray implements gojay.MarshalerJSONArray.
func (v Ranges) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range v {
		enc.Object(&v[i])
	}
}

// IsNil implements gojay.MarshalerJSONArray.
func (v Ranges) IsNil() bool { return len(v) == 0 }

// UnmarshalJSONArray implements gojay.UnmarshalerJSONArray.
func (v *Ranges) UnmarshalJSONArray(dec *gojay.Decoder) error {
	value := Range{}
	if err := dec.Object(&value); err != nil {
		return err
	}
	*v = append(*v, value)
	return nil
}

// compile time check whether the Ranges implements a gojay.MarshalerJSONArray and gojay.UnmarshalerJSONArray interfaces.
var (
	_ gojay.MarshalerJSONArray   = (*Ranges)(nil)
	_ gojay.UnmarshalerJSONArray = (*Ranges)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *Location) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyURI, string(v.URI))
	enc.ObjectKey(keyRange, &v.Range)
}

// IsNil returns wether the structure is nil value or not.
func (v *Location) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *Location) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyURI:
		return dec.String((*string)(&v.URI))
	case keyRange:
		return dec.Object(&v.Range)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *Location) NKeys() int { return 2 }

// compile time check whether the Location implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*Location)(nil)
	_ gojay.UnmarshalerJSONObject = (*Location)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *LocationLink) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKeyOmitEmpty(keyOriginSelectionRange, v.OriginSelectionRange)
	enc.StringKey(keyTargetURI, string(v.TargetURI))
	enc.ObjectKey(keyTargetRange, &v.TargetRange)
	enc.ObjectKey(keyTargetSelectionRange, &v.TargetSelectionRange)
}

// IsNil returns wether the structure is nil value or not.
func (v *LocationLink) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *LocationLink) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyOriginSelectionRange:
		if v.OriginSelectionRange == nil {
			v.OriginSelectionRange = &Range{}
		}
		return dec.Object(v.OriginSelectionRange)
	case keyTargetURI:
		return dec.String((*string)(&v.TargetURI))
	case keyTargetRange:
		return dec.Object(&v.TargetRange)
	case keyTargetSelectionRange:
		return dec.Object(&v.TargetSelectionRange)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *LocationLink) NKeys() int { return 4 }

// compile time check whether the LocationLink implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*LocationLink)(nil)
	_ gojay.UnmarshalerJSONObject = (*LocationLink)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CodeDescription) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyHref, string(v.Href))
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *CodeDescription) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *CodeDescription) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyHref {
		return dec.String((*string)(&v.Href))
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *CodeDescription) NKeys() int { return 1 }

// compile time check whether the CodeDescription implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CodeDescription)(nil)
	_ gojay.UnmarshalerJSONObject = (*CodeDescription)(nil)
)

// DiagnosticRelatedInformations represents a slice of DiagnosticRelatedInformation.
type DiagnosticRelatedInformations []DiagnosticRelatedInformation

// MarshalJSONArray implements gojay.MarshalerJSONArray.
func (v DiagnosticRelatedInformations) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range v {
		enc.Object(&v[i])
	}
}

// IsNil implements gojay.MarshalerJSONArray.
func (v DiagnosticRelatedInformations) IsNil() bool { return len(v) == 0 }

// UnmarshalJSONArray implements gojay.UnmarshalerJSONArray.
func (v *DiagnosticRelatedInformations) UnmarshalJSONArray(dec *gojay.Decoder) error {
	value := DiagnosticRelatedInformation{}
	if err := dec.Object(&value); err != nil {
		return err
	}
	*v = append(*v, value)
	return nil
}

// compile time check whether the DiagnosticRelatedInformation implements a gojay.MarshalerJSONArray and gojay.UnmarshalerJSONArray interfaces.
var (
	_ gojay.MarshalerJSONArray   = (*DiagnosticRelatedInformations)(nil)
	_ gojay.UnmarshalerJSONArray = (*DiagnosticRelatedInformations)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *Command) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyTitle, v.Title)
	enc.StringKey(keyCommand, v.Command)
	enc.ArrayKeyOmitEmpty(keyArguments, (*Interfaces)(&v.Arguments))
}

// IsNil returns wether the structure is nil value or not.
func (v *Command) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *Command) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyTitle:
		return dec.String(&v.Title)
	case keyCommand:
		return dec.String(&v.Command)
	case keyArguments:
		return dec.Array((*Interfaces)(&v.Arguments))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *Command) NKeys() int { return 3 }

// compile time check whether the Command implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*Command)(nil)
	_ gojay.UnmarshalerJSONObject = (*Command)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *TextEdit) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKey(keyRange, &v.Range)
	enc.StringKey(keyNewText, v.NewText)
}

// IsNil returns wether the structure is nil value or not.
func (v *TextEdit) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *TextEdit) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyRange:
		return dec.Object(&v.Range)
	case keyNewText:
		return dec.String(&v.NewText)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *TextEdit) NKeys() int { return 2 }

// compile time check whether the TextEdit implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*TextEdit)(nil)
	_ gojay.UnmarshalerJSONObject = (*TextEdit)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ChangeAnnotation) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyLabel, v.Label)
	enc.BoolKeyOmitEmpty(keyNeedsConfirmation, v.NeedsConfirmation)
	enc.StringKeyOmitEmpty(keyDescription, v.Description)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *ChangeAnnotation) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *ChangeAnnotation) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyLabel:
		return dec.String(&v.Label)
	case keyNeedsConfirmation:
		return dec.Bool(&v.NeedsConfirmation)
	case keyDescription:
		return dec.String(&v.Description)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *ChangeAnnotation) NKeys() int { return 3 }

// compile time check whether the ChangeAnnotation implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ChangeAnnotation)(nil)
	_ gojay.UnmarshalerJSONObject = (*ChangeAnnotation)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *AnnotatedTextEdit) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKey(keyRange, &v.Range)
	enc.StringKey(keyNewText, v.NewText)
	enc.StringKey(keyAnnotationID, string(v.AnnotationID))
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *AnnotatedTextEdit) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *AnnotatedTextEdit) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyRange:
		return dec.Object(&v.Range)
	case keyNewText:
		return dec.String(&v.NewText)
	case keyAnnotationID:
		return dec.String((*string)(&v.AnnotationID))
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *AnnotatedTextEdit) NKeys() int { return 3 }

// compile time check whether the AnnotatedTextEdit implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*AnnotatedTextEdit)(nil)
	_ gojay.UnmarshalerJSONObject = (*AnnotatedTextEdit)(nil)
)

// TextEdits represents a slice of TextEdit.
type TextEdits []TextEdit

// MarshalJSONArray implements gojay.MarshalerJSONArray.
func (v TextEdits) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range v {
		enc.Object(&v[i])
	}
}

// IsNil returns wether the structure is nil value or not.
func (v TextEdits) IsNil() bool { return len(v) == 0 }

// UnmarshalJSONArray implements gojay.UnmarshalerJSONArray.
func (v *TextEdits) UnmarshalJSONArray(dec *gojay.Decoder) error {
	value := TextEdit{}
	if err := dec.Object(&value); err != nil {
		return err
	}
	*v = append(*v, value)
	return nil
}

// compile time check whether the TextEdits implements a gojay.MarshalerJSONArray and gojay.UnmarshalerJSONArray interfaces.
var (
	_ gojay.MarshalerJSONArray   = (*TextEdits)(nil)
	_ gojay.UnmarshalerJSONArray = (*TextEdits)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *TextDocumentEdit) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKey(keyTextDocument, &v.TextDocument)
	enc.ArrayKey(keyEdits, (*TextEdits)(&v.Edits))
}

// IsNil returns wether the structure is nil value or not.
func (v *TextDocumentEdit) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *TextDocumentEdit) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyTextDocument:
		return dec.Object(&v.TextDocument)
	case keyEdits:
		return dec.Array((*TextEdits)(&v.Edits))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *TextDocumentEdit) NKeys() int { return 2 }

// compile time check whether the TextDocumentEdit implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*TextDocumentEdit)(nil)
	_ gojay.UnmarshalerJSONObject = (*TextDocumentEdit)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CreateFileOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyOverwrite, v.Overwrite)
	enc.BoolKeyOmitEmpty(keyIgnoreIfExists, v.IgnoreIfExists)
}

// IsNil returns wether the structure is nil value or not.
func (v *CreateFileOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *CreateFileOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyOverwrite:
		return dec.Bool(&v.Overwrite)
	case keyIgnoreIfExists:
		return dec.Bool(&v.IgnoreIfExists)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *CreateFileOptions) NKeys() int { return 2 }

// compile time check whether the CreateFileOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CreateFileOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*CreateFileOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CreateFile) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyKind, string(v.Kind))
	enc.StringKey(keyURI, string(v.URI))
	enc.ObjectKeyOmitEmpty(keyOptions, v.Options)
	enc.StringKeyOmitEmpty(keyAnnotationID, string(v.AnnotationID))
}

// IsNil returns wether the structure is nil value or not.
func (v *CreateFile) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *CreateFile) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyKind:
		return dec.String((*string)(&v.Kind))
	case keyURI:
		return dec.String((*string)(&v.URI))
	case keyOptions:
		if v.Options == nil {
			v.Options = &CreateFileOptions{}
		}
		return dec.Object(v.Options)
	case keyAnnotationID:
		return dec.String((*string)(&v.AnnotationID))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *CreateFile) NKeys() int { return 4 }

// compile time check whether the CreateFile implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CreateFile)(nil)
	_ gojay.UnmarshalerJSONObject = (*CreateFile)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *RenameFileOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyOverwrite, v.Overwrite)
	enc.BoolKeyOmitEmpty(keyIgnoreIfExists, v.IgnoreIfExists)
}

// IsNil returns wether the structure is nil value or not.
func (v *RenameFileOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *RenameFileOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyOverwrite:
		return dec.Bool(&v.Overwrite)
	case keyIgnoreIfExists:
		return dec.Bool(&v.IgnoreIfExists)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *RenameFileOptions) NKeys() int { return 2 }

// compile time check whether the RenameFileOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*RenameFileOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*RenameFileOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *RenameFile) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyKind, string(v.Kind))
	enc.StringKey(keyOldURI, string(v.OldURI))
	enc.StringKey(keyNewURI, string(v.NewURI))
	enc.ObjectKeyOmitEmpty(keyOptions, v.Options)
	enc.StringKeyOmitEmpty(keyAnnotationID, string(v.AnnotationID))
}

// IsNil returns wether the structure is nil value or not.
func (v *RenameFile) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *RenameFile) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyKind:
		return dec.String((*string)(&v.Kind))
	case keyOldURI:
		return dec.String((*string)(&v.OldURI))
	case keyNewURI:
		return dec.String((*string)(&v.NewURI))
	case keyOptions:
		if v.Options == nil {
			v.Options = &RenameFileOptions{}
		}
		return dec.Object(v.Options)
	case keyAnnotationID:
		return dec.String((*string)(&v.AnnotationID))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *RenameFile) NKeys() int { return 5 }

// compile time check whether the RenameFile implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*RenameFile)(nil)
	_ gojay.UnmarshalerJSONObject = (*RenameFile)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DeleteFileOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyRecursive, v.Recursive)
	enc.BoolKeyOmitEmpty(keyIgnoreIfNotExists, v.IgnoreIfNotExists)
}

// IsNil returns wether the structure is nil value or not.
func (v *DeleteFileOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *DeleteFileOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyRecursive:
		return dec.Bool(&v.Recursive)
	case keyIgnoreIfNotExists:
		return dec.Bool(&v.IgnoreIfNotExists)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DeleteFileOptions) NKeys() int { return 2 }

// compile time check whether the DeleteFileOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DeleteFileOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*DeleteFileOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DeleteFile) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyKind, string(v.Kind))
	enc.StringKey(keyURI, string(v.URI))
	enc.ObjectKeyOmitEmpty(keyOptions, v.Options)
	enc.StringKeyOmitEmpty(keyAnnotationID, string(v.AnnotationID))
}

// IsNil returns wether the structure is nil value or not.
func (v *DeleteFile) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *DeleteFile) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyKind:
		return dec.String((*string)(&v.Kind))
	case keyURI:
		return dec.String((*string)(&v.URI))
	case keyOptions:
		if v.Options == nil {
			v.Options = &DeleteFileOptions{}
		}
		return dec.Object(v.Options)
	case keyAnnotationID:
		return dec.String((*string)(&v.AnnotationID))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DeleteFile) NKeys() int { return 4 }

// compile time check whether the DeleteFile implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DeleteFile)(nil)
	_ gojay.UnmarshalerJSONObject = (*DeleteFile)(nil)
)

// TextEditsMap represents a map of WorkspaceEdit.Changes.
type TextEditsMap map[DocumentURI][]TextEdit

// compile time check whether the TextEditsMap implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*TextEditsMap)(nil)
	_ gojay.UnmarshalerJSONObject = (*TextEditsMap)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v TextEditsMap) MarshalJSONObject(enc *gojay.Encoder) {
	for key, value := range v {
		value := value
		enc.ArrayKeyOmitEmpty(string(key), (*TextEdits)(&value))
	}
}

// IsNil returns wether the structure is nil value or not.
func (v TextEditsMap) IsNil() bool {
	return v == nil
}

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v TextEditsMap) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	edits := []TextEdit{}
	err := dec.Array((*TextEdits)(&edits))
	if err != nil {
		return err
	}
	v[DocumentURI(k)] = TextEdits(edits)
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v TextEditsMap) NKeys() int { return 0 }

// TextDocumentEdits represents a TextDocumentEdit slice.
type TextDocumentEdits []TextDocumentEdit

// compile time check whether the documentChanges implements a gojay.MarshalerJSONArray and gojay.UnmarshalerJSONArray interfaces.
var (
	_ gojay.MarshalerJSONArray   = (*TextDocumentEdits)(nil)
	_ gojay.UnmarshalerJSONArray = (*TextDocumentEdits)(nil)
)

// MarshalJSONArray implements gojay.MarshalerJSONArray.
func (v TextDocumentEdits) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range v {
		enc.ObjectOmitEmpty(&v[i])
	}
}

// IsNil implements gojay.MarshalerJSONArray.
func (v TextDocumentEdits) IsNil() bool { return len(v) == 0 }

// UnmarshalJSONArray implements gojay.UnmarshalerJSONArray.
func (v *TextDocumentEdits) UnmarshalJSONArray(dec *gojay.Decoder) error {
	t := TextDocumentEdit{}
	if err := dec.Object(&t); err != nil {
		return err
	}
	*v = append(*v, t)
	return nil
}

// ChangeAnnotationsMap represents a map of WorkspaceEdit.ChangeAnnotations.
type ChangeAnnotationsMap map[ChangeAnnotationIdentifier]ChangeAnnotation

// compile time check whether the ChangeAnnotationsMap implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ChangeAnnotationsMap)(nil)
	_ gojay.UnmarshalerJSONObject = (*ChangeAnnotationsMap)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v ChangeAnnotationsMap) MarshalJSONObject(enc *gojay.Encoder) {
	for key, value := range v {
		value := value
		enc.ObjectKeyOmitEmpty(string(key), &value)
	}
}

// IsNil returns wether the structure is nil value or not.
func (v ChangeAnnotationsMap) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v ChangeAnnotationsMap) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	edits := ChangeAnnotation{}
	if err := dec.Object(&edits); err != nil {
		return err
	}
	v[ChangeAnnotationIdentifier(k)] = edits
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v ChangeAnnotationsMap) NKeys() int { return 0 }

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *WorkspaceEdit) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKeyOmitEmpty(keyChanges, (*TextEditsMap)(&v.Changes))
	enc.ArrayKeyOmitEmpty(keyDocumentChanges, (*TextDocumentEdits)(&v.DocumentChanges))
	enc.ObjectKeyOmitEmpty(keyChangeAnnotations, (*ChangeAnnotationsMap)(&v.ChangeAnnotations))
}

// IsNil returns wether the structure is nil value or not.
func (v *WorkspaceEdit) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *WorkspaceEdit) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyChanges:
		if v.Changes == nil {
			v.Changes = make(map[DocumentURI][]TextEdit)
		}
		return dec.Object(TextEditsMap(v.Changes))
	case keyDocumentChanges:
		if v.DocumentChanges == nil {
			v.DocumentChanges = []TextDocumentEdit{}
		}
		return dec.Array((*TextDocumentEdits)(&v.DocumentChanges))
	case keyChangeAnnotations:
		if v.ChangeAnnotations == nil {
			v.ChangeAnnotations = make(map[ChangeAnnotationIdentifier]ChangeAnnotation)
		}
		return dec.Object(ChangeAnnotationsMap(v.ChangeAnnotations))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *WorkspaceEdit) NKeys() int { return 3 }

// compile time check whether the WorkspaceEdit implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*WorkspaceEdit)(nil)
	_ gojay.UnmarshalerJSONObject = (*WorkspaceEdit)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *TextDocumentIdentifier) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyURI, string(v.URI))
}

// IsNil returns wether the structure is nil value or not.
func (v *TextDocumentIdentifier) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *TextDocumentIdentifier) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyURI {
		return dec.String((*string)(&v.URI))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *TextDocumentIdentifier) NKeys() int { return 1 }

// compile time check whether the TextDocumentIdentifier implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*TextDocumentIdentifier)(nil)
	_ gojay.UnmarshalerJSONObject = (*TextDocumentIdentifier)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *TextDocumentItem) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyURI, string(v.URI))
	enc.StringKey(keyLanguageID, string(v.LanguageID))
	enc.Int32Key(keyVersion, v.Version)
	enc.StringKey(keyText, v.Text)
}

// IsNil returns wether the structure is nil value or not.
func (v *TextDocumentItem) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *TextDocumentItem) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyURI:
		return dec.String((*string)(&v.URI))
	case keyLanguageID:
		return dec.String((*string)(&v.LanguageID))
	case keyVersion:
		return dec.Int32(&v.Version)
	case keyText:
		return dec.String(&v.Text)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *TextDocumentItem) NKeys() int { return 4 }

// compile time check whether the TextDocumentItem implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*TextDocumentItem)(nil)
	_ gojay.UnmarshalerJSONObject = (*TextDocumentItem)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *VersionedTextDocumentIdentifier) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyURI, string(v.URI))
	enc.Int32Key(keyVersion, v.Version)
}

// IsNil returns wether the structure is nil value or not.
func (v *VersionedTextDocumentIdentifier) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *VersionedTextDocumentIdentifier) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyURI:
		return dec.String((*string)(&v.URI))
	case keyVersion:
		return dec.Int32(&v.Version)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *VersionedTextDocumentIdentifier) NKeys() int { return 2 }

// compile time check whether the VersionedTextDocumentIdentifier implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*VersionedTextDocumentIdentifier)(nil)
	_ gojay.UnmarshalerJSONObject = (*VersionedTextDocumentIdentifier)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *OptionalVersionedTextDocumentIdentifier) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyURI, string(v.URI))
	if v.Version == nil {
		v.Version = NewVersion(0)
	}
	enc.Int32KeyNullEmpty(keyVersion, *v.Version)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *OptionalVersionedTextDocumentIdentifier) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *OptionalVersionedTextDocumentIdentifier) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyURI:
		return dec.String((*string)(&v.URI))
	case keyVersion:
		return dec.Int32Null(&v.Version)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *OptionalVersionedTextDocumentIdentifier) NKeys() int { return 2 }

// compile time check whether the OptionalVersionedTextDocumentIdentifier implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*OptionalVersionedTextDocumentIdentifier)(nil)
	_ gojay.UnmarshalerJSONObject = (*OptionalVersionedTextDocumentIdentifier)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *TextDocumentPositionParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKey(keyTextDocument, &v.TextDocument)
	enc.ObjectKey(keyPosition, &v.Position)
}

// IsNil returns wether the structure is nil value or not.
func (v *TextDocumentPositionParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *TextDocumentPositionParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyTextDocument:
		return dec.Object(&v.TextDocument)
	case keyPosition:
		return dec.Object(&v.Position)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *TextDocumentPositionParams) NKeys() int { return 2 }

// compile time check whether the TextDocumentPositionParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*TextDocumentPositionParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*TextDocumentPositionParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DocumentFilter) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKeyOmitEmpty(keyLanguage, v.Language)
	enc.StringKeyOmitEmpty(keyScheme, v.Scheme)
	enc.StringKeyOmitEmpty(keyPattern, v.Pattern)
}

// IsNil returns wether the structure is nil value or not.
func (v *DocumentFilter) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *DocumentFilter) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyLanguage:
		return dec.String(&v.Language)
	case keyScheme:
		return dec.String(&v.Scheme)
	case keyPattern:
		return dec.String(&v.Pattern)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DocumentFilter) NKeys() int { return 3 }

// compile time check whether the DocumentFilter implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DocumentFilter)(nil)
	_ gojay.UnmarshalerJSONObject = (*DocumentFilter)(nil)
)

// MarshalJSONArray implements gojay.MarshalerJSONArray.
func (v DocumentSelector) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range v {
		enc.Object(v[i])
	}
}

// IsNil implements gojay.MarshalerJSONArray.
func (v DocumentSelector) IsNil() bool { return len(v) == 0 }

// UnmarshalJSONArray implements gojay.UnmarshalerJSONArray.
func (v *DocumentSelector) UnmarshalJSONArray(dec *gojay.Decoder) error {
	value := &DocumentFilter{}
	if err := dec.Object(value); err != nil {
		return err
	}
	*v = append(*v, value)
	return nil
}

// compile time check whether the DocumentSelector implements a gojay.MarshalerJSONArray and gojay.UnmarshalerJSONArray interfaces.
var (
	_ gojay.MarshalerJSONArray   = (*DocumentSelector)(nil)
	_ gojay.UnmarshalerJSONArray = (*DocumentSelector)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *MarkupContent) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyKind, string(v.Kind))
	enc.StringKey(keyValue, v.Value)
}

// IsNil returns wether the structure is nil value or not.
func (v *MarkupContent) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *MarkupContent) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyKind:
		return dec.String((*string)(&v.Kind))
	case keyValue:
		return dec.String(&v.Value)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *MarkupContent) NKeys() int { return 2 }

// compile time check whether the MarkupContent implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*MarkupContent)(nil)
	_ gojay.UnmarshalerJSONObject = (*MarkupContent)(nil)
)
