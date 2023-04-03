// SPDX-FileCopyrightText: 2021 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

//go:build gojay
// +build gojay

package protocol

import (
	"github.com/francoispqt/gojay"
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
//nolint:funlen,gocritic // TODO(zchee): fix gocritic:typeSwitchVar
func (v *ServerCapabilities) MarshalJSONObject(enc *gojay.Encoder) {
	switch v.TextDocumentSync.(type) {
	case float64: // TextDocumentSyncKind
		enc.Float64Key(keyTextDocumentSync, v.TextDocumentSync.(float64))
	case *TextDocumentSyncOptions:
		enc.ObjectKey(keyTextDocumentSync, v.TextDocumentSync.(*TextDocumentSyncOptions))
	}

	enc.ObjectKeyOmitEmpty(keyCompletionProvider, v.CompletionProvider)

	switch v.HoverProvider.(type) {
	case bool:
		enc.BoolKey(keyHoverProvider, v.HoverProvider.(bool))
	case *HoverOptions:
		enc.ObjectKey(keyHoverProvider, v.HoverProvider.(*HoverOptions))
	}

	enc.ObjectKeyOmitEmpty(keySignatureHelpProvider, v.SignatureHelpProvider)

	switch v.DeclarationProvider.(type) {
	case bool:
		enc.BoolKey(keyDeclarationProvider, v.DeclarationProvider.(bool))
	case *DeclarationOptions:
		enc.ObjectKey(keyDeclarationProvider, v.DeclarationProvider.(*DeclarationOptions))
	case *DeclarationRegistrationOptions:
		enc.ObjectKey(keyDeclarationProvider, v.DeclarationProvider.(*DeclarationRegistrationOptions))
	}

	switch v.DefinitionProvider.(type) {
	case bool:
		enc.BoolKey(keyDefinitionProvider, v.DefinitionProvider.(bool))
	case *DefinitionOptions:
		enc.ObjectKey(keyDefinitionProvider, v.DefinitionProvider.(*DefinitionOptions))
	}

	switch v.TypeDefinitionProvider.(type) {
	case bool:
		enc.BoolKey(keyTypeDefinitionProvider, v.TypeDefinitionProvider.(bool))
	case *TypeDefinitionOptions:
		enc.ObjectKey(keyTypeDefinitionProvider, v.TypeDefinitionProvider.(*TypeDefinitionOptions))
	case *TypeDefinitionRegistrationOptions:
		enc.ObjectKey(keyTypeDefinitionProvider, v.TypeDefinitionProvider.(*TypeDefinitionRegistrationOptions))
	}

	switch v.ImplementationProvider.(type) {
	case bool:
		enc.BoolKey(keyImplementationProvider, v.ImplementationProvider.(bool))
	case *ImplementationOptions:
		enc.ObjectKey(keyImplementationProvider, v.ImplementationProvider.(*ImplementationOptions))
	case *ImplementationRegistrationOptions:
		enc.ObjectKey(keyImplementationProvider, v.ImplementationProvider.(*ImplementationRegistrationOptions))
	}

	switch v.ReferencesProvider.(type) {
	case bool:
		enc.BoolKey(keyReferencesProvider, v.ReferencesProvider.(bool))
	case *ReferencesOptions:
		enc.ObjectKey(keyReferencesProvider, v.ReferencesProvider.(*ReferencesOptions))
	}

	switch v.DocumentHighlightProvider.(type) {
	case bool:
		enc.BoolKey(keyDocumentHighlightProvider, v.DocumentHighlightProvider.(bool))
	case *DocumentHighlightOptions:
		enc.ObjectKey(keyDocumentHighlightProvider, v.DocumentHighlightProvider.(*DocumentHighlightOptions))
	}

	switch v.DocumentSymbolProvider.(type) {
	case bool:
		enc.BoolKey(keyDocumentSymbolProvider, v.DocumentSymbolProvider.(bool))
	case *DocumentSymbolOptions:
		enc.ObjectKey(keyDocumentSymbolProvider, v.DocumentSymbolProvider.(*DocumentSymbolOptions))
	}

	switch v.CodeActionProvider.(type) {
	case bool:
		enc.BoolKey(keyCodeActionProvider, v.CodeActionProvider.(bool))
	case *CodeActionOptions:
		enc.ObjectKey(keyCodeActionProvider, v.CodeActionProvider.(*CodeActionOptions))
	}

	enc.ObjectKeyOmitEmpty(keyCodeLensProvider, v.CodeLensProvider)
	enc.ObjectKeyOmitEmpty(keyDocumentLinkProvider, v.DocumentLinkProvider)

	switch v.ColorProvider.(type) {
	case bool:
		enc.BoolKey(keyColorProvider, v.ColorProvider.(bool))
	case *DocumentColorOptions:
		enc.ObjectKey(keyColorProvider, v.ColorProvider.(*DocumentColorOptions))
	case *DocumentColorRegistrationOptions:
		enc.ObjectKey(keyColorProvider, v.ColorProvider.(*DocumentColorRegistrationOptions))
	}

	switch v.WorkspaceSymbolProvider.(type) {
	case bool:
		enc.BoolKey(keyWorkspaceSymbolProvider, v.WorkspaceSymbolProvider.(bool))
	case *WorkspaceSymbolOptions:
		enc.ObjectKey(keyWorkspaceSymbolProvider, v.WorkspaceSymbolProvider.(*WorkspaceSymbolOptions))
	}

	switch v.DocumentFormattingProvider.(type) {
	case bool:
		enc.BoolKey(keyDocumentFormattingProvider, v.DocumentFormattingProvider.(bool))
	case *DocumentFormattingOptions:
		enc.ObjectKey(keyDocumentFormattingProvider, v.DocumentFormattingProvider.(*DocumentFormattingOptions))
	}

	switch v.DocumentRangeFormattingProvider.(type) {
	case bool:
		enc.BoolKey(keyDocumentRangeFormattingProvider, v.DocumentRangeFormattingProvider.(bool))
	case *DocumentRangeFormattingOptions:
		enc.ObjectKey(keyDocumentRangeFormattingProvider, v.DocumentRangeFormattingProvider.(*DocumentRangeFormattingOptions))
	}

	enc.ObjectKeyOmitEmpty(keyDocumentOnTypeFormattingProvider, v.DocumentOnTypeFormattingProvider)

	switch v.RenameProvider.(type) {
	case bool:
		enc.BoolKey(keyRenameProvider, v.RenameProvider.(bool))
	case *RenameOptions:
		enc.ObjectKey(keyRenameProvider, v.RenameProvider.(*RenameOptions))
	}

	switch v.FoldingRangeProvider.(type) {
	case bool:
		enc.BoolKey(keyFoldingRangeProvider, v.FoldingRangeProvider.(bool))
	case *FoldingRangeOptions:
		enc.ObjectKey(keyFoldingRangeProvider, v.FoldingRangeProvider.(*FoldingRangeOptions))
	case *FoldingRangeRegistrationOptions:
		enc.ObjectKey(keyFoldingRangeProvider, v.FoldingRangeProvider.(*FoldingRangeRegistrationOptions))
	}

	switch v.SelectionRangeProvider.(type) {
	case bool:
		enc.BoolKey(keySelectionRangeProvider, v.SelectionRangeProvider.(bool))
	case *EnableSelectionRange:
		enc.BoolKey(keySelectionRangeProvider, bool(*v.SelectionRangeProvider.(*EnableSelectionRange)))
	case *SelectionRangeOptions:
		enc.ObjectKey(keySelectionRangeProvider, v.SelectionRangeProvider.(*SelectionRangeOptions))
	case *SelectionRangeRegistrationOptions:
		enc.ObjectKey(keySelectionRangeProvider, v.SelectionRangeProvider.(*SelectionRangeRegistrationOptions))
	}

	enc.ObjectKeyOmitEmpty(keyExecuteCommandProvider, v.ExecuteCommandProvider)

	switch v.CallHierarchyProvider.(type) {
	case bool:
		enc.BoolKey(keyCallHierarchyProvider, v.CallHierarchyProvider.(bool))
	case *CallHierarchyOptions:
		enc.ObjectKey(keyCallHierarchyProvider, v.CallHierarchyProvider.(*CallHierarchyOptions))
	case *CallHierarchyRegistrationOptions:
		enc.ObjectKey(keyCallHierarchyProvider, v.CallHierarchyProvider.(*CallHierarchyRegistrationOptions))
	}

	switch v.LinkedEditingRangeProvider.(type) {
	case bool:
		enc.BoolKey(keyLinkedEditingRangeProvider, v.LinkedEditingRangeProvider.(bool))
	case *LinkedEditingRangeOptions:
		enc.ObjectKey(keyLinkedEditingRangeProvider, v.LinkedEditingRangeProvider.(*LinkedEditingRangeOptions))
	case *LinkedEditingRangeRegistrationOptions:
		enc.ObjectKey(keyLinkedEditingRangeProvider, v.LinkedEditingRangeProvider.(*LinkedEditingRangeRegistrationOptions))
	}

	switch v.SemanticTokensProvider.(type) {
	case *SemanticTokensOptions:
		enc.ObjectKey(keySemanticTokensProvider, v.SemanticTokensProvider.(*SemanticTokensOptions))
	case *SemanticTokensRegistrationOptions:
		enc.ObjectKey(keySemanticTokensProvider, v.SemanticTokensProvider.(*SemanticTokensRegistrationOptions))
	}

	enc.ObjectKeyOmitEmpty(keyWorkspace, v.Workspace)

	switch v.MonikerProvider.(type) {
	case bool:
		enc.BoolKey(keyMonikerProvider, v.MonikerProvider.(bool))
	case *MonikerOptions:
		enc.ObjectKey(keyMonikerProvider, v.MonikerProvider.(*MonikerOptions))
	case *MonikerRegistrationOptions:
		enc.ObjectKey(keyMonikerProvider, v.MonikerProvider.(*MonikerRegistrationOptions))
	}

	enc.AddInterfaceKeyOmitEmpty(keyExperimental, v.Experimental)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *ServerCapabilities) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
//nolint:funlen
func (v *ServerCapabilities) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyTextDocumentSync:
		return dec.Interface(&v.TextDocumentSync)

	case keyCompletionProvider:
		if v.CompletionProvider == nil {
			v.CompletionProvider = &CompletionOptions{}
		}
		return dec.Object(v.CompletionProvider)

	case keyHoverProvider:
		return dec.Interface(&v.HoverProvider)

	case keySignatureHelpProvider:
		if v.SignatureHelpProvider == nil {
			v.SignatureHelpProvider = &SignatureHelpOptions{}
		}
		return dec.Object(v.SignatureHelpProvider)

	case keyDeclarationProvider:
		return dec.Interface(&v.DeclarationProvider)

	case keyDefinitionProvider:
		return dec.Interface(&v.DefinitionProvider)

	case keyTypeDefinitionProvider:
		return dec.Interface(&v.TypeDefinitionProvider)

	case keyImplementationProvider:
		return dec.Interface(&v.ImplementationProvider)

	case keyReferencesProvider:
		return dec.Interface(&v.ReferencesProvider)

	case keyDocumentHighlightProvider:
		return dec.Interface(&v.DocumentHighlightProvider)

	case keyDocumentSymbolProvider:
		return dec.Interface(&v.DocumentSymbolProvider)

	case keyCodeActionProvider:
		return dec.Interface(&v.CodeActionProvider)

	case keyCodeLensProvider:
		if v.CodeLensProvider == nil {
			v.CodeLensProvider = &CodeLensOptions{}
		}
		return dec.Object(v.CodeLensProvider)

	case keyDocumentLinkProvider:
		if v.DocumentLinkProvider == nil {
			v.DocumentLinkProvider = &DocumentLinkOptions{}
		}
		return dec.Object(v.DocumentLinkProvider)

	case keyColorProvider:
		return dec.Interface(&v.ColorProvider)

	case keyWorkspaceSymbolProvider:
		return dec.Interface(&v.WorkspaceSymbolProvider)

	case keyDocumentFormattingProvider:
		return dec.Interface(&v.DocumentFormattingProvider)

	case keyDocumentRangeFormattingProvider:
		return dec.Interface(&v.DocumentRangeFormattingProvider)

	case keyDocumentOnTypeFormattingProvider:
		if v.DocumentOnTypeFormattingProvider == nil {
			v.DocumentOnTypeFormattingProvider = &DocumentOnTypeFormattingOptions{}
		}
		return dec.Object(v.DocumentOnTypeFormattingProvider)

	case keyRenameProvider:
		return dec.Interface(&v.RenameProvider)

	case keyFoldingRangeProvider:
		return dec.Interface(&v.FoldingRangeProvider)

	case keySelectionRangeProvider:
		return dec.Interface(&v.SelectionRangeProvider)

	case keyExecuteCommandProvider:
		if v.ExecuteCommandProvider == nil {
			v.ExecuteCommandProvider = &ExecuteCommandOptions{}
		}
		return dec.Object(v.ExecuteCommandProvider)

	case keyCallHierarchyProvider:
		return dec.Interface(&v.CallHierarchyProvider)

	case keyLinkedEditingRangeProvider:
		return dec.Interface(&v.LinkedEditingRangeProvider)

	case keySemanticTokensProvider:
		return dec.Interface(&v.SemanticTokensProvider)

	case keyWorkspace:
		if v.Workspace == nil {
			v.Workspace = &ServerCapabilitiesWorkspace{}
		}
		return dec.Object(v.Workspace)

	case keyMonikerProvider:
		return dec.Interface(&v.MonikerProvider)

	case keyExperimental:
		return dec.Interface(&v.Experimental)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *ServerCapabilities) NKeys() int { return 29 }

