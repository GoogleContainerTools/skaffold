// SPDX-FileCopyrightText: 2019 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

//go:build gojay
// +build gojay

package protocol

import (
	"github.com/francoispqt/gojay"
)

// MarshalJSONArray implements gojay.MarshalerJSONArray.
func (v WorkspaceFolders) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range v {
		enc.Object(&v[i])
	}
}

// IsNil implements gojay.MarshalerJSONArray.
func (v WorkspaceFolders) IsNil() bool { return len(v) == 0 }

// UnmarshalJSONArray implements gojay.UnmarshalerJSONArray.
func (v *WorkspaceFolders) UnmarshalJSONArray(dec *gojay.Decoder) error {
	var value WorkspaceFolder
	if err := dec.Object(&value); err != nil {
		return err
	}
	*v = append(*v, value)
	return nil
}

// compile time check whether the WorkspaceFolders implements a gojay.MarshalerJSONArray and gojay.UnmarshalerJSONArray interfaces.
var (
	_ gojay.MarshalerJSONArray   = (*WorkspaceFolders)(nil)
	_ gojay.UnmarshalerJSONArray = (*WorkspaceFolders)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ClientInfo) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyName, v.Name)
	enc.StringKeyOmitEmpty(keyVersion, v.Version)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *ClientInfo) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *ClientInfo) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyName:
		return dec.String(&v.Name)
	case keyVersion:
		return dec.String(&v.Version)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *ClientInfo) NKeys() int { return 2 }

// compile time check whether the ClientInfo implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ClientInfo)(nil)
	_ gojay.UnmarshalerJSONObject = (*ClientInfo)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *InitializeParams) MarshalJSONObject(enc *gojay.Encoder) {
	encodeProgressToken(enc, keyWorkDoneToken, v.WorkDoneToken)
	enc.Int32KeyNullEmpty(keyProcessID, v.ProcessID)
	enc.ObjectKeyOmitEmpty(keyClientInfo, v.ClientInfo)
	enc.StringKeyOmitEmpty(keyLocale, v.Locale)
	enc.StringKeyOmitEmpty(keyRootPath, v.RootPath)
	enc.StringKeyNullEmpty(keyRootURI, string(v.RootURI))
	enc.AddInterfaceKeyOmitEmpty(keyInitializationOptions, v.InitializationOptions)
	enc.ObjectKey(keyCapabilities, &v.Capabilities)
	enc.StringKeyOmitEmpty(keyTrace, string(v.Trace))
	enc.ArrayKeyOmitEmpty(keyWorkspaceFolders, WorkspaceFolders(v.WorkspaceFolders))
}

// IsNil returns wether the structure is nil value or not.
func (v *InitializeParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *InitializeParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyWorkDoneToken:
		return decodeProgressToken(dec, k, keyWorkDoneToken, v.WorkDoneToken)
	case keyProcessID:
		processID := &v.ProcessID
		return dec.Int32Null(&processID)
	case keyClientInfo:
		if v.ClientInfo == nil {
			v.ClientInfo = &ClientInfo{}
		}
		return dec.Object(v.ClientInfo)
	case keyLocale:
		return dec.String(&v.Locale)
	case keyRootPath:
		return dec.String(&v.RootPath)
	case keyRootURI:
		s := (*string)(&v.RootURI)
		return dec.StringNull(&s)
	case keyInitializationOptions:
		return dec.Interface(&v.InitializationOptions)
	case keyCapabilities:
		return dec.Object(&v.Capabilities)
	case keyTrace:
		return dec.String((*string)(&v.Trace))
	case keyWorkspaceFolders:
		return dec.Array((*WorkspaceFolders)(&v.WorkspaceFolders))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *InitializeParams) NKeys() int { return 10 }

// compile time check whether the InitializeParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*InitializeParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*InitializeParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *LogTraceParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyMessage, v.Message)
	enc.StringKeyOmitEmpty(keyVerbose, string(v.Verbose))
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *LogTraceParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *LogTraceParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyMessage:
		return dec.String(&v.Message)
	case keyVerbose:
		return dec.String((*string)(&v.Verbose))
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *LogTraceParams) NKeys() int { return 2 }

