// SPDX-FileCopyrightText: 2021 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

//go:build gojay
// +build gojay

package protocol

import (
	"github.com/francoispqt/gojay"
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKeyOmitEmpty(keyWorkspace, v.Workspace)
	enc.ObjectKeyOmitEmpty(keyTextDocument, v.TextDocument)
	enc.ObjectKeyOmitEmpty(keyWindow, v.Window)
	enc.ObjectKeyOmitEmpty(keyGeneral, v.General)
	enc.AddInterfaceKeyOmitEmpty(keyExperimental, v.Experimental)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *ClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *ClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyWorkspace:
		if v.Workspace == nil {
			v.Workspace = &WorkspaceClientCapabilities{}
		}
		return dec.Object(v.Workspace)
	case keyTextDocument:
		if v.TextDocument == nil {
			v.TextDocument = &TextDocumentClientCapabilities{}
		}
		return dec.Object(v.TextDocument)
	case keyWindow:
		if v.Window == nil {
			v.Window = &WindowClientCapabilities{}
		}
		return dec.Object(v.Window)
	case keyGeneral:
		if v.General == nil {
			v.General = &GeneralClientCapabilities{}
		}
		return dec.Object(v.General)
	case keyExperimental:
		return dec.Interface(&v.Experimental)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *ClientCapabilities) NKeys() int { return 5 }

// compile time check whether the ClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*ClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *WorkspaceClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyApplyEdit, v.ApplyEdit)
	enc.ObjectKeyOmitEmpty(keyWorkspaceEdit, v.WorkspaceEdit)
	enc.ObjectKeyOmitEmpty(keyDidChangeConfiguration, v.DidChangeConfiguration)
	enc.ObjectKeyOmitEmpty(keyDidChangeWatchedFiles, v.DidChangeWatchedFiles)
	enc.ObjectKeyOmitEmpty(keySymbol, v.Symbol)
	enc.ObjectKeyOmitEmpty(keyExecuteCommand, v.ExecuteCommand)
	enc.BoolKeyOmitEmpty(keyWorkspaceFolders, v.WorkspaceFolders)
	enc.BoolKeyOmitEmpty(keyConfiguration, v.Configuration)
	enc.ObjectKeyOmitEmpty(keySemanticTokens, v.SemanticTokens)
	enc.ObjectKeyOmitEmpty(keyCodeLens, v.CodeLens)
	enc.ObjectKeyOmitEmpty(keyFileOperations, v.FileOperations)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *WorkspaceClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *WorkspaceClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyApplyEdit:
		return dec.Bool(&v.ApplyEdit)
	case keyWorkspaceEdit:
		if v.WorkspaceEdit == nil {
			v.WorkspaceEdit = &WorkspaceClientCapabilitiesWorkspaceEdit{}
		}
		return dec.Object(v.WorkspaceEdit)
	case keyDidChangeConfiguration:
		if v.DidChangeConfiguration == nil {
			v.DidChangeConfiguration = &DidChangeConfigurationWorkspaceClientCapabilities{}
		}
		return dec.Object(v.DidChangeConfiguration)
	case keyDidChangeWatchedFiles:
		if v.DidChangeWatchedFiles == nil {
			v.DidChangeWatchedFiles = &DidChangeWatchedFilesWorkspaceClientCapabilities{}
		}
		return dec.Object(v.DidChangeWatchedFiles)
	case keySymbol:
		if v.Symbol == nil {
			v.Symbol = &WorkspaceSymbolClientCapabilities{}
		}
		return dec.Object(v.Symbol)
	case keyExecuteCommand:
		if v.ExecuteCommand == nil {
			v.ExecuteCommand = &ExecuteCommandClientCapabilities{}
		}
		return dec.Object(v.ExecuteCommand)
	case keyWorkspaceFolders:
		return dec.Bool(&v.WorkspaceFolders)
	case keyConfiguration:
		return dec.Bool(&v.Configuration)
	case keySemanticTokens:
		if v.SemanticTokens == nil {
			v.SemanticTokens = &SemanticTokensWorkspaceClientCapabilities{}
		}
		return dec.Object(v.SemanticTokens)
	case keyCodeLens:
		if v.CodeLens == nil {
			v.CodeLens = &CodeLensWorkspaceClientCapabilities{}
		}
		return dec.Object(v.CodeLens)
	case keyFileOperations:
		if v.FileOperations == nil {
			v.FileOperations = &WorkspaceClientCapabilitiesFileOperations{}
		}
		return dec.Object(v.FileOperations)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *WorkspaceClientCapabilities) NKeys() int { return 11 }

// compile time check whether the WorkspaceClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*WorkspaceClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*WorkspaceClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *WorkspaceClientCapabilitiesWorkspaceEdit) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyDocumentChanges, v.DocumentChanges)
	enc.StringKeyOmitEmpty(keyFailureHandling, v.FailureHandling)
	enc.ArrayKeyOmitEmpty(keyResourceOperations, (*Strings)(&v.ResourceOperations))
	enc.BoolKeyOmitEmpty(keyNormalizesLineEndings, v.NormalizesLineEndings)
	enc.ObjectKeyOmitEmpty(keyChangeAnnotationSupport, v.ChangeAnnotationSupport)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *WorkspaceClientCapabilitiesWorkspaceEdit) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *WorkspaceClientCapabilitiesWorkspaceEdit) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDocumentChanges:
		return dec.Bool(&v.DocumentChanges)
	case keyFailureHandling:
		return dec.String(&v.FailureHandling)
	case keyResourceOperations:
		var values Strings
		err := dec.Array(&values)
		if err == nil && len(values) > 0 {
			v.ResourceOperations = []string(values)
		}
		return err
	case keyNormalizesLineEndings:
		return dec.Bool(&v.NormalizesLineEndings)
	case keyChangeAnnotationSupport:
		if v.ChangeAnnotationSupport == nil {
			v.ChangeAnnotationSupport = &WorkspaceClientCapabilitiesWorkspaceEditChangeAnnotationSupport{}
		}
		return dec.Object(v.ChangeAnnotationSupport)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *WorkspaceClientCapabilitiesWorkspaceEdit) NKeys() int { return 5 }

// compile time check whether the WorkspaceClientCapabilitiesWorkspaceEdit implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*WorkspaceClientCapabilitiesWorkspaceEdit)(nil)
	_ gojay.UnmarshalerJSONObject = (*WorkspaceClientCapabilitiesWorkspaceEdit)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *WorkspaceClientCapabilitiesWorkspaceEditChangeAnnotationSupport) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyGroupsOnLabel, v.GroupsOnLabel)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *WorkspaceClientCapabilitiesWorkspaceEditChangeAnnotationSupport) IsNil() bool {
	return v == nil
}

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *WorkspaceClientCapabilitiesWorkspaceEditChangeAnnotationSupport) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyGroupsOnLabel {
		return dec.Bool(&v.GroupsOnLabel)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *WorkspaceClientCapabilitiesWorkspaceEditChangeAnnotationSupport) NKeys() int { return 1 }