// compile time check whether the ServerCapabilities implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ServerCapabilities)(nil)
	_ gojay.UnmarshalerJSONObject = (*ServerCapabilities)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *TextDocumentSyncOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyOpenClose, v.OpenClose)
	enc.Float64KeyOmitEmpty(keyChange, float64(v.Change))
	enc.BoolKeyOmitEmpty(keyWillSave, v.WillSave)
	enc.BoolKeyOmitEmpty(keyWillSaveWaitUntil, v.WillSaveWaitUntil)
	enc.ObjectKeyOmitEmpty(keySave, v.Save)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *TextDocumentSyncOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *TextDocumentSyncOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyOpenClose:
		return dec.Bool(&v.OpenClose)
	case keyChange:
		return dec.Float64((*float64)(&v.Change))
	case keyWillSave:
		return dec.Bool(&v.WillSave)
	case keyWillSaveWaitUntil:
		return dec.Bool(&v.WillSaveWaitUntil)
	case keySave:
		if v.Save == nil {
			v.Save = &SaveOptions{}
		}
		return dec.Object(v.Save)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *TextDocumentSyncOptions) NKeys() int { return 5 }

// compile time check whether the TextDocumentSyncOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*TextDocumentSyncOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*TextDocumentSyncOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *SaveOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyIncludeText, v.IncludeText)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *SaveOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *SaveOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyIncludeText {
		return dec.Bool(&v.IncludeText)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *SaveOptions) NKeys() int { return 1 }