// compile time check whether the LogTraceParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*LogTraceParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*LogTraceParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *SetTraceParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyValue, string(v.Value))
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *SetTraceParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *SetTraceParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyValue {
		return dec.String((*string)(&v.Value))
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *SetTraceParams) NKeys() int { return 1 }

// compile time check whether the SetTraceParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*SetTraceParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*SetTraceParams)(nil)
)

// FileOperationFilters represents a slice of FileOperationFilter.
type FileOperationFilters []FileOperationFilter

// compile time check whether the FileOperationFilters implements a gojay.MarshalerJSONArray and gojay.UnmarshalerJSONArray interfaces.
var (
	_ gojay.MarshalerJSONArray   = (*FileOperationFilters)(nil)
	_ gojay.UnmarshalerJSONArray = (*FileOperationFilters)(nil)
)

// MarshalJSONArray implements gojay.MarshalerJSONArray.
func (v FileOperationFilters) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range v {
		enc.Object(&v[i])
	}
}

// IsNil implements gojay.MarshalerJSONArray.
func (v FileOperationFilters) IsNil() bool { return len(v) == 0 }

// UnmarshalJSONArray implements gojay.UnmarshalerJSONArray.
func (v *FileOperationFilters) UnmarshalJSONArray(dec *gojay.Decoder) error {
	var value FileOperationFilter
	if err := dec.Object(&value); err != nil {
		return err
	}
	*v = append(*v, value)
	return nil
}

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *FileOperationPatternOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyIgnoreCase, v.IgnoreCase)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *FileOperationPatternOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *FileOperationPatternOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyIgnoreCase {
		return dec.Bool(&v.IgnoreCase)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *FileOperationPatternOptions) NKeys() int { return 1 }

// compile time check whether the FileOperationPatternOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*FileOperationPatternOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*FileOperationPatternOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *FileOperationPattern) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyGlob, v.Glob)
	enc.StringKeyOmitEmpty(keyMatches, string(v.Matches))
	enc.ObjectKeyOmitEmpty(keyOptions, &v.Options)
}

// IsNil returns wether the structure is nil value or not.
func (v *FileOperationPattern) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *FileOperationPattern) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyGlob:
		return dec.String(&v.Glob)
	case keyMatches:
		return dec.String((*string)(&v.Matches))
	case keyOptions:
		return dec.Object(&v.Options)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *FileOperationPattern) NKeys() int { return 3 }

// compile time check whether the FileOperationPattern implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*FileOperationPattern)(nil)
	_ gojay.UnmarshalerJSONObject = (*FileOperationPattern)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *FileOperationFilter) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKeyOmitEmpty(keyScheme, v.Scheme)
	enc.ObjectKey(keyPattern, &v.Pattern)
}

// IsNil returns wether the structure is nil value or not.
func (v *FileOperationFilter) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *FileOperationFilter) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyScheme:
		return dec.String(&v.Scheme)
	case keyPattern:
		return dec.Object(&v.Pattern)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *FileOperationFilter) NKeys() int { return 2 }

// compile time check whether the FileOperationFilter implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*FileOperationFilter)(nil)
	_ gojay.UnmarshalerJSONObject = (*FileOperationFilter)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CreateFilesParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKey(keyFiles, FileCreates(v.Files))
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *CreateFilesParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *CreateFilesParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyFiles {
		return dec.Array((*FileCreates)(&v.Files))
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *CreateFilesParams) NKeys() int { return 1 }

// compile time check whether the CreateFilesParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CreateFilesParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*CreateFilesParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *FileCreate) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyURI, v.URI)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *FileCreate) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *FileCreate) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyURI {
		return dec.String(&v.URI)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *FileCreate) NKeys() int { return 1 }

// compile time check whether the FileCreate implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*FileCreate)(nil)
	_ gojay.UnmarshalerJSONObject = (*FileCreate)(nil)
)

// FileCreates represents a slice of FileCreate.
type FileCreates []FileCreate