// compile time check whether the WorkspaceClientCapabilitiesWorkspaceEditChangeAnnotationSupport implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*WorkspaceClientCapabilitiesWorkspaceEditChangeAnnotationSupport)(nil)
	_ gojay.UnmarshalerJSONObject = (*WorkspaceClientCapabilitiesWorkspaceEditChangeAnnotationSupport)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DidChangeConfigurationWorkspaceClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyDynamicRegistration, v.DynamicRegistration)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *DidChangeConfigurationWorkspaceClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *DidChangeConfigurationWorkspaceClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyDynamicRegistration {
		return dec.Bool(&v.DynamicRegistration)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DidChangeConfigurationWorkspaceClientCapabilities) NKeys() int { return 1 }

// compile time check whether the DidChangeConfigurationWorkspaceClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DidChangeConfigurationWorkspaceClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*DidChangeConfigurationWorkspaceClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DidChangeWatchedFilesWorkspaceClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyDynamicRegistration, v.DynamicRegistration)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *DidChangeWatchedFilesWorkspaceClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *DidChangeWatchedFilesWorkspaceClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyDynamicRegistration {
		return dec.Bool(&v.DynamicRegistration)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DidChangeWatchedFilesWorkspaceClientCapabilities) NKeys() int { return 1 }

// compile time check whether the DidChangeWatchedFilesWorkspaceClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DidChangeWatchedFilesWorkspaceClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*DidChangeWatchedFilesWorkspaceClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *WorkspaceSymbolClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyDynamicRegistration, v.DynamicRegistration)
	enc.ObjectKeyOmitEmpty(keySymbolKind, v.SymbolKind)
	enc.ObjectKeyOmitEmpty(keyTagSupport, v.TagSupport)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *WorkspaceSymbolClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *WorkspaceSymbolClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDynamicRegistration:
		return dec.Bool(&v.DynamicRegistration)
	case keySymbolKind:
		if v.SymbolKind == nil {
			v.SymbolKind = &SymbolKindCapabilities{}
		}
		return dec.Object(v.SymbolKind)
	case keyTagSupport:
		if v.TagSupport == nil {
			v.TagSupport = &TagSupportCapabilities{}
		}
		return dec.Object(v.TagSupport)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *WorkspaceSymbolClientCapabilities) NKeys() int { return 3 }

// compile time check whether the WorkspaceSymbolClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*WorkspaceSymbolClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*WorkspaceSymbolClientCapabilities)(nil)
)

// SymbolKinds represents a slice of SymbolKind.
type SymbolKinds []SymbolKind

// compile time check whether the SymbolKinds implements a gojay.MarshalerJSONArray and gojay.UnmarshalerJSONArray interfaces.
var (
	_ gojay.MarshalerJSONArray   = (*SymbolKinds)(nil)
	_ gojay.UnmarshalerJSONArray = (*SymbolKinds)(nil)
)

// MarshalJSONArray implements gojay.MarshalerJSONArray.
func (v SymbolKinds) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range v {
		enc.Float64OmitEmpty(float64(v[i]))
	}
}

// IsNil implements gojay.MarshalerJSONArray.
func (v SymbolKinds) IsNil() bool { return len(v) == 0 }

// UnmarshalJSONArray implements gojay.UnmarshalerJSONArray.
func (v *SymbolKinds) UnmarshalJSONArray(dec *gojay.Decoder) error {
	var value float64
	if err := dec.Float64(&value); err != nil {
		return err
	}
	*v = append(*v, SymbolKind(value))
	return nil
}

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *SymbolKindCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKeyOmitEmpty(keyValueSet, (*SymbolKinds)(&v.ValueSet))
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *SymbolKindCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *SymbolKindCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyValueSet {
		return dec.Array((*SymbolKinds)(&v.ValueSet))
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *SymbolKindCapabilities) NKeys() int { return 1 }

// compile time check whether the SymbolKindCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*SymbolKindCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*SymbolKindCapabilities)(nil)
)

// SymbolTags represents a slice of SymbolTag.
type SymbolTags []SymbolTag

// compile time check whether the SymbolTags implements a gojay.MarshalerJSONArray and gojay.UnmarshalerJSONArray interfaces.
var (
	_ gojay.MarshalerJSONArray   = (*SymbolTags)(nil)
	_ gojay.UnmarshalerJSONArray = (*SymbolTags)(nil)
)

// MarshalJSONArray implements gojay.MarshalerJSONArray.
func (v SymbolTags) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range v {
		enc.Float64OmitEmpty(float64(v[i]))
	}
}

// IsNil implements gojay.MarshalerJSONArray.
func (v SymbolTags) IsNil() bool { return len(v) == 0 }

// UnmarshalJSONArray decodes JSON array elements into slice.
func (v *SymbolTags) UnmarshalJSONArray(dec *gojay.Decoder) error {
	var value float64
	if err := dec.Float64(&value); err != nil {
		return err
	}
	*v = append(*v, SymbolTag(value))
	return nil
}

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *TagSupportCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKeyOmitEmpty(keyValueSet, (*SymbolTags)(&v.ValueSet))
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *TagSupportCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *TagSupportCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyValueSet {
		return dec.Array((*SymbolTags)(&v.ValueSet))
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *TagSupportCapabilities) NKeys() int { return 1 }

// compile time check whether the WorkspaceSymbolTagSupport implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*TagSupportCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*TagSupportCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ExecuteCommandClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyDynamicRegistration, v.DynamicRegistration)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *ExecuteCommandClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *ExecuteCommandClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyDynamicRegistration {
		return dec.Bool(&v.DynamicRegistration)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *ExecuteCommandClientCapabilities) NKeys() int { return 1 }

// compile time check whether the ExecuteCommandClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ExecuteCommandClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*ExecuteCommandClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *SemanticTokensWorkspaceClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyRefreshSupport, v.RefreshSupport)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *SemanticTokensWorkspaceClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *SemanticTokensWorkspaceClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyRefreshSupport {
		return dec.Bool(&v.RefreshSupport)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *SemanticTokensWorkspaceClientCapabilities) NKeys() int { return 1 }

// compile time check whether the SemanticTokensWorkspaceClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*SemanticTokensWorkspaceClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*SemanticTokensWorkspaceClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CodeLensWorkspaceClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyRefreshSupport, v.RefreshSupport)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *CodeLensWorkspaceClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *CodeLensWorkspaceClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyRefreshSupport {
		return dec.Bool(&v.RefreshSupport)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *CodeLensWorkspaceClientCapabilities) NKeys() int { return 1 }

// compile time check whether the CodeLensWorkspaceClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CodeLensWorkspaceClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*CodeLensWorkspaceClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *WorkspaceClientCapabilitiesFileOperations) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyDynamicRegistration, v.DynamicRegistration)
	enc.BoolKeyOmitEmpty(keyDidCreate, v.DidCreate)
	enc.BoolKeyOmitEmpty(keyWillCreate, v.WillCreate)
	enc.BoolKeyOmitEmpty(keyDidRename, v.DidRename)
	enc.BoolKeyOmitEmpty(keyWillRename, v.WillRename)
	enc.BoolKeyOmitEmpty(keyDidDelete, v.DidDelete)
	enc.BoolKeyOmitEmpty(keyWillDelete, v.WillDelete)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *WorkspaceClientCapabilitiesFileOperations) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *WorkspaceClientCapabilitiesFileOperations) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDynamicRegistration:
		return dec.Bool(&v.DynamicRegistration)
	case keyDidCreate:
		return dec.Bool(&v.DidCreate)
	case keyWillCreate:
		return dec.Bool(&v.WillCreate)
	case keyDidRename:
		return dec.Bool(&v.DidRename)
	case keyWillRename:
		return dec.Bool(&v.WillRename)
	case keyDidDelete:
		return dec.Bool(&v.DidDelete)
	case keyWillDelete:
		return dec.Bool(&v.WillDelete)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *WorkspaceClientCapabilitiesFileOperations) NKeys() int { return 7 }