// compile time check whether the SaveOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*SaveOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*SaveOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CompletionOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyResolveProvider, v.ResolveProvider)
	enc.ArrayKeyOmitEmpty(keyTriggerCharacters, (*Strings)(&v.TriggerCharacters))
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *CompletionOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *CompletionOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyResolveProvider:
		return dec.Bool(&v.ResolveProvider)
	case keyTriggerCharacters:
		return dec.Array((*Strings)(&v.TriggerCharacters))
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *CompletionOptions) NKeys() int { return 2 }

// compile time check whether the CompletionOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CompletionOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*CompletionOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *HoverOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyWorkDoneProgress, v.WorkDoneProgress)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *HoverOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *HoverOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyWorkDoneProgress {
		return dec.Bool(&v.WorkDoneProgress)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *HoverOptions) NKeys() int { return 1 }

// compile time check whether the HoverOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*HoverOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*HoverOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *SignatureHelpOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKeyOmitEmpty(keyTriggerCharacters, (*Strings)(&v.TriggerCharacters))
	enc.ArrayKeyOmitEmpty(keyRetriggerCharacters, (*Strings)(&v.RetriggerCharacters))
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *SignatureHelpOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *SignatureHelpOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyTriggerCharacters:
		return dec.Array((*Strings)(&v.TriggerCharacters))
	case keyRetriggerCharacters:
		return dec.Array((*Strings)(&v.RetriggerCharacters))
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *SignatureHelpOptions) NKeys() int { return 2 }