// compile time check whether the FileCreates implements a gojay.MarshalerJSONArray and gojay.UnmarshalerJSONArray interfaces.
var (
	_ gojay.MarshalerJSONArray   = (*FileCreates)(nil)
	_ gojay.UnmarshalerJSONArray = (*FileCreates)(nil)
)

// MarshalJSONArray implements gojay.MarshalerJSONArray.
func (v FileCreates) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range v {
		enc.Object(&v[i])
	}
}

// IsNil implements gojay.MarshalerJSONArray.
func (v FileCreates) IsNil() bool { return len(v) == 0 }

// UnmarshalJSONArray implements gojay.UnmarshalerJSONArray.
func (v *FileCreates) UnmarshalJSONArray(dec *gojay.Decoder) error {
	var value FileCreate
	if err := dec.Object(&value); err != nil {
		return err
	}
	*v = append(*v, value)
	return nil
}

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *RenameFilesParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKey(keyFiles, FileRenames(v.Files))
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *RenameFilesParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *RenameFilesParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyFiles {
		return dec.Array((*FileRenames)(&v.Files))
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *RenameFilesParams) NKeys() int { return 1 }

// compile time check whether the RenameFilesParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*RenameFilesParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*RenameFilesParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *FileRename) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyOldURI, v.OldURI)
	enc.StringKey(keyNewURI, v.NewURI)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *FileRename) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *FileRename) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyOldURI:
		return dec.String(&v.OldURI)
	case keyNewURI:
		return dec.String(&v.NewURI)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *FileRename) NKeys() int { return 2 }

// compile time check whether the FileRename implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*FileRename)(nil)
	_ gojay.UnmarshalerJSONObject = (*FileRename)(nil)
)

// FileRenames represents a slice of FileRename.
type FileRenames []FileRename

// compile time check whether the FileRenames implements a gojay.MarshalerJSONArray and gojay.UnmarshalerJSONArray interfaces.
var (
	_ gojay.MarshalerJSONArray   = (*FileRenames)(nil)
	_ gojay.UnmarshalerJSONArray = (*FileRenames)(nil)
)

// MarshalJSONArray implements gojay.MarshalerJSONArray.
func (v FileRenames) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range v {
		enc.Object(&v[i])
	}
}

// IsNil implements gojay.MarshalerJSONArray.
func (v FileRenames) IsNil() bool { return len(v) == 0 }

// UnmarshalJSONArray implements gojay.UnmarshalerJSONArray.
func (v *FileRenames) UnmarshalJSONArray(dec *gojay.Decoder) error {
	var value FileRename
	if err := dec.Object(&value); err != nil {
		return err
	}
	*v = append(*v, value)
	return nil
}

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DeleteFilesParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKey(keyFiles, FileDeletes(v.Files))
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *DeleteFilesParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *DeleteFilesParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyFiles {
		return dec.Array((*FileDeletes)(&v.Files))
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *DeleteFilesParams) NKeys() int { return 1 }

// compile time check whether the DeleteFilesParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DeleteFilesParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*DeleteFilesParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *FileDelete) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyURI, v.URI)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *FileDelete) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *FileDelete) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyURI {
		return dec.String(&v.URI)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *FileDelete) NKeys() int { return 1 }

// compile time check whether the FileDelete implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*FileDelete)(nil)
	_ gojay.UnmarshalerJSONObject = (*FileDelete)(nil)
)

// FileDeletes represents a slice of FileDelete.
type FileDeletes []FileDelete

// compile time check whether the FileDeletes implements a gojay.MarshalerJSONArray and gojay.UnmarshalerJSONArray interfaces.
var (
	_ gojay.MarshalerJSONArray   = (*FileDeletes)(nil)
	_ gojay.UnmarshalerJSONArray = (*FileDeletes)(nil)
)

// MarshalJSONArray implements gojay.MarshalerJSONArray.
func (v FileDeletes) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range v {
		enc.Object(&v[i])
	}
}