// compile time check whether the WorkspaceClientCapabilitiesFileOperations implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*WorkspaceClientCapabilitiesFileOperations)(nil)
	_ gojay.UnmarshalerJSONObject = (*WorkspaceClientCapabilitiesFileOperations)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *TextDocumentClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKeyOmitEmpty(keySynchronization, v.Synchronization)
	enc.ObjectKeyOmitEmpty(keyCompletion, v.Completion)
	enc.ObjectKeyOmitEmpty(keyHover, v.Hover)
	enc.ObjectKeyOmitEmpty(keySignatureHelp, v.SignatureHelp)
	enc.ObjectKeyOmitEmpty(keyDeclaration, v.Declaration)
	enc.ObjectKeyOmitEmpty(keyDefinition, v.Definition)
	enc.ObjectKeyOmitEmpty(keyTypeDefinition, v.TypeDefinition)
	enc.ObjectKeyOmitEmpty(keyImplementation, v.Implementation)
	enc.ObjectKeyOmitEmpty(keyReferences, v.References)
	enc.ObjectKeyOmitEmpty(keyDocumentHighlight, v.DocumentHighlight)
	enc.ObjectKeyOmitEmpty(keyDocumentSymbol, v.DocumentSymbol)
	enc.ObjectKeyOmitEmpty(keyCodeAction, v.CodeAction)
	enc.ObjectKeyOmitEmpty(keyCodeLens, v.CodeLens)
	enc.ObjectKeyOmitEmpty(keyDocumentLink, v.DocumentLink)
	enc.ObjectKeyOmitEmpty(keyColorProvider, v.ColorProvider)
	enc.ObjectKeyOmitEmpty(keyFormatting, v.Formatting)
	enc.ObjectKeyOmitEmpty(keyRangeFormatting, v.RangeFormatting)
	enc.ObjectKeyOmitEmpty(keyOnTypeFormatting, v.OnTypeFormatting)
	enc.ObjectKeyOmitEmpty(keyPublishDiagnostics, v.PublishDiagnostics)
	enc.ObjectKeyOmitEmpty(keyRename, v.Rename)
	enc.ObjectKeyOmitEmpty(keyFoldingRange, v.FoldingRange)
	enc.ObjectKeyOmitEmpty(keySelectionRange, v.SelectionRange)
	enc.ObjectKeyOmitEmpty(keyCallHierarchy, v.CallHierarchy)
	enc.ObjectKeyOmitEmpty(keySemanticTokens, v.SemanticTokens)
	enc.ObjectKeyOmitEmpty(keyLinkedEditingRange, v.LinkedEditingRange)
	enc.ObjectKeyOmitEmpty(keyMoniker, v.Moniker)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *TextDocumentClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
//nolint:funlen,gocognit
func (v *TextDocumentClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keySynchronization:
		if v.Synchronization == nil {
			v.Synchronization = &TextDocumentSyncClientCapabilities{}
		}
		return dec.Object(v.Synchronization)
	case keyCompletion:
		if v.Completion == nil {
			v.Completion = &CompletionTextDocumentClientCapabilities{}
		}
		return dec.Object(v.Completion)
	case keyHover:
		if v.Hover == nil {
			v.Hover = &HoverTextDocumentClientCapabilities{}
		}
		return dec.Object(v.Hover)
	case keySignatureHelp:
		if v.SignatureHelp == nil {
			v.SignatureHelp = &SignatureHelpTextDocumentClientCapabilities{}
		}
		return dec.Object(v.SignatureHelp)
	case keyDeclaration:
		if v.Declaration == nil {
			v.Declaration = &DeclarationTextDocumentClientCapabilities{}
		}
		return dec.Object(v.Declaration)
	case keyDefinition:
		if v.Definition == nil {
			v.Definition = &DefinitionTextDocumentClientCapabilities{}
		}
		return dec.Object(v.Definition)
	case keyTypeDefinition:
		if v.TypeDefinition == nil {
			v.TypeDefinition = &TypeDefinitionTextDocumentClientCapabilities{}
		}
		return dec.Object(v.TypeDefinition)
	case keyImplementation:
		if v.Implementation == nil {
			v.Implementation = &ImplementationTextDocumentClientCapabilities{}
		}
		return dec.Object(v.Implementation)
	case keyReferences:
		if v.References == nil {
			v.References = &ReferencesTextDocumentClientCapabilities{}
		}
		return dec.Object(v.References)
	case keyDocumentHighlight:
		if v.DocumentHighlight == nil {
			v.DocumentHighlight = &DocumentHighlightClientCapabilities{}
		}
		return dec.Object(v.DocumentHighlight)
	case keyDocumentSymbol:
		if v.DocumentSymbol == nil {
			v.DocumentSymbol = &DocumentSymbolClientCapabilities{}
		}
		return dec.Object(v.DocumentSymbol)
	case keyCodeAction:
		if v.CodeAction == nil {
			v.CodeAction = &CodeActionClientCapabilities{}
		}
		return dec.Object(v.CodeAction)
	case keyCodeLens:
		if v.CodeLens == nil {
			v.CodeLens = &CodeLensClientCapabilities{}
		}
		return dec.Object(v.CodeLens)
	case keyDocumentLink:
		if v.DocumentLink == nil {
			v.DocumentLink = &DocumentLinkClientCapabilities{}
		}
		return dec.Object(v.DocumentLink)
	case keyColorProvider:
		if v.ColorProvider == nil {
			v.ColorProvider = &DocumentColorClientCapabilities{}
		}
		return dec.Object(v.ColorProvider)
	case keyFormatting:
		if v.Formatting == nil {
			v.Formatting = &DocumentFormattingClientCapabilities{}
		}
		return dec.Object(v.Formatting)
	case keyRangeFormatting:
		if v.RangeFormatting == nil {
			v.RangeFormatting = &DocumentRangeFormattingClientCapabilities{}
		}
		return dec.Object(v.RangeFormatting)
	case keyOnTypeFormatting:
		if v.OnTypeFormatting == nil {
			v.OnTypeFormatting = &DocumentOnTypeFormattingClientCapabilities{}
		}
		return dec.Object(v.OnTypeFormatting)
	case keyPublishDiagnostics:
		if v.PublishDiagnostics == nil {
			v.PublishDiagnostics = &PublishDiagnosticsClientCapabilities{}
		}
		return dec.Object(v.PublishDiagnostics)
	case keyRename:
		if v.Rename == nil {
			v.Rename = &RenameClientCapabilities{}
		}
		return dec.Object(v.Rename)
	case keyFoldingRange:
		if v.FoldingRange == nil {
			v.FoldingRange = &FoldingRangeClientCapabilities{}
		}
		return dec.Object(v.FoldingRange)
	case keySelectionRange:
		if v.SelectionRange == nil {
			v.SelectionRange = &SelectionRangeClientCapabilities{}
		}
		return dec.Object(v.SelectionRange)
	case keyCallHierarchy:
		if v.CallHierarchy == nil {
			v.CallHierarchy = &CallHierarchyClientCapabilities{}
		}
		return dec.Object(v.CallHierarchy)
	case keySemanticTokens:
		if v.SemanticTokens == nil {
			v.SemanticTokens = &SemanticTokensClientCapabilities{}
		}
		return dec.Object(v.SemanticTokens)
	case keyLinkedEditingRange:
		if v.LinkedEditingRange == nil {
			v.LinkedEditingRange = &LinkedEditingRangeClientCapabilities{}
		}
		return dec.Object(v.LinkedEditingRange)
	case keyMoniker:
		if v.Moniker == nil {
			v.Moniker = &MonikerClientCapabilities{}
		}
		return dec.Object(v.Moniker)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *TextDocumentClientCapabilities) NKeys() int { return 26 }