// compile time check whether the SignatureHelpOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*SignatureHelpOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*SignatureHelpOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DeclarationOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyWorkDoneProgress, v.WorkDoneProgress)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *DeclarationOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *DeclarationOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyWorkDoneProgress {
		return dec.Bool(&v.WorkDoneProgress)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *DeclarationOptions) NKeys() int { return 1 }

// compile time check whether the DeclarationOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DeclarationOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*DeclarationOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DeclarationRegistrationOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyWorkDoneProgress, v.WorkDoneProgress)
	enc.AddArrayKey(keyDocumentSelector, &v.DocumentSelector)
	enc.StringKeyOmitEmpty(keyID, v.ID)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *DeclarationRegistrationOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *DeclarationRegistrationOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyWorkDoneProgress:
		return dec.Bool(&v.WorkDoneProgress)
	case keyDocumentSelector:
		if v.DocumentSelector == nil {
			v.DocumentSelector = DocumentSelector{}
		}
		return dec.Array(&v.DocumentSelector)
	case keyID:
		return dec.String(&v.ID)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *DeclarationRegistrationOptions) NKeys() int { return 3 }

// compile time check whether the DeclarationRegistrationOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DeclarationRegistrationOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*DeclarationRegistrationOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DefinitionOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyWorkDoneProgress, v.WorkDoneProgress)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *DefinitionOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *DefinitionOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyWorkDoneProgress {
		return dec.Bool(&v.WorkDoneProgress)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *DefinitionOptions) NKeys() int { return 1 }

// compile time check whether the DefinitionOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DefinitionOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*DefinitionOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *TypeDefinitionOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyWorkDoneProgress, v.WorkDoneProgress)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *TypeDefinitionOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *TypeDefinitionOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyWorkDoneProgress {
		return dec.Bool(&v.WorkDoneProgress)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *TypeDefinitionOptions) NKeys() int { return 1 }

// compile time check whether the TypeDefinitionOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*TypeDefinitionOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*TypeDefinitionOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *TypeDefinitionRegistrationOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.AddArrayKey(keyDocumentSelector, &v.DocumentSelector)
	enc.BoolKeyOmitEmpty(keyWorkDoneProgress, v.WorkDoneProgress)
	enc.StringKeyOmitEmpty(keyID, v.ID)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *TypeDefinitionRegistrationOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *TypeDefinitionRegistrationOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyWorkDoneProgress:
		return dec.Bool(&v.WorkDoneProgress)
	case keyDocumentSelector:
		if v.DocumentSelector == nil {
			v.DocumentSelector = DocumentSelector{}
		}
		return dec.Array(&v.DocumentSelector)
	case keyID:
		return dec.String(&v.ID)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *TypeDefinitionRegistrationOptions) NKeys() int { return 3 }

// compile time check whether the TypeDefinitionRegistrationOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*TypeDefinitionRegistrationOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*TypeDefinitionRegistrationOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ImplementationOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyWorkDoneProgress, v.WorkDoneProgress)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *ImplementationOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *ImplementationOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyWorkDoneProgress {
		return dec.Bool(&v.WorkDoneProgress)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *ImplementationOptions) NKeys() int { return 1 }