// IsNil implements gojay.MarshalerJSONArray.
func (v FileDeletes) IsNil() bool { return len(v) == 0 }

// UnmarshalJSONArray implements gojay.UnmarshalerJSONArray.
func (v *FileDeletes) UnmarshalJSONArray(dec *gojay.Decoder) error {
	var value FileDelete
	if err := dec.Object(&value); err != nil {
		return err
	}
	*v = append(*v, value)
	return nil
}

// CompletionItemKinds represents a slice of CompletionItemKind.
type CompletionItemKinds []CompletionItemKind

// compile time check whether the CompletionItemKinds implements a gojay.MarshalerJSONArray and gojay.UnmarshalerJSONArray interfaces.
var (
	_ gojay.MarshalerJSONArray   = (*CompletionItemKinds)(nil)
	_ gojay.UnmarshalerJSONArray = (*CompletionItemKinds)(nil)
)

// MarshalJSONArray implements gojay.MarshalerJSONArray.
func (v CompletionItemKinds) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range v {
		enc.Float64(float64(v[i]))
	}
}

// IsNil implements gojay.MarshalerJSONArray.
func (v CompletionItemKinds) IsNil() bool { return len(v) == 0 }

// UnmarshalJSONArray implements gojay.UnmarshalerJSONArray.
func (v *CompletionItemKinds) UnmarshalJSONArray(dec *gojay.Decoder) error {
	var value CompletionItemKind
	if err := dec.Float64((*float64)(&value)); err != nil {
		return err
	}
	*v = append(*v, value)
	return nil
}

// InsertTextModes represents a slice of InsertTextMode.
type InsertTextModes []InsertTextMode

// compile time check whether the InsertTextModes implements a gojay.MarshalerJSONArray and gojay.UnmarshalerJSONArray interfaces.
var (
	_ gojay.MarshalerJSONArray   = (*InsertTextModes)(nil)
	_ gojay.UnmarshalerJSONArray = (*InsertTextModes)(nil)
)

// MarshalJSONArray implements gojay.MarshalerJSONArray.
func (v InsertTextModes) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range v {
		enc.Float64(float64(v[i]))
	}
}

// IsNil implements gojay.MarshalerJSONArray.
func (v InsertTextModes) IsNil() bool { return len(v) == 0 }

// UnmarshalJSONArray implements gojay.UnmarshalerJSONArray.
func (v *InsertTextModes) UnmarshalJSONArray(dec *gojay.Decoder) error {
	var value InsertTextMode
	if err := dec.Float64((*float64)(&value)); err != nil {
		return err
	}
	*v = append(*v, value)
	return nil
}

// MarkupKinds represents a slice of MarkupKind.
type MarkupKinds []MarkupKind

// compile time check whether the MarkupKinds implements a gojay.MarshalerJSONArray and gojay.UnmarshalerJSONArray interfaces.
var (
	_ gojay.MarshalerJSONArray   = (*MarkupKinds)(nil)
	_ gojay.UnmarshalerJSONArray = (*MarkupKinds)(nil)
)

// MarshalJSONArray implements gojay.MarshalerJSONArray.
func (v MarkupKinds) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range v {
		enc.String(string(v[i]))
	}
}

// IsNil implements gojay.MarshalerJSONArray.
func (v MarkupKinds) IsNil() bool { return len(v) == 0 }

// UnmarshalJSONArray decodes JSON array elements into slice.
func (v *MarkupKinds) UnmarshalJSONArray(dec *gojay.Decoder) error {
	var value MarkupKind
	if err := dec.String((*string)(&value)); err != nil {
		return err
	}
	*v = append(*v, value)
	return nil
}

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DocumentHighlightParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKeyOmitEmpty(keyTextDocument, &v.TextDocument)
	enc.ObjectKeyOmitEmpty(keyPosition, &v.Position)
	encodeProgressToken(enc, keyWorkDoneToken, v.WorkDoneToken)
	encodeProgressToken(enc, keyPartialResultToken, v.PartialResultToken)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *DocumentHighlightParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *DocumentHighlightParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyTextDocument:
		return dec.Object(&v.TextDocument)
	case keyPosition:
		return dec.Object(&v.Position)
	case keyWorkDoneToken:
		return decodeProgressToken(dec, k, keyWorkDoneToken, v.WorkDoneToken)
	case keyPartialResultToken:
		return decodeProgressToken(dec, k, keyPartialResultToken, v.PartialResultToken)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *DocumentHighlightParams) NKeys() int { return 4 }