// compile time check whether the TextDocumentClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*TextDocumentClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*TextDocumentClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *TextDocumentSyncClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyDynamicRegistration, v.DynamicRegistration)
	enc.BoolKeyOmitEmpty(keyWillSave, v.WillSave)
	enc.BoolKeyOmitEmpty(keyWillSaveWaitUntil, v.WillSaveWaitUntil)
	enc.BoolKeyOmitEmpty(keyDidSave, v.DidSave)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *TextDocumentSyncClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *TextDocumentSyncClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDynamicRegistration:
		return dec.Bool(&v.DynamicRegistration)
	case keyWillSave:
		return dec.Bool(&v.WillSave)
	case keyWillSaveWaitUntil:
		return dec.Bool(&v.WillSaveWaitUntil)
	case keyDidSave:
		return dec.Bool(&v.DidSave)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *TextDocumentSyncClientCapabilities) NKeys() int { return 4 }

// compile time check whether the TextDocumentSyncClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*TextDocumentSyncClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*TextDocumentSyncClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CompletionTextDocumentClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyDynamicRegistration, v.DynamicRegistration)
	enc.ObjectKeyOmitEmpty(keyCompletionItem, v.CompletionItem)
	enc.ObjectKeyOmitEmpty(keyCompletionItemKind, v.CompletionItemKind)
	enc.BoolKeyOmitEmpty(keyContextSupport, v.ContextSupport)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *CompletionTextDocumentClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *CompletionTextDocumentClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDynamicRegistration:
		return dec.Bool(&v.DynamicRegistration)
	case keyCompletionItem:
		if v.CompletionItem == nil {
			v.CompletionItem = &CompletionTextDocumentClientCapabilitiesItem{}
		}
		return dec.Object(v.CompletionItem)
	case keyCompletionItemKind:
		if v.CompletionItemKind == nil {
			v.CompletionItemKind = &CompletionTextDocumentClientCapabilitiesItemKind{}
		}
		return dec.Object(v.CompletionItemKind)
	case keyContextSupport:
		return dec.Bool(&v.ContextSupport)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *CompletionTextDocumentClientCapabilities) NKeys() int { return 4 }

// compile time check whether the CompletionTextDocumentClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CompletionTextDocumentClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*CompletionTextDocumentClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CompletionTextDocumentClientCapabilitiesItem) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keySnippetSupport, v.SnippetSupport)
	enc.BoolKeyOmitEmpty(keyCommitCharactersSupport, v.CommitCharactersSupport)
	enc.ArrayKeyOmitEmpty(keyDocumentationFormat, (*MarkupKinds)(&v.DocumentationFormat))
	enc.BoolKeyOmitEmpty(keyDeprecatedSupport, v.DeprecatedSupport)
	enc.BoolKeyOmitEmpty(keyPreselectSupport, v.PreselectSupport)
	enc.ObjectKeyOmitEmpty(keyTagSupport, v.TagSupport)
	enc.BoolKeyOmitEmpty(keyInsertReplaceSupport, v.InsertReplaceSupport)
	enc.ObjectKeyOmitEmpty(keyResolveSupport, v.ResolveSupport)
	enc.ObjectKeyOmitEmpty(keyInsertTextModeSupport, v.InsertTextModeSupport)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *CompletionTextDocumentClientCapabilitiesItem) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *CompletionTextDocumentClientCapabilitiesItem) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keySnippetSupport:
		return dec.Bool(&v.SnippetSupport)
	case keyCommitCharactersSupport:
		return dec.Bool(&v.CommitCharactersSupport)
	case keyDocumentationFormat:
		return dec.Array((*MarkupKinds)(&v.DocumentationFormat))
	case keyDeprecatedSupport:
		return dec.Bool(&v.DeprecatedSupport)
	case keyPreselectSupport:
		return dec.Bool(&v.PreselectSupport)
	case keyTagSupport:
		if v.TagSupport == nil {
			v.TagSupport = &CompletionTextDocumentClientCapabilitiesItemTagSupport{}
		}
		return dec.Object(v.TagSupport)
	case keyInsertReplaceSupport:
		return dec.Bool(&v.InsertReplaceSupport)
	case keyResolveSupport:
		if v.ResolveSupport == nil {
			v.ResolveSupport = &CompletionTextDocumentClientCapabilitiesItemResolveSupport{}
		}
		return dec.Object(v.ResolveSupport)
	case keyInsertTextModeSupport:
		if v.InsertTextModeSupport == nil {
			v.InsertTextModeSupport = &CompletionTextDocumentClientCapabilitiesItemInsertTextModeSupport{}
		}
		return dec.Object(v.InsertTextModeSupport)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *CompletionTextDocumentClientCapabilitiesItem) NKeys() int { return 9 }

// compile time check whether the CompletionTextDocumentClientCapabilitiesItem implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CompletionTextDocumentClientCapabilitiesItem)(nil)
	_ gojay.UnmarshalerJSONObject = (*CompletionTextDocumentClientCapabilitiesItem)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CompletionTextDocumentClientCapabilitiesItemTagSupport) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKey(keyValueSet, (*CompletionItemTags)(&v.ValueSet))
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *CompletionTextDocumentClientCapabilitiesItemTagSupport) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *CompletionTextDocumentClientCapabilitiesItemTagSupport) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyValueSet {
		return dec.Array((*CompletionItemTags)(&v.ValueSet))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *CompletionTextDocumentClientCapabilitiesItemTagSupport) NKeys() int { return 1 }

// compile time check whether the CompletionTextDocumentClientCapabilitiesItemTagSupport implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CompletionTextDocumentClientCapabilitiesItemTagSupport)(nil)
	_ gojay.UnmarshalerJSONObject = (*CompletionTextDocumentClientCapabilitiesItemTagSupport)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CompletionTextDocumentClientCapabilitiesItemResolveSupport) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKey(keyProperties, (*Strings)(&v.Properties))
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *CompletionTextDocumentClientCapabilitiesItemResolveSupport) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *CompletionTextDocumentClientCapabilitiesItemResolveSupport) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyProperties {
		return dec.Array((*Strings)(&v.Properties))
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *CompletionTextDocumentClientCapabilitiesItemResolveSupport) NKeys() int { return 1 }