// compile time check whether the ImplementationOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ImplementationOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*ImplementationOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ImplementationRegistrationOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.AddArrayKey(keyDocumentSelector, &v.DocumentSelector)
	enc.BoolKeyOmitEmpty(keyWorkDoneProgress, v.WorkDoneProgress)
	enc.StringKeyOmitEmpty(keyID, v.ID)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *ImplementationRegistrationOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *ImplementationRegistrationOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDocumentSelector:
		if v.DocumentSelector == nil {
			v.DocumentSelector = DocumentSelector{}
		}
		return dec.Array(&v.DocumentSelector)
	case keyWorkDoneProgress:
		return dec.Bool(&v.WorkDoneProgress)
	case keyID:
		return dec.String(&v.ID)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *ImplementationRegistrationOptions) NKeys() int { return 3 }

// compile time check whether the ImplementationRegistrationOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ImplementationRegistrationOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*ImplementationRegistrationOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ReferenceOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyWorkDoneProgress, v.WorkDoneProgress)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *ReferenceOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *ReferenceOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyWorkDoneProgress {
		return dec.Bool(&v.WorkDoneProgress)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *ReferenceOptions) NKeys() int { return 1 }

// compile time check whether the ReferenceOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ReferenceOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*ReferenceOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DocumentHighlightOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyWorkDoneProgress, v.WorkDoneProgress)
}

// IsNil returns wether the structure is nil value or not.
func (v *DocumentHighlightOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *DocumentHighlightOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyWorkDoneProgress {
		return dec.Bool(&v.WorkDoneProgress)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *DocumentHighlightOptions) NKeys() int { return 1 }

// compile time check whether the DocumentHighlightOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DocumentHighlightOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*DocumentHighlightOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DocumentSymbolOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyWorkDoneProgress, v.WorkDoneProgress)
	enc.StringKeyOmitEmpty(keyLabel, v.Label)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *DocumentSymbolOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *DocumentSymbolOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyWorkDoneProgress:
		return dec.Bool(&v.WorkDoneProgress)
	case keyLabel:
		return dec.String(&v.Label)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *DocumentSymbolOptions) NKeys() int { return 2 }

// compile time check whether the DocumentSymbolOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DocumentSymbolOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*DocumentSymbolOptions)(nil)
)

// CodeActionKinds represents a slice of CodeActionKind.
type CodeActionKinds []CodeActionKind

// compile time check whether the CodeActionKinds implements a gojay.MarshalerJSONArray and gojay.UnmarshalerJSONArray interfaces.
var (
	_ gojay.MarshalerJSONArray   = (*CodeActionKinds)(nil)
	_ gojay.UnmarshalerJSONArray = (*CodeActionKinds)(nil)
)

// MarshalJSONArray implements gojay.MarshalerJSONArray.
func (v CodeActionKinds) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range v {
		enc.String(string(v[i]))
	}
}

// IsNil implements gojay.MarshalerJSONArray.
func (v CodeActionKinds) IsNil() bool { return len(v) == 0 }

// UnmarshalJSONArray implements gojay.UnmarshalerJSONArray.
func (v *CodeActionKinds) UnmarshalJSONArray(dec *gojay.Decoder) error {
	var value CodeActionKind
	if err := dec.String((*string)(&value)); err != nil {
		return err
	}
	*v = append(*v, value)
	return nil
}

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CodeActionOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKeyOmitEmpty(keyCodeActionKinds, (*CodeActionKinds)(&v.CodeActionKinds))
	enc.BoolKeyOmitEmpty(keyResolveProvider, v.ResolveProvider)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *CodeActionOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *CodeActionOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyCodeActionKinds:
		return dec.Array((*CodeActionKinds)(&v.CodeActionKinds))
	case keyResolveProvider:
		return dec.Bool(&v.ResolveProvider)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *CodeActionOptions) NKeys() int { return 2 }

// compile time check whether the CodeActionOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CodeActionOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*CodeActionOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CodeLensOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyResolveProvider, v.ResolveProvider)
}

// IsNil returns wether the structure is nil value or not.
func (v *CodeLensOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *CodeLensOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyResolveProvider {
		return dec.Bool(&v.ResolveProvider)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *CodeLensOptions) NKeys() int { return 1 }

// compile time check whether the CodeLensOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CodeLensOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*CodeLensOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DocumentLinkOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyResolveProvider, v.ResolveProvider)
}

// IsNil returns wether the structure is nil value or not.
func (v *DocumentLinkOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *DocumentLinkOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyResolveProvider {
		return dec.Bool(&v.ResolveProvider)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *DocumentLinkOptions) NKeys() int { return 1 }

// compile time check whether the DocumentLinkOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DocumentLinkOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*DocumentLinkOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DocumentColorOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyWorkDoneProgress, v.WorkDoneProgress)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *DocumentColorOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *DocumentColorOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyWorkDoneProgress {
		return dec.Bool(&v.WorkDoneProgress)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *DocumentColorOptions) NKeys() int { return 1 }