// compile time check whether the DocumentHighlightParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DocumentHighlightParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*DocumentHighlightParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DeclarationParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKeyOmitEmpty(keyTextDocument, &v.TextDocument)
	enc.ObjectKeyOmitEmpty(keyPosition, &v.Position)
	encodeProgressToken(enc, keyWorkDoneToken, v.WorkDoneToken)
	encodeProgressToken(enc, keyPartialResultToken, v.PartialResultToken)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *DeclarationParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *DeclarationParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyTextDocument:
		return dec.Object(&v.TextDocument)
	case keyPosition:
		return dec.Object(&v.Position)
	case keyWorkDoneToken:
		return decodeProgressToken(dec, k, keyWorkDoneToken, v.WorkDoneToken)
	case keyPartialResultToken:
		return decodeProgressToken(dec, k, keyPartialResultToken, v.PartialResultToken)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *DeclarationParams) NKeys() int { return 4 }

// compile time check whether the DeclarationParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DeclarationParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*DeclarationParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DefinitionParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKeyOmitEmpty(keyTextDocument, &v.TextDocument)
	enc.ObjectKeyOmitEmpty(keyPosition, &v.Position)
	encodeProgressToken(enc, keyWorkDoneToken, v.WorkDoneToken)
	encodeProgressToken(enc, keyPartialResultToken, v.PartialResultToken)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *DefinitionParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *DefinitionParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyTextDocument:
		return dec.Object(&v.TextDocument)
	case keyPosition:
		return dec.Object(&v.Position)
	case keyWorkDoneToken:
		return decodeProgressToken(dec, k, keyWorkDoneToken, v.WorkDoneToken)
	case keyPartialResultToken:
		return decodeProgressToken(dec, k, keyPartialResultToken, v.PartialResultToken)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *DefinitionParams) NKeys() int { return 4 }

// compile time check whether the DefinitionParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DefinitionParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*DefinitionParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *TypeDefinitionParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKeyOmitEmpty(keyTextDocument, &v.TextDocument)
	enc.ObjectKeyOmitEmpty(keyPosition, &v.Position)
	encodeProgressToken(enc, keyWorkDoneToken, v.WorkDoneToken)
	encodeProgressToken(enc, keyPartialResultToken, v.PartialResultToken)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *TypeDefinitionParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *TypeDefinitionParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyTextDocument:
		return dec.Object(&v.TextDocument)
	case keyPosition:
		return dec.Object(&v.Position)
	case keyWorkDoneToken:
		return decodeProgressToken(dec, k, keyWorkDoneToken, v.WorkDoneToken)
	case keyPartialResultToken:
		return decodeProgressToken(dec, k, keyPartialResultToken, v.PartialResultToken)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *TypeDefinitionParams) NKeys() int { return 4 }

// compile time check whether the TypeDefinitionParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*TypeDefinitionParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*TypeDefinitionParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ImplementationParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKeyOmitEmpty(keyTextDocument, &v.TextDocument)
	enc.ObjectKeyOmitEmpty(keyPosition, &v.Position)
	encodeProgressToken(enc, keyWorkDoneToken, v.WorkDoneToken)
	encodeProgressToken(enc, keyPartialResultToken, v.PartialResultToken)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *ImplementationParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *ImplementationParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyTextDocument:
		return dec.Object(&v.TextDocument)
	case keyPosition:
		return dec.Object(&v.Position)
	case keyWorkDoneToken:
		return decodeProgressToken(dec, k, keyWorkDoneToken, v.WorkDoneToken)
	case keyPartialResultToken:
		return decodeProgressToken(dec, k, keyPartialResultToken, v.PartialResultToken)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *ImplementationParams) NKeys() int { return 4 }