// compile time check whether the CompletionTextDocumentClientCapabilitiesItemResolveSupport implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CompletionTextDocumentClientCapabilitiesItemResolveSupport)(nil)
	_ gojay.UnmarshalerJSONObject = (*CompletionTextDocumentClientCapabilitiesItemResolveSupport)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CompletionTextDocumentClientCapabilitiesItemInsertTextModeSupport) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKey(keyValueSet, (*InsertTextModes)(&v.ValueSet))
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *CompletionTextDocumentClientCapabilitiesItemInsertTextModeSupport) IsNil() bool {
	return v == nil
}

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *CompletionTextDocumentClientCapabilitiesItemInsertTextModeSupport) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyValueSet {
		return dec.Array((*InsertTextModes)(&v.ValueSet))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *CompletionTextDocumentClientCapabilitiesItemInsertTextModeSupport) NKeys() int { return 1 }

// compile time check whether the CompletionTextDocumentClientCapabilitiesItemInsertTextModeSupport implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CompletionTextDocumentClientCapabilitiesItemInsertTextModeSupport)(nil)
	_ gojay.UnmarshalerJSONObject = (*CompletionTextDocumentClientCapabilitiesItemInsertTextModeSupport)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CompletionTextDocumentClientCapabilitiesItemKind) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKey(keyValueSet, (*CompletionItemKinds)(&v.ValueSet))
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *CompletionTextDocumentClientCapabilitiesItemKind) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *CompletionTextDocumentClientCapabilitiesItemKind) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyValueSet {
		return dec.Array((*CompletionItemKinds)(&v.ValueSet))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *CompletionTextDocumentClientCapabilitiesItemKind) NKeys() int { return 1 }

// compile time check whether the CompletionTextDocumentClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CompletionTextDocumentClientCapabilitiesItemKind)(nil)
	_ gojay.UnmarshalerJSONObject = (*CompletionTextDocumentClientCapabilitiesItemKind)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *HoverTextDocumentClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyDynamicRegistration, v.DynamicRegistration)
	enc.ArrayKeyOmitEmpty(keyContentFormat, (*MarkupKinds)(&v.ContentFormat))
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *HoverTextDocumentClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *HoverTextDocumentClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDynamicRegistration:
		return dec.Bool(&v.DynamicRegistration)
	case keyContentFormat:
		return dec.Array((*MarkupKinds)(&v.ContentFormat))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *HoverTextDocumentClientCapabilities) NKeys() int { return 2 }

// compile time check whether the HoverTextDocumentClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*HoverTextDocumentClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*HoverTextDocumentClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *SignatureHelpTextDocumentClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyDynamicRegistration, v.DynamicRegistration)
	enc.ObjectKeyOmitEmpty(keySignatureInformation, v.SignatureInformation)
	enc.BoolKeyOmitEmpty(keyContextSupport, v.ContextSupport)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *SignatureHelpTextDocumentClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *SignatureHelpTextDocumentClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDynamicRegistration:
		return dec.Bool(&v.DynamicRegistration)
	case keySignatureInformation:
		if v.SignatureInformation == nil {
			v.SignatureInformation = &TextDocumentClientCapabilitiesSignatureInformation{}
		}
		return dec.Object(v.SignatureInformation)
	case keyContextSupport:
		return dec.Bool(&v.ContextSupport)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *SignatureHelpTextDocumentClientCapabilities) NKeys() int { return 3 }

// compile time check whether the SignatureHelpTextDocumentClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*SignatureHelpTextDocumentClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*SignatureHelpTextDocumentClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *TextDocumentClientCapabilitiesSignatureInformation) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKeyOmitEmpty(keyDocumentationFormat, (*MarkupKinds)(&v.DocumentationFormat))
	enc.ObjectKeyOmitEmpty(keyParameterInformation, v.ParameterInformation)
	enc.BoolKeyOmitEmpty(keyActiveParameterSupport, v.ActiveParameterSupport)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *TextDocumentClientCapabilitiesSignatureInformation) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *TextDocumentClientCapabilitiesSignatureInformation) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDocumentationFormat:
		return dec.Array((*MarkupKinds)(&v.DocumentationFormat))
	case keyParameterInformation:
		if v.ParameterInformation == nil {
			v.ParameterInformation = &TextDocumentClientCapabilitiesParameterInformation{}
		}
		return dec.Object(v.ParameterInformation)
	case keyActiveParameterSupport:
		return dec.Bool(&v.ActiveParameterSupport)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *TextDocumentClientCapabilitiesSignatureInformation) NKeys() int { return 3 }

// compile time check whether the TextDocumentClientCapabilitiesSignatureInformation implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*TextDocumentClientCapabilitiesSignatureInformation)(nil)
	_ gojay.UnmarshalerJSONObject = (*TextDocumentClientCapabilitiesSignatureInformation)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *TextDocumentClientCapabilitiesParameterInformation) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyLabelOffsetSupport, v.LabelOffsetSupport)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *TextDocumentClientCapabilitiesParameterInformation) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *TextDocumentClientCapabilitiesParameterInformation) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyLabelOffsetSupport {
		return dec.Bool(&v.LabelOffsetSupport)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *TextDocumentClientCapabilitiesParameterInformation) NKeys() int { return 1 }

// compile time check whether the TextDocumentClientCapabilitiesSignatureInformation implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*TextDocumentClientCapabilitiesParameterInformation)(nil)
	_ gojay.UnmarshalerJSONObject = (*TextDocumentClientCapabilitiesParameterInformation)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DeclarationTextDocumentClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyDynamicRegistration, v.DynamicRegistration)
	enc.BoolKeyOmitEmpty(keyLinkSupport, v.LinkSupport)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *DeclarationTextDocumentClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *DeclarationTextDocumentClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDynamicRegistration:
		return dec.Bool(&v.DynamicRegistration)
	case keyLinkSupport:
		return dec.Bool(&v.LinkSupport)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DeclarationTextDocumentClientCapabilities) NKeys() int { return 2 }

// compile time check whether the DeclarationTextDocumentClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DeclarationTextDocumentClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*DeclarationTextDocumentClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DefinitionTextDocumentClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyDynamicRegistration, v.DynamicRegistration)
	enc.BoolKeyOmitEmpty(keyLinkSupport, v.LinkSupport)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *DefinitionTextDocumentClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *DefinitionTextDocumentClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDynamicRegistration:
		return dec.Bool(&v.DynamicRegistration)
	case keyLinkSupport:
		return dec.Bool(&v.LinkSupport)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DefinitionTextDocumentClientCapabilities) NKeys() int { return 2 }

// compile time check whether the DefinitionTextDocumentClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DefinitionTextDocumentClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*DefinitionTextDocumentClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *TypeDefinitionTextDocumentClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyDynamicRegistration, v.DynamicRegistration)
	enc.BoolKeyOmitEmpty(keyLinkSupport, v.LinkSupport)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *TypeDefinitionTextDocumentClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *TypeDefinitionTextDocumentClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDynamicRegistration:
		return dec.Bool(&v.DynamicRegistration)
	case keyLinkSupport:
		return dec.Bool(&v.LinkSupport)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *TypeDefinitionTextDocumentClientCapabilities) NKeys() int { return 2 }

// compile time check whether the TypeDefinitionTextDocumentClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*TypeDefinitionTextDocumentClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*TypeDefinitionTextDocumentClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ImplementationTextDocumentClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyDynamicRegistration, v.DynamicRegistration)
	enc.BoolKeyOmitEmpty(keyLinkSupport, v.LinkSupport)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *ImplementationTextDocumentClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *ImplementationTextDocumentClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDynamicRegistration:
		return dec.Bool(&v.DynamicRegistration)
	case keyLinkSupport:
		return dec.Bool(&v.LinkSupport)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *ImplementationTextDocumentClientCapabilities) NKeys() int { return 2 }

// compile time check whether the ImplementationTextDocumentClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ImplementationTextDocumentClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*ImplementationTextDocumentClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ReferencesTextDocumentClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyDynamicRegistration, v.DynamicRegistration)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *ReferencesTextDocumentClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *ReferencesTextDocumentClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyDynamicRegistration {
		return dec.Bool(&v.DynamicRegistration)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *ReferencesTextDocumentClientCapabilities) NKeys() int { return 1 }

// compile time check whether the ReferencesTextDocumentClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ReferencesTextDocumentClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*ReferencesTextDocumentClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DocumentHighlightClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyDynamicRegistration, v.DynamicRegistration)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *DocumentHighlightClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *DocumentHighlightClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyDynamicRegistration {
		return dec.Bool(&v.DynamicRegistration)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DocumentHighlightClientCapabilities) NKeys() int { return 1 }

// compile time check whether the DocumentHighlightClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DocumentHighlightClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*DocumentHighlightClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DocumentSymbolClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyDynamicRegistration, v.DynamicRegistration)
	enc.ObjectKeyOmitEmpty(keySymbolKind, v.SymbolKind)
	enc.BoolKeyOmitEmpty(keyHierarchicalDocumentSymbolSupport, v.HierarchicalDocumentSymbolSupport)
	enc.ObjectKeyOmitEmpty(keyTagSupport, v.TagSupport)
	enc.BoolKeyOmitEmpty(keyLabelSupport, v.LabelSupport)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *DocumentSymbolClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *DocumentSymbolClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDynamicRegistration:
		return dec.Bool(&v.DynamicRegistration)
	case keySymbolKind:
		if v.SymbolKind == nil {
			v.SymbolKind = &SymbolKindCapabilities{}
		}
		return dec.Object(v.SymbolKind)
	case keyHierarchicalDocumentSymbolSupport:
		return dec.Bool(&v.HierarchicalDocumentSymbolSupport)
	case keyTagSupport:
		if v.TagSupport == nil {
			v.TagSupport = &DocumentSymbolClientCapabilitiesTagSupport{}
		}
		return dec.Object(v.TagSupport)
	case keyLabelSupport:
		return dec.Bool(&v.LabelSupport)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DocumentSymbolClientCapabilities) NKeys() int { return 5 }

// compile time check whether the DocumentSymbolClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DocumentSymbolClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*DocumentSymbolClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DocumentSymbolClientCapabilitiesTagSupport) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKey(keyValueSet, (*SymbolTags)(&v.ValueSet))
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *DocumentSymbolClientCapabilitiesTagSupport) IsNil() bool {
	return v == nil
}

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *DocumentSymbolClientCapabilitiesTagSupport) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyValueSet {
		return dec.Array((*SymbolTags)(&v.ValueSet))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DocumentSymbolClientCapabilitiesTagSupport) NKeys() int { return 1 }

// compile time check whether the DocumentSymbolClientCapabilitiesTagSupport implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DocumentSymbolClientCapabilitiesTagSupport)(nil)
	_ gojay.UnmarshalerJSONObject = (*DocumentSymbolClientCapabilitiesTagSupport)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CodeActionClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyDynamicRegistration, v.DynamicRegistration)
	enc.ObjectKeyOmitEmpty(keyCodeActionLiteralSupport, v.CodeActionLiteralSupport)
	enc.BoolKeyOmitEmpty(keyIsPreferredSupport, v.IsPreferredSupport)
	enc.BoolKeyOmitEmpty(keyDisabledSupport, v.DisabledSupport)
	enc.BoolKeyOmitEmpty(keyDataSupport, v.DataSupport)
	enc.ObjectKeyOmitEmpty(keyResolveSupport, v.ResolveSupport)
	enc.BoolKeyOmitEmpty(keyHonorsChangeAnnotations, v.HonorsChangeAnnotations)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *CodeActionClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *CodeActionClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDynamicRegistration:
		return dec.Bool(&v.DynamicRegistration)
	case keyCodeActionLiteralSupport:
		if v.CodeActionLiteralSupport == nil {
			v.CodeActionLiteralSupport = &CodeActionClientCapabilitiesLiteralSupport{}
		}
		return dec.Object(v.CodeActionLiteralSupport)
	case keyIsPreferredSupport:
		return dec.Bool(&v.IsPreferredSupport)
	case keyDisabledSupport:
		return dec.Bool(&v.DisabledSupport)
	case keyDataSupport:
		return dec.Bool(&v.DataSupport)
	case keyResolveSupport:
		if v.ResolveSupport == nil {
			v.ResolveSupport = &CodeActionClientCapabilitiesResolveSupport{}
		}
		return dec.Object(v.ResolveSupport)
	case keyHonorsChangeAnnotations:
		return dec.Bool(&v.HonorsChangeAnnotations)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *CodeActionClientCapabilities) NKeys() int { return 7 }

// compile time check whether the CodeActionClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CodeActionClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*CodeActionClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CodeActionClientCapabilitiesLiteralSupport) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKeyOmitEmpty(keyCodeActionKind, v.CodeActionKind)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *CodeActionClientCapabilitiesLiteralSupport) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *CodeActionClientCapabilitiesLiteralSupport) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyCodeActionKind {
		if v.CodeActionKind == nil {
			v.CodeActionKind = &CodeActionClientCapabilitiesKind{}
		}
		return dec.Object(v.CodeActionKind)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *CodeActionClientCapabilitiesLiteralSupport) NKeys() int { return 1 }

// compile time check whether the CodeActionClientCapabilitiesLiteralSupport implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CodeActionClientCapabilitiesLiteralSupport)(nil)
	_ gojay.UnmarshalerJSONObject = (*CodeActionClientCapabilitiesLiteralSupport)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CodeActionClientCapabilitiesKind) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKey(keyValueSet, (*CodeActionKinds)(&v.ValueSet))
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *CodeActionClientCapabilitiesKind) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *CodeActionClientCapabilitiesKind) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyValueSet {
		return dec.Array((*CodeActionKinds)(&v.ValueSet))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *CodeActionClientCapabilitiesKind) NKeys() int { return 1 }

// compile time check whether the CodeActionClientCapabilitiesKind implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CodeActionClientCapabilitiesKind)(nil)
	_ gojay.UnmarshalerJSONObject = (*CodeActionClientCapabilitiesKind)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CodeActionClientCapabilitiesResolveSupport) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKey(keyProperties, (*Strings)(&v.Properties))
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *CodeActionClientCapabilitiesResolveSupport) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *CodeActionClientCapabilitiesResolveSupport) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyProperties {
		return dec.Array((*Strings)(&v.Properties))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *CodeActionClientCapabilitiesResolveSupport) NKeys() int { return 1 }