// compile time check whether the DocumentColorOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DocumentColorOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*DocumentColorOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DocumentColorRegistrationOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.AddArrayKey(keyDocumentSelector, &v.DocumentSelector)
	enc.StringKeyOmitEmpty(keyID, v.ID)
	enc.BoolKeyOmitEmpty(keyWorkDoneProgress, v.WorkDoneProgress)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *DocumentColorRegistrationOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *DocumentColorRegistrationOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDocumentSelector:
		if v.DocumentSelector == nil {
			v.DocumentSelector = DocumentSelector{}
		}
		return dec.Array(&v.DocumentSelector)
	case keyID:
		return dec.String(&v.ID)
	case keyWorkDoneProgress:
		return dec.Bool(&v.WorkDoneProgress)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *DocumentColorRegistrationOptions) NKeys() int { return 3 }

// compile time check whether the DocumentColorRegistrationOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DocumentColorRegistrationOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*DocumentColorRegistrationOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *WorkspaceSymbolOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyWorkDoneProgress, v.WorkDoneProgress)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *WorkspaceSymbolOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *WorkspaceSymbolOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyWorkDoneProgress {
		return dec.Bool(&v.WorkDoneProgress)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *WorkspaceSymbolOptions) NKeys() int { return 1 }

// compile time check whether the WorkspaceSymbolOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*WorkspaceSymbolOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*WorkspaceSymbolOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DocumentFormattingOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyWorkDoneProgress, v.WorkDoneProgress)
}

// IsNil returns wether the structure is nil value or not.
func (v *DocumentFormattingOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *DocumentFormattingOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyWorkDoneProgress {
		return dec.Bool(&v.WorkDoneProgress)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DocumentFormattingOptions) NKeys() int { return 1 }

// compile time check whether the DocumentFormattingOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DocumentFormattingOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*DocumentFormattingOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DocumentRangeFormattingOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyWorkDoneProgress, v.WorkDoneProgress)
}

// IsNil returns wether the structure is nil value or not.
func (v *DocumentRangeFormattingOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *DocumentRangeFormattingOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyWorkDoneProgress {
		return dec.Bool(&v.WorkDoneProgress)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DocumentRangeFormattingOptions) NKeys() int { return 1 }

// compile time check whether the DocumentRangeFormattingOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DocumentRangeFormattingOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*DocumentRangeFormattingOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v DocumentOnTypeFormattingOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyFirstTriggerCharacter, v.FirstTriggerCharacter)
	enc.ArrayKeyOmitEmpty(keyMoreTriggerCharacter, (*Strings)(&v.MoreTriggerCharacter))
}

// IsNil returns wether the structure is nil value or not.
func (v *DocumentOnTypeFormattingOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *DocumentOnTypeFormattingOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyFirstTriggerCharacter:
		return dec.String(&v.FirstTriggerCharacter)
	case keyMoreTriggerCharacter:
		var values Strings
		err := dec.Array(&values)
		if err == nil && len(values) > 0 {
			v.MoreTriggerCharacter = []string(values)
		}
		return err
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DocumentOnTypeFormattingOptions) NKeys() int { return 2 }

// compile time check whether the DocumentOnTypeFormattingOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DocumentOnTypeFormattingOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*DocumentOnTypeFormattingOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *RenameOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyPrepareProvider, v.PrepareProvider)
}

// IsNil returns wether the structure is nil value or not.
func (v *RenameOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *RenameOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyPrepareProvider {
		return dec.Bool(&v.PrepareProvider)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *RenameOptions) NKeys() int { return 1 }

// compile time check whether the RenameOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*RenameOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*RenameOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *FoldingRangeOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyWorkDoneProgress, v.WorkDoneProgress)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *FoldingRangeOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *FoldingRangeOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyWorkDoneProgress {
		return dec.Bool(&v.WorkDoneProgress)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *FoldingRangeOptions) NKeys() int { return 1 }

// compile time check whether the FoldingRangeOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*FoldingRangeOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*FoldingRangeOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *FoldingRangeRegistrationOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.AddArrayKey(keyDocumentSelector, &v.DocumentSelector)
	enc.BoolKeyOmitEmpty(keyWorkDoneProgress, v.WorkDoneProgress)
	enc.StringKeyOmitEmpty(keyID, v.ID)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *FoldingRangeRegistrationOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *FoldingRangeRegistrationOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDocumentSelector:
		if v.DocumentSelector == nil {
			v.DocumentSelector = DocumentSelector{}
		}
		return dec.Array(&v.DocumentSelector)
	case keyWorkDoneProgress:
		return dec.Bool(&v.WorkDoneProgress)
	case keyID:
		return dec.String(&v.ID)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *FoldingRangeRegistrationOptions) NKeys() int { return 4 }

// compile time check whether the FoldingRangeRegistrationOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*FoldingRangeRegistrationOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*FoldingRangeRegistrationOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ExecuteCommandOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.AddArrayKey(keyCommands, (*Strings)(&v.Commands))
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *ExecuteCommandOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *ExecuteCommandOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyCommands {
		return dec.Array((*Strings)(&v.Commands))
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *ExecuteCommandOptions) NKeys() int { return 1 }