// compile time check whether the ImplementationParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ImplementationParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*ImplementationParams)(nil)
)

// TokenFormats represents a slice of TokenFormat.
type TokenFormats []TokenFormat

// compile time check whether the CodeActionKinds implements a gojay.MarshalerJSONArray and gojay.UnmarshalerJSONArray interfaces.
var (
	_ gojay.MarshalerJSONArray   = (*TokenFormats)(nil)
	_ gojay.UnmarshalerJSONArray = (*TokenFormats)(nil)
)

// MarshalJSONArray implements gojay.MarshalerJSONArray.
func (v TokenFormats) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range v {
		enc.String(string(v[i]))
	}
}

// IsNil implements gojay.MarshalerJSONArray.
func (v TokenFormats) IsNil() bool { return len(v) == 0 }

// UnmarshalJSONArray implements gojay.UnmarshalerJSONArray.
func (v *TokenFormats) UnmarshalJSONArray(dec *gojay.Decoder) error {
	var value TokenFormat
	if err := dec.String((*string)(&value)); err != nil {
		return err
	}
	*v = append(*v, value)
	return nil
}

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ShowDocumentParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyURI, string(v.URI))
	enc.BoolKeyOmitEmpty(keyExternal, v.External)
	enc.BoolKeyOmitEmpty(keyTakeFocus, v.TakeFocus)
	enc.ObjectKeyOmitEmpty(keySelection, v.Selection)
}

// IsNil returns wether the structure is nil value or not.
func (v *ShowDocumentParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *ShowDocumentParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyURI:
		return dec.String((*string)(&v.URI))
	case keyExternal:
		return dec.Bool(&v.External)
	case keyTakeFocus:
		return dec.Bool(&v.TakeFocus)
	case keySelection:
		if v.Selection == nil {
			v.Selection = &Range{}
		}
		return dec.Object(v.Selection)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *ShowDocumentParams) NKeys() int { return 4 }

// compile time check whether the ShowDocumentParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ShowDocumentParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*ShowDocumentParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ShowDocumentResult) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKey(keySuccess, v.Success)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *ShowDocumentResult) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *ShowDocumentResult) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keySuccess {
		return dec.Bool(&v.Success)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *ShowDocumentResult) NKeys() int { return 1 }

// compile time check whether the ShowDocumentResult implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ShowDocumentResult)(nil)
	_ gojay.UnmarshalerJSONObject = (*ShowDocumentResult)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *InitializeResult) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKey(keyCapabilities, &v.Capabilities)
	enc.ObjectKeyOmitEmpty(keyServerInfo, v.ServerInfo)
}

// IsNil returns wether the structure is nil value or not.
func (v *InitializeResult) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *InitializeResult) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyCapabilities:
		return dec.Object(&v.Capabilities)
	case keyServerInfo:
		if v.ServerInfo == nil {
			v.ServerInfo = &ServerInfo{}
		}
		return dec.Object(v.ServerInfo)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *InitializeResult) NKeys() int { return 2 }

// compile time check whether the InitializeResult implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*InitializeResult)(nil)
	_ gojay.UnmarshalerJSONObject = (*InitializeResult)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ServerInfo) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyName, v.Name)
	enc.StringKeyOmitEmpty(keyVersion, v.Version)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *ServerInfo) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *ServerInfo) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyName:
		return dec.String(&v.Name)
	case keyVersion:
		return dec.String(&v.Version)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *ServerInfo) NKeys() int { return 2 }

// compile time check whether the ServerInfo implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ServerInfo)(nil)
	_ gojay.UnmarshalerJSONObject = (*ServerInfo)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *InitializeError) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyRetry, v.Retry)
}

// IsNil returns wether the structure is nil value or not.
func (v *InitializeError) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *InitializeError) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyRetry {
		return dec.Bool(&v.Retry)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *InitializeError) NKeys() int { return 1 }