// compile time check whether the CodeActionClientCapabilitiesResolveSupport implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CodeActionClientCapabilitiesResolveSupport)(nil)
	_ gojay.UnmarshalerJSONObject = (*CodeActionClientCapabilitiesResolveSupport)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CodeLensClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyDynamicRegistration, v.DynamicRegistration)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *CodeLensClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *CodeLensClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyDynamicRegistration {
		return dec.Bool(&v.DynamicRegistration)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *CodeLensClientCapabilities) NKeys() int { return 1 }

// compile time check whether the CodeLensClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CodeLensClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*CodeLensClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DocumentLinkClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyDynamicRegistration, v.DynamicRegistration)
	enc.BoolKeyOmitEmpty(keyTooltipSupport, v.TooltipSupport)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *DocumentLinkClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *DocumentLinkClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDynamicRegistration:
		return dec.Bool(&v.DynamicRegistration)
	case keyTooltipSupport:
		return dec.Bool(&v.TooltipSupport)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DocumentLinkClientCapabilities) NKeys() int { return 2 }

// compile time check whether the DocumentLinkClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DocumentLinkClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*DocumentLinkClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DocumentColorClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyDynamicRegistration, v.DynamicRegistration)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *DocumentColorClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *DocumentColorClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyDynamicRegistration {
		return dec.Bool(&v.DynamicRegistration)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DocumentColorClientCapabilities) NKeys() int { return 1 }

// compile time check whether the DocumentColorClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DocumentColorClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*DocumentColorClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DocumentFormattingClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyDynamicRegistration, v.DynamicRegistration)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *DocumentFormattingClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *DocumentFormattingClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyDynamicRegistration {
		return dec.Bool(&v.DynamicRegistration)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DocumentFormattingClientCapabilities) NKeys() int { return 1 }

// compile time check whether the DocumentFormattingClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DocumentFormattingClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*DocumentFormattingClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DocumentRangeFormattingClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyDynamicRegistration, v.DynamicRegistration)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *DocumentRangeFormattingClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *DocumentRangeFormattingClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyDynamicRegistration {
		return dec.Bool(&v.DynamicRegistration)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DocumentRangeFormattingClientCapabilities) NKeys() int { return 1 }

// compile time check whether the DocumentRangeFormattingClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DocumentRangeFormattingClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*DocumentRangeFormattingClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DocumentOnTypeFormattingClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyDynamicRegistration, v.DynamicRegistration)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *DocumentOnTypeFormattingClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *DocumentOnTypeFormattingClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyDynamicRegistration {
		return dec.Bool(&v.DynamicRegistration)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DocumentOnTypeFormattingClientCapabilities) NKeys() int { return 1 }

// compile time check whether the DocumentOnTypeFormattingClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DocumentOnTypeFormattingClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*DocumentOnTypeFormattingClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *PublishDiagnosticsClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyRelatedInformation, v.RelatedInformation)
	enc.ObjectKeyOmitEmpty(keyTagSupport, v.TagSupport)
	enc.BoolKeyOmitEmpty(keyVersionSupport, v.VersionSupport)
	enc.BoolKeyOmitEmpty(keyCodeDescriptionSupport, v.CodeDescriptionSupport)
	enc.BoolKeyOmitEmpty(keyDataSupport, v.DataSupport)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *PublishDiagnosticsClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *PublishDiagnosticsClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyRelatedInformation:
		return dec.Bool(&v.RelatedInformation)
	case keyTagSupport:
		if v.TagSupport == nil {
			v.TagSupport = &PublishDiagnosticsClientCapabilitiesTagSupport{}
		}
		return dec.Object(v.TagSupport)
	case keyVersionSupport:
		return dec.Bool(&v.VersionSupport)
	case keyCodeDescriptionSupport:
		return dec.Bool(&v.CodeDescriptionSupport)
	case keyDataSupport:
		return dec.Bool(&v.DataSupport)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *PublishDiagnosticsClientCapabilities) NKeys() int { return 5 }

// compile time check whether the PublishDiagnosticsClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*PublishDiagnosticsClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*PublishDiagnosticsClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *PublishDiagnosticsClientCapabilitiesTagSupport) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKeyOmitEmpty(keyValueSet, (*DiagnosticTags)(&v.ValueSet))
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *PublishDiagnosticsClientCapabilitiesTagSupport) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *PublishDiagnosticsClientCapabilitiesTagSupport) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyValueSet {
		if v.ValueSet == nil {
			v.ValueSet = []DiagnosticTag{}
		}
		return dec.Array((*DiagnosticTags)(&v.ValueSet))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *PublishDiagnosticsClientCapabilitiesTagSupport) NKeys() int { return 1 }

// compile time check whether the CodeActionClientCapabilitiesKind implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*PublishDiagnosticsClientCapabilitiesTagSupport)(nil)
	_ gojay.UnmarshalerJSONObject = (*PublishDiagnosticsClientCapabilitiesTagSupport)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *RenameClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyDynamicRegistration, v.DynamicRegistration)
	enc.BoolKeyOmitEmpty(keyPrepareSupport, v.PrepareSupport)
	enc.Float64KeyOmitEmpty(keyPrepareSupportDefaultBehavior, float64(v.PrepareSupportDefaultBehavior))
	enc.BoolKeyOmitEmpty(keyHonorsChangeAnnotations, v.HonorsChangeAnnotations)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *RenameClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *RenameClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDynamicRegistration:
		return dec.Bool(&v.DynamicRegistration)
	case keyPrepareSupport:
		return dec.Bool(&v.PrepareSupport)
	case keyPrepareSupportDefaultBehavior:
		return dec.Float64((*float64)(&v.PrepareSupportDefaultBehavior))
	case keyHonorsChangeAnnotations:
		return dec.Bool(&v.HonorsChangeAnnotations)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *RenameClientCapabilities) NKeys() int { return 4 }

// compile time check whether the RenameClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*RenameClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*RenameClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *FoldingRangeClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyDynamicRegistration, v.DynamicRegistration)
	enc.Uint32KeyOmitEmpty(keyRangeLimit, v.RangeLimit)
	enc.BoolKeyOmitEmpty(keyLineFoldingOnly, v.LineFoldingOnly)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *FoldingRangeClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *FoldingRangeClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDynamicRegistration:
		return dec.Bool(&v.DynamicRegistration)
	case keyRangeLimit:
		return dec.Uint32(&v.RangeLimit)
	case keyLineFoldingOnly:
		return dec.Bool(&v.LineFoldingOnly)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *FoldingRangeClientCapabilities) NKeys() int { return 3 }

// compile time check whether the FoldingRangeClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*FoldingRangeClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*FoldingRangeClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *SelectionRangeClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyDynamicRegistration, v.DynamicRegistration)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *SelectionRangeClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *SelectionRangeClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyDynamicRegistration {
		return dec.Bool(&v.DynamicRegistration)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *SelectionRangeClientCapabilities) NKeys() int { return 1 }

// compile time check whether the SelectionRangeClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*SelectionRangeClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*SelectionRangeClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CallHierarchyClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyDynamicRegistration, v.DynamicRegistration)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *CallHierarchyClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *CallHierarchyClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyDynamicRegistration {
		return dec.Bool(&v.DynamicRegistration)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *CallHierarchyClientCapabilities) NKeys() int { return 1 }