// compile time check whether the ExecuteCommandOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ExecuteCommandOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*ExecuteCommandOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CallHierarchyOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyWorkDoneProgress, v.WorkDoneProgress)
}

// IsNil returns wether the structure is nil value or not.
func (v *CallHierarchyOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *CallHierarchyOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyWorkDoneProgress {
		return dec.Bool(&v.WorkDoneProgress)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *CallHierarchyOptions) NKeys() int { return 1 }

// compile time check whether the CallHierarchyOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CallHierarchyOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*CallHierarchyOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *CallHierarchyRegistrationOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.AddArrayKey(keyDocumentSelector, &v.DocumentSelector)
	enc.BoolKeyOmitEmpty(keyWorkDoneProgress, v.WorkDoneProgress)
	enc.StringKeyOmitEmpty(keyID, v.ID)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *CallHierarchyRegistrationOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *CallHierarchyRegistrationOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDocumentSelector:
		if v.DocumentSelector == nil {
			v.DocumentSelector = DocumentSelector{}
		}
		return dec.Array(&v.DocumentSelector)
	case keyWorkDoneProgress:
		return dec.Bool(&v.WorkDoneProgress)
	case keyID:
		return dec.String(&v.ID)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *CallHierarchyRegistrationOptions) NKeys() int { return 3 }

// compile time check whether the CallHierarchyRegistrationOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*CallHierarchyRegistrationOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*CallHierarchyRegistrationOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *LinkedEditingRangeOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyWorkDoneProgress, v.WorkDoneProgress)
}

// IsNil returns wether the structure is nil value or not.
func (v *LinkedEditingRangeOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *LinkedEditingRangeOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyWorkDoneProgress {
		return dec.Bool(&v.WorkDoneProgress)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *LinkedEditingRangeOptions) NKeys() int { return 1 }

// compile time check whether the LinkedEditingRangeOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*LinkedEditingRangeOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*LinkedEditingRangeOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *LinkedEditingRangeRegistrationOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.AddArrayKey(keyDocumentSelector, &v.DocumentSelector)
	enc.BoolKeyOmitEmpty(keyWorkDoneProgress, v.WorkDoneProgress)
	enc.StringKeyOmitEmpty(keyID, v.ID)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *LinkedEditingRangeRegistrationOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *LinkedEditingRangeRegistrationOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDocumentSelector:
		if v.DocumentSelector == nil {
			v.DocumentSelector = DocumentSelector{}
		}
		return dec.Array(&v.DocumentSelector)
	case keyWorkDoneProgress:
		return dec.Bool(&v.WorkDoneProgress)
	case keyID:
		return dec.String(&v.ID)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *LinkedEditingRangeRegistrationOptions) NKeys() int { return 3 }

// compile time check whether the LinkedEditingRangeRegistrationOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*LinkedEditingRangeRegistrationOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*LinkedEditingRangeRegistrationOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *SemanticTokensOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyWorkDoneProgress, v.WorkDoneProgress)
}

// IsNil returns wether the structure is nil value or not.
func (v *SemanticTokensOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *SemanticTokensOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyWorkDoneProgress {
		return dec.Bool(&v.WorkDoneProgress)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *SemanticTokensOptions) NKeys() int { return 1 }

// compile time check whether the SemanticTokensOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*SemanticTokensOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*SemanticTokensOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *SemanticTokensRegistrationOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.AddArrayKey(keyDocumentSelector, &v.DocumentSelector)
	enc.BoolKeyOmitEmpty(keyWorkDoneProgress, v.WorkDoneProgress)
	enc.StringKeyOmitEmpty(keyID, v.ID)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *SemanticTokensRegistrationOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *SemanticTokensRegistrationOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDocumentSelector:
		if v.DocumentSelector == nil {
			v.DocumentSelector = DocumentSelector{}
		}
		return dec.Array(&v.DocumentSelector)
	case keyWorkDoneProgress:
		return dec.Bool(&v.WorkDoneProgress)
	case keyID:
		return dec.String(&v.ID)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *SemanticTokensRegistrationOptions) NKeys() int { return 3 }

// compile time check whether the SemanticTokensRegistrationOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*SemanticTokensRegistrationOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*SemanticTokensRegistrationOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ServerCapabilitiesWorkspace) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKeyOmitEmpty(keyWorkspaceFolders, v.WorkspaceFolders)
	enc.ObjectKeyOmitEmpty(keyFileOperations, v.FileOperations)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *ServerCapabilitiesWorkspace) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *ServerCapabilitiesWorkspace) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyWorkspaceFolders:
		if v.WorkspaceFolders == nil {
			v.WorkspaceFolders = &ServerCapabilitiesWorkspaceFolders{}
		}
		return dec.Object(v.WorkspaceFolders)
	case keyFileOperations:
		if v.FileOperations == nil {
			v.FileOperations = &ServerCapabilitiesWorkspaceFileOperations{}
		}
		return dec.Object(v.FileOperations)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *ServerCapabilitiesWorkspace) NKeys() int { return 2 }