// compile time check whether the InitializeError implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*InitializeError)(nil)
	_ gojay.UnmarshalerJSONObject = (*InitializeError)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ReferencesOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyWorkDoneProgress, v.WorkDoneProgress)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *ReferencesOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *ReferencesOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyWorkDoneProgress {
		return dec.Bool(&v.WorkDoneProgress)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *ReferencesOptions) NKeys() int { return 1 }

// compile time check whether the ReferencesOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ReferencesOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*ReferencesOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *LinkedEditingRangeParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKeyOmitEmpty(keyTextDocument, &v.TextDocument)
	enc.ObjectKeyOmitEmpty(keyPosition, &v.Position)
	encodeProgressToken(enc, keyWorkDoneToken, v.WorkDoneToken)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *LinkedEditingRangeParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *LinkedEditingRangeParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
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
func (v *LinkedEditingRangeParams) NKeys() int { return 3 }

// compile time check whether the LinkedEditingRangeParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*LinkedEditingRangeParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*LinkedEditingRangeParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *LinkedEditingRanges) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKey(keyRanges, Ranges(v.Ranges))
	enc.StringKeyOmitEmpty(keyWordPattern, v.WordPattern)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *LinkedEditingRanges) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *LinkedEditingRanges) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyRanges:
		return dec.Array((*Ranges)(&v.Ranges))
	case keyWordPattern:
		return dec.String(&v.WordPattern)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *LinkedEditingRanges) NKeys() int { return 2 }

// compile time check whether the LinkedEditingRanges implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*LinkedEditingRanges)(nil)
	_ gojay.UnmarshalerJSONObject = (*LinkedEditingRanges)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *MonikerParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKeyOmitEmpty(keyTextDocument, &v.TextDocument)
	enc.ObjectKeyOmitEmpty(keyPosition, &v.Position)
	encodeProgressToken(enc, keyWorkDoneToken, v.WorkDoneToken)
	encodeProgressToken(enc, keyPartialResultToken, v.PartialResultToken)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *MonikerParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *MonikerParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyTextDocument:
		return dec.Object(&v.TextDocument)
	case keyPosition:
		return dec.Object(&v.Position)
	case keyWorkDoneToken:
		return decodeProgressToken(dec, k, keyWorkDoneToken, v.WorkDoneToken)
	case keyPartialResultToken:
		return decodeProgressToken(dec, k, keyPartialResultToken, v.PartialResultToken)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *MonikerParams) NKeys() int { return 3 }

// compile time check whether the MonikerParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*MonikerParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*MonikerParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *StaticRegistrationOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKeyOmitEmpty(keyID, v.ID)
}

// IsNil returns wether the structure is nil value or not.
func (v *StaticRegistrationOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *StaticRegistrationOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyID {
		return dec.String(&v.ID)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *StaticRegistrationOptions) NKeys() int { return 1 }

// compile time check whether the StaticRegistrationOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*StaticRegistrationOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*StaticRegistrationOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DocumentLinkRegistrationOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.AddArrayKey(keyDocumentSelector, &v.DocumentSelector)
	enc.BoolKeyOmitEmpty(keyResolveProvider, v.ResolveProvider)
}

// IsNil returns wether the structure is nil value or not.
func (v *DocumentLinkRegistrationOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *DocumentLinkRegistrationOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDocumentSelector:
		if v.DocumentSelector == nil {
			v.DocumentSelector = DocumentSelector{}
		}
		return dec.Array(&v.DocumentSelector)
	case keyResolveProvider:
		return dec.Bool(&v.ResolveProvider)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DocumentLinkRegistrationOptions) NKeys() int { return 2 }

// compile time check whether the DocumentLinkRegistrationOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DocumentLinkRegistrationOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*DocumentLinkRegistrationOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *InitializedParams) MarshalJSONObject(enc *gojay.Encoder) {}

// IsNil returns wether the structure is nil value or not.
func (v *InitializedParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *InitializedParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *InitializedParams) NKeys() int { return 0 }

// compile time check whether the InitializedParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*InitializedParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*InitializedParams)(nil)
)