// compile time check whether the CallHierarchyClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CallHierarchyClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*CallHierarchyClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *SemanticTokensClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyDynamicRegistration, v.DynamicRegistration)
	enc.ObjectKey(keyRequests, &v.Requests)
	enc.ArrayKey(keyTokenTypes, (*Strings)(&v.TokenTypes))
	enc.ArrayKey(keyTokenModifiers, (*Strings)(&v.TokenModifiers))
	enc.ArrayKey(keyFormats, (*TokenFormats)(&v.Formats))
	enc.BoolKeyOmitEmpty(keyOverlappingTokenSupport, v.OverlappingTokenSupport)
	enc.BoolKeyOmitEmpty(keyMultilineTokenSupport, v.MultilineTokenSupport)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *SemanticTokensClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *SemanticTokensClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDynamicRegistration:
		return dec.Bool(&v.DynamicRegistration)
	case keyRequests:
		return dec.Object(&v.Requests)
	case keyTokenTypes:
		return dec.Array((*Strings)(&v.TokenTypes))
	case keyTokenModifiers:
		return dec.Array((*Strings)(&v.TokenModifiers))
	case keyFormats:
		return dec.Array((*TokenFormats)(&v.Formats))
	case keyOverlappingTokenSupport:
		return dec.Bool(&v.OverlappingTokenSupport)
	case keyMultilineTokenSupport:
		return dec.Bool(&v.MultilineTokenSupport)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *SemanticTokensClientCapabilities) NKeys() int { return 7 }

// compile time check whether the SemanticTokensClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*SemanticTokensClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*SemanticTokensClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *SemanticTokensWorkspaceClientCapabilitiesRequests) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyRange, v.Range)
	enc.AddInterfaceKey(keyFull, v.Full)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *SemanticTokensWorkspaceClientCapabilitiesRequests) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *SemanticTokensWorkspaceClientCapabilitiesRequests) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyRange:
		return dec.Bool(&v.Range)
	case keyFull:
		return dec.Interface(&v.Full)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *SemanticTokensWorkspaceClientCapabilitiesRequests) NKeys() int { return 2 }

// compile time check whether the SemanticTokensWorkspaceClientCapabilitiesRequests implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*SemanticTokensWorkspaceClientCapabilitiesRequests)(nil)
	_ gojay.UnmarshalerJSONObject = (*SemanticTokensWorkspaceClientCapabilitiesRequests)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *LinkedEditingRangeClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyDynamicRegistration, v.DynamicRegistration)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *LinkedEditingRangeClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *LinkedEditingRangeClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyDynamicRegistration {
		return dec.Bool(&v.DynamicRegistration)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *LinkedEditingRangeClientCapabilities) NKeys() int { return 1 }

// compile time check whether the LinkedEditingRangeClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*LinkedEditingRangeClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*LinkedEditingRangeClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *MonikerClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyDynamicRegistration, v.DynamicRegistration)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *MonikerClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *MonikerClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyDynamicRegistration {
		return dec.Bool(&v.DynamicRegistration)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *MonikerClientCapabilities) NKeys() int { return 1 }

// compile time check whether the MonikerClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*MonikerClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*MonikerClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *WindowClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyWorkDoneProgress, v.WorkDoneProgress)
	enc.ObjectKeyOmitEmpty(keyShowMessage, v.ShowMessage)
	enc.ObjectKeyOmitEmpty(keyShowDocument, v.ShowDocument)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *WindowClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *WindowClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyWorkDoneProgress:
		return dec.Bool(&v.WorkDoneProgress)
	case keyShowMessage:
		if v.ShowMessage == nil {
			v.ShowMessage = &ShowMessageRequestClientCapabilities{}
		}
		return dec.Object(v.ShowMessage)
	case keyShowDocument:
		if v.ShowDocument == nil {
			v.ShowDocument = &ShowDocumentClientCapabilities{}
		}
		return dec.Object(v.ShowDocument)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *WindowClientCapabilities) NKeys() int { return 3 }

// compile time check whether the WindowClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*WindowClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*WindowClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ShowMessageRequestClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKeyOmitEmpty(keyMessageActionItem, v.MessageActionItem)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *ShowMessageRequestClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *ShowMessageRequestClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyMessageActionItem {
		if v.MessageActionItem == nil {
			v.MessageActionItem = &ShowMessageRequestClientCapabilitiesMessageActionItem{}
		}
		return dec.Object(v.MessageActionItem)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *ShowMessageRequestClientCapabilities) NKeys() int { return 1 }

// compile time check whether the ShowMessageRequestClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ShowMessageRequestClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*ShowMessageRequestClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ShowMessageRequestClientCapabilitiesMessageActionItem) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyAdditionalPropertiesSupport, v.AdditionalPropertiesSupport)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *ShowMessageRequestClientCapabilitiesMessageActionItem) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *ShowMessageRequestClientCapabilitiesMessageActionItem) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyAdditionalPropertiesSupport {
		return dec.Bool(&v.AdditionalPropertiesSupport)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *ShowMessageRequestClientCapabilitiesMessageActionItem) NKeys() int { return 1 }

// compile time check whether the ShowMessageRequestClientCapabilitiesMessageActionItem implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ShowMessageRequestClientCapabilitiesMessageActionItem)(nil)
	_ gojay.UnmarshalerJSONObject = (*ShowMessageRequestClientCapabilitiesMessageActionItem)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ShowDocumentClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKey(keySupport, v.Support)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *ShowDocumentClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *ShowDocumentClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keySupport {
		return dec.Bool(&v.Support)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *ShowDocumentClientCapabilities) NKeys() int { return 1 }

// compile time check whether the ShowDocumentClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ShowDocumentClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*ShowDocumentClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *GeneralClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKeyOmitEmpty(keyRegularExpressions, v.RegularExpressions)
	enc.ObjectKeyOmitEmpty(keyMarkdown, v.Markdown)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *GeneralClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *GeneralClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyRegularExpressions:
		if v.RegularExpressions == nil {
			v.RegularExpressions = &RegularExpressionsClientCapabilities{}
		}
		return dec.Object(v.RegularExpressions)
	case keyMarkdown:
		if v.Markdown == nil {
			v.Markdown = &MarkdownClientCapabilities{}
		}
		return dec.Object(v.Markdown)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *GeneralClientCapabilities) NKeys() int { return 2 }

// compile time check whether the GeneralClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*GeneralClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*GeneralClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *RegularExpressionsClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyEngine, v.Engine)
	enc.StringKeyOmitEmpty(keyVersion, v.Version)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *RegularExpressionsClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *RegularExpressionsClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyEngine:
		return dec.String(&v.Engine)
	case keyVersion:
		return dec.String(&v.Version)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *RegularExpressionsClientCapabilities) NKeys() int { return 2 }

// compile time check whether the RegularExpressionsClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*RegularExpressionsClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*RegularExpressionsClientCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *MarkdownClientCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyParser, v.Parser)
	enc.StringKeyOmitEmpty(keyVersion, v.Version)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *MarkdownClientCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *MarkdownClientCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyParser:
		return dec.String(&v.Parser)
	case keyVersion:
		return dec.String(&v.Version)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *MarkdownClientCapabilities) NKeys() int { return 2 }

// compile time check whether the MarkdownClientCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*MarkdownClientCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*MarkdownClientCapabilities)(nil)
)