// compile time check whether the ServerCapabilitiesWorkspace implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ServerCapabilitiesWorkspace)(nil)
	_ gojay.UnmarshalerJSONObject = (*ServerCapabilitiesWorkspace)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ServerCapabilitiesWorkspaceFolders) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKey(keySupported, v.Supported)
	enc.AddInterfaceKeyOmitEmpty(keyChangeNotifications, v.ChangeNotifications)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *ServerCapabilitiesWorkspaceFolders) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *ServerCapabilitiesWorkspaceFolders) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keySupported:
		return dec.Bool(&v.Supported)
	case keyChangeNotifications:
		return dec.Interface(&v.ChangeNotifications)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *ServerCapabilitiesWorkspaceFolders) NKeys() int { return 2 }

// compile time check whether the ServerCapabilitiesWorkspaceFolders implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ServerCapabilitiesWorkspaceFolders)(nil)
	_ gojay.UnmarshalerJSONObject = (*ServerCapabilitiesWorkspaceFolders)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ServerCapabilitiesWorkspaceFileOperations) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKeyOmitEmpty(keyDidCreate, v.DidCreate)
	enc.ObjectKeyOmitEmpty(keyWillCreate, v.WillCreate)
	enc.ObjectKeyOmitEmpty(keyDidRename, v.DidRename)
	enc.ObjectKeyOmitEmpty(keyWillRename, v.WillRename)
	enc.ObjectKeyOmitEmpty(keyDidDelete, v.DidDelete)
	enc.ObjectKeyOmitEmpty(keyWillDelete, v.WillDelete)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *ServerCapabilitiesWorkspaceFileOperations) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *ServerCapabilitiesWorkspaceFileOperations) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDidCreate:
		if v.DidCreate == nil {
			v.DidCreate = &FileOperationRegistrationOptions{}
		}
		return dec.Object(v.DidCreate)
	case keyWillCreate:
		if v.WillCreate == nil {
			v.WillCreate = &FileOperationRegistrationOptions{}
		}
		return dec.Object(v.WillCreate)
	case keyDidRename:
		if v.DidRename == nil {
			v.DidRename = &FileOperationRegistrationOptions{}
		}
		return dec.Object(v.DidRename)
	case keyWillRename:
		if v.WillRename == nil {
			v.WillRename = &FileOperationRegistrationOptions{}
		}
		return dec.Object(v.WillRename)
	case keyDidDelete:
		if v.DidDelete == nil {
			v.DidDelete = &FileOperationRegistrationOptions{}
		}
		return dec.Object(v.DidDelete)
	case keyWillDelete:
		if v.WillDelete == nil {
			v.WillDelete = &FileOperationRegistrationOptions{}
		}
		return dec.Object(v.WillDelete)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *ServerCapabilitiesWorkspaceFileOperations) NKeys() int { return 6 }

// compile time check whether the ServerCapabilitiesWorkspaceFileOperations implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ServerCapabilitiesWorkspaceFileOperations)(nil)
	_ gojay.UnmarshalerJSONObject = (*ServerCapabilitiesWorkspaceFileOperations)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *FileOperationRegistrationOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKey(keyFilters, FileOperationFilters(v.Filters))
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *FileOperationRegistrationOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *FileOperationRegistrationOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyFilters {
		return dec.Array((*FileOperationFilters)(&v.Filters))
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *FileOperationRegistrationOptions) NKeys() int { return 1 }

// compile time check whether the FileOperationRegistrationOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*FileOperationRegistrationOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*FileOperationRegistrationOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *MonikerOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKeyOmitEmpty(keyWorkDoneProgress, v.WorkDoneProgress)
}

// IsNil returns wether the structure is nil value or not.
func (v *MonikerOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *MonikerOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyWorkDoneProgress {
		return dec.Bool(&v.WorkDoneProgress)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *MonikerOptions) NKeys() int { return 1 }

// compile time check whether the MonikerOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*MonikerOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*MonikerOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *MonikerRegistrationOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.AddArrayKey(keyDocumentSelector, &v.DocumentSelector)
	enc.BoolKeyOmitEmpty(keyWorkDoneProgress, v.WorkDoneProgress)
}

// IsNil implements gojay.MarshalerJSONObject.
func (v *MonikerRegistrationOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject.
func (v *MonikerRegistrationOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyDocumentSelector:
		if v.DocumentSelector == nil {
			v.DocumentSelector = DocumentSelector{}
		}
		return dec.Array(&v.DocumentSelector)
	case keyWorkDoneProgress:
		return dec.Bool(&v.WorkDoneProgress)
	}
	return nil
}

// NKeys implements gojay.UnmarshalerJSONObject.
func (v *MonikerRegistrationOptions) NKeys() int { return 2 }

// compile time check whether the MonikerRegistrationOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*MonikerRegistrationOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*MonikerRegistrationOptions)(nil)
)
