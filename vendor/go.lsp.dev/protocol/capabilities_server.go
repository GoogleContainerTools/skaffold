// SPDX-FileCopyrightText: 2021 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

import (
	"strconv"
)

// ServerCapabilities efines the capabilities provided by a language server.
type ServerCapabilities struct {
	// TextDocumentSync defines how text documents are synced. Is either a detailed structure defining each notification
	// or for backwards compatibility the TextDocumentSyncKind number.
	//
	// If omitted it defaults to TextDocumentSyncKind.None`
	TextDocumentSync interface{} `json:"textDocumentSync,omitempty"` // *TextDocumentSyncOptions | TextDocumentSyncKind

	// CompletionProvider is The server provides completion support.
	CompletionProvider *CompletionOptions `json:"completionProvider,omitempty"`

	// HoverProvider is the server provides hover support.
	HoverProvider interface{} `json:"hoverProvider,omitempty"` // TODO(zchee): bool | *HoverOptions

	// SignatureHelpProvider is the server provides signature help support.
	SignatureHelpProvider *SignatureHelpOptions `json:"signatureHelpProvider,omitempty"`

	// DeclarationProvider is the server provides Goto Declaration support.
	//
	// @since 3.14.0.
	DeclarationProvider interface{} `json:"declarationProvider,omitempty"` // TODO(zchee): bool | *DeclarationOptions | *DeclarationRegistrationOptions

	// DefinitionProvider is the server provides Goto definition support.
	DefinitionProvider interface{} `json:"definitionProvider,omitempty"` // TODO(zchee): bool | *DefinitionOptions

	// TypeDefinitionProvider is the provides Goto Type Definition support.
	//
	// @since 3.6.0.
	TypeDefinitionProvider interface{} `json:"typeDefinitionProvider,omitempty"` // TODO(zchee): bool | *TypeDefinitionOptions | *TypeDefinitionRegistrationOptions

	// ImplementationProvider is the provides Goto Implementation support.
	//
	// @since 3.6.0.
	ImplementationProvider interface{} `json:"implementationProvider,omitempty"` // TODO(zchee): bool | *ImplementationOptions | *ImplementationRegistrationOptions

	// ReferencesProvider is the server provides find references support.
	ReferencesProvider interface{} `json:"referencesProvider,omitempty"` // TODO(zchee): bool | *ReferenceOptions

	// DocumentHighlightProvider is the server provides document highlight support.
	DocumentHighlightProvider interface{} `json:"documentHighlightProvider,omitempty"` // TODO(zchee): bool | *DocumentHighlightOptions

	// DocumentSymbolProvider is the server provides document symbol support.
	DocumentSymbolProvider interface{} `json:"documentSymbolProvider,omitempty"` // TODO(zchee): bool | *DocumentSymbolOptions

	// CodeActionProvider is the server provides code actions.
	//
	// CodeActionOptions may only be specified if the client states that it supports CodeActionLiteralSupport in its
	// initial Initialize request.
	CodeActionProvider interface{} `json:"codeActionProvider,omitempty"` // TODO(zchee): bool | *CodeActionOptions

	// CodeLensProvider is the server provides code lens.
	CodeLensProvider *CodeLensOptions `json:"codeLensProvider,omitempty"`

	// The server provides document link support.
	DocumentLinkProvider *DocumentLinkOptions `json:"documentLinkProvider,omitempty"`

	// ColorProvider is the server provides color provider support.
	//
	// @since 3.6.0.
	ColorProvider interface{} `json:"colorProvider,omitempty"` // TODO(zchee): bool | *DocumentColorOptions | *DocumentColorRegistrationOptions

	// WorkspaceSymbolProvider is the server provides workspace symbol support.
	WorkspaceSymbolProvider interface{} `json:"workspaceSymbolProvider,omitempty"` // TODO(zchee): bool | *WorkspaceSymbolOptions

	// DocumentFormattingProvider is the server provides document formatting.
	DocumentFormattingProvider interface{} `json:"documentFormattingProvider,omitempty"` // TODO(zchee): bool | *DocumentFormattingOptions

	// DocumentRangeFormattingProvider is the server provides document range formatting.
	DocumentRangeFormattingProvider interface{} `json:"documentRangeFormattingProvider,omitempty"` // TODO(zchee): bool | *DocumentRangeFormattingOptions

	// DocumentOnTypeFormattingProvider is the server provides document formatting on typing.
	DocumentOnTypeFormattingProvider *DocumentOnTypeFormattingOptions `json:"documentOnTypeFormattingProvider,omitempty"`

	// RenameProvider is the server provides rename support.
	//
	// RenameOptions may only be specified if the client states that it supports PrepareSupport in its
	// initial Initialize request.
	RenameProvider interface{} `json:"renameProvider,omitempty"` // TODO(zchee): bool | *RenameOptions

	// FoldingRangeProvider is the server provides folding provider support.
	//
	// @since 3.10.0.
	FoldingRangeProvider interface{} `json:"foldingRangeProvider,omitempty"` // TODO(zchee): bool | *FoldingRangeOptions | *FoldingRangeRegistrationOptions

	// SelectionRangeProvider is the server provides selection range support.
	//
	// @since 3.15.0.
	SelectionRangeProvider interface{} `json:"selectionRangeProvider,omitempty"` // TODO(zchee): bool | *SelectionRangeOptions | *SelectionRangeRegistrationOptions

	// ExecuteCommandProvider is the server provides execute command support.
	ExecuteCommandProvider *ExecuteCommandOptions `json:"executeCommandProvider,omitempty"`

	// CallHierarchyProvider is the server provides call hierarchy support.
	//
	// @since 3.16.0.
	CallHierarchyProvider interface{} `json:"callHierarchyProvider,omitempty"` // TODO(zchee): bool | *CallHierarchyOptions | *CallHierarchyRegistrationOptions

	// LinkedEditingRangeProvider is the server provides linked editing range support.
	//
	// @since 3.16.0.
	LinkedEditingRangeProvider interface{} `json:"linkedEditingRangeProvider,omitempty"` // TODO(zchee): bool | *LinkedEditingRangeOptions | *LinkedEditingRangeRegistrationOptions

	// SemanticTokensProvider is the server provides semantic tokens support.
	//
	// @since 3.16.0.
	SemanticTokensProvider interface{} `json:"semanticTokensProvider,omitempty"` // TODO(zchee): *SemanticTokensOptions | *SemanticTokensRegistrationOptions

	// Workspace is the window specific server capabilities.
	Workspace *ServerCapabilitiesWorkspace `json:"workspace,omitempty"`

	// MonikerProvider is the server provides moniker support.
	//
	// @since 3.16.0.
	MonikerProvider interface{} `json:"monikerProvider,omitempty"` // TODO(zchee): bool | *MonikerOptions | *MonikerRegistrationOptions

	// Experimental server capabilities.
	Experimental interface{} `json:"experimental,omitempty"`
}

// TextDocumentSyncOptions TextDocumentSync options.
type TextDocumentSyncOptions struct {
	// OpenClose open and close notifications are sent to the server.
	OpenClose bool `json:"openClose,omitempty"`

	// Change notifications are sent to the server. See TextDocumentSyncKind.None, TextDocumentSyncKind.Full
	// and TextDocumentSyncKind.Incremental. If omitted it defaults to TextDocumentSyncKind.None.
	Change TextDocumentSyncKind `json:"change,omitempty"`

	// WillSave notifications are sent to the server.
	WillSave bool `json:"willSave,omitempty"`

	// WillSaveWaitUntil will save wait until requests are sent to the server.
	WillSaveWaitUntil bool `json:"willSaveWaitUntil,omitempty"`

	// Save notifications are sent to the server.
	Save *SaveOptions `json:"save,omitempty"`
}

// SaveOptions save options.
type SaveOptions struct {
	// IncludeText is the client is supposed to include the content on save.
	IncludeText bool `json:"includeText,omitempty"`
}

// TextDocumentSyncKind defines how the host (editor) should sync document changes to the language server.
type TextDocumentSyncKind float64

const (
	// TextDocumentSyncKindNone documents should not be synced at all.
	TextDocumentSyncKindNone TextDocumentSyncKind = 0

	// TextDocumentSyncKindFull documents are synced by always sending the full content
	// of the document.
	TextDocumentSyncKindFull TextDocumentSyncKind = 1

	// TextDocumentSyncKindIncremental documents are synced by sending the full content on open.
	// After that only incremental updates to the document are
	// send.
	TextDocumentSyncKindIncremental TextDocumentSyncKind = 2
)

// String implements fmt.Stringer.
func (k TextDocumentSyncKind) String() string {
	switch k {
	case TextDocumentSyncKindNone:
		return "None"
	case TextDocumentSyncKindFull:
		return "Full"
	case TextDocumentSyncKindIncremental:
		return "Incremental"
	default:
		return strconv.FormatFloat(float64(k), 'f', -10, 64)
	}
}

// CompletionOptions Completion options.
type CompletionOptions struct {
	// The server provides support to resolve additional
	// information for a completion item.
	ResolveProvider bool `json:"resolveProvider,omitempty"`

	// The characters that trigger completion automatically.
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
}

// HoverOptions option of hover provider server capabilities.
type HoverOptions struct {
	WorkDoneProgressOptions
}

// SignatureHelpOptions SignatureHelp options.
type SignatureHelpOptions struct {
	// The characters that trigger signature help
	// automatically.
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`

	// RetriggerCharacters is the slist of characters that re-trigger signature help.
	//
	// These trigger characters are only active when signature help is already
	// showing.
	// All trigger characters are also counted as re-trigger characters.
	//
	// @since 3.15.0.
	RetriggerCharacters []string `json:"retriggerCharacters,omitempty"`
}

// DeclarationOptions registration option of Declaration server capability.
//
// @since 3.15.0.
type DeclarationOptions struct {
	WorkDoneProgressOptions
}

// DeclarationRegistrationOptions registration option of Declaration server capability.
//
// @since 3.15.0.
type DeclarationRegistrationOptions struct {
	DeclarationOptions
	TextDocumentRegistrationOptions
	StaticRegistrationOptions
}

// DefinitionOptions registration option of Definition server capability.
//
// @since 3.15.0.
type DefinitionOptions struct {
	WorkDoneProgressOptions
}

// TypeDefinitionOptions registration option of TypeDefinition server capability.
//
// @since 3.15.0.
type TypeDefinitionOptions struct {
	WorkDoneProgressOptions
}

// TypeDefinitionRegistrationOptions registration option of TypeDefinition server capability.
//
// @since 3.15.0.
type TypeDefinitionRegistrationOptions struct {
	TextDocumentRegistrationOptions
	TypeDefinitionOptions
	StaticRegistrationOptions
}

// ImplementationOptions registration option of Implementation server capability.
//
// @since 3.15.0.
type ImplementationOptions struct {
	WorkDoneProgressOptions
}

// ImplementationRegistrationOptions registration option of Implementation server capability.
//
// @since 3.15.0.
type ImplementationRegistrationOptions struct {
	TextDocumentRegistrationOptions
	ImplementationOptions
	StaticRegistrationOptions
}

// ReferenceOptions registration option of Reference server capability.
type ReferenceOptions struct {
	WorkDoneProgressOptions
}

// DocumentHighlightOptions registration option of DocumentHighlight server capability.
//
// @since 3.15.0.
type DocumentHighlightOptions struct {
	WorkDoneProgressOptions
}

// DocumentSymbolOptions registration option of DocumentSymbol server capability.
//
// @since 3.15.0.
type DocumentSymbolOptions struct {
	WorkDoneProgressOptions

	// Label a human-readable string that is shown when multiple outlines trees
	// are shown for the same document.
	//
	// @since 3.16.0.
	Label string `json:"label,omitempty"`
}

// CodeActionOptions CodeAction options.
type CodeActionOptions struct {
	// CodeActionKinds that this server may return.
	//
	// The list of kinds may be generic, such as "CodeActionKind.Refactor", or the server
	// may list out every specific kind they provide.
	CodeActionKinds []CodeActionKind `json:"codeActionKinds,omitempty"`

	// ResolveProvider is the server provides support to resolve additional
	// information for a code action.
	//
	// @since 3.16.0.
	ResolveProvider bool `json:"resolveProvider,omitempty"`
}

// CodeLensOptions CodeLens options.
type CodeLensOptions struct {
	// Code lens has a resolve provider as well.
	ResolveProvider bool `json:"resolveProvider,omitempty"`
}

// DocumentLinkOptions document link options.
type DocumentLinkOptions struct {
	// ResolveProvider document links have a resolve provider as well.
	ResolveProvider bool `json:"resolveProvider,omitempty"`
}

// DocumentColorOptions registration option of DocumentColor server capability.
//
// @since 3.15.0.
type DocumentColorOptions struct {
	WorkDoneProgressOptions
}

// DocumentColorRegistrationOptions registration option of DocumentColor server capability.
//
// @since 3.15.0.
type DocumentColorRegistrationOptions struct {
	TextDocumentRegistrationOptions
	StaticRegistrationOptions
	DocumentColorOptions
}

// WorkspaceSymbolOptions registration option of WorkspaceSymbol server capability.
//
// @since 3.15.0.
type WorkspaceSymbolOptions struct {
	WorkDoneProgressOptions
}

// DocumentFormattingOptions registration option of DocumentFormatting server capability.
//
// @since 3.15.0.
type DocumentFormattingOptions struct {
	WorkDoneProgressOptions
}

// DocumentRangeFormattingOptions registration option of DocumentRangeFormatting server capability.
//
// @since 3.15.0.
type DocumentRangeFormattingOptions struct {
	WorkDoneProgressOptions
}

// DocumentOnTypeFormattingOptions format document on type options.
type DocumentOnTypeFormattingOptions struct {
	// FirstTriggerCharacter a character on which formatting should be triggered, like "}".
	FirstTriggerCharacter string `json:"firstTriggerCharacter"`

	// MoreTriggerCharacter more trigger characters.
	MoreTriggerCharacter []string `json:"moreTriggerCharacter,omitempty"`
}

// RenameOptions rename options.
type RenameOptions struct {
	// PrepareProvider renames should be checked and tested before being executed.
	PrepareProvider bool `json:"prepareProvider,omitempty"`
}

// FoldingRangeOptions registration option of FoldingRange server capability.
//
// @since 3.15.0.
type FoldingRangeOptions struct {
	WorkDoneProgressOptions
}

// FoldingRangeRegistrationOptions registration option of FoldingRange server capability.
//
// @since 3.15.0.
type FoldingRangeRegistrationOptions struct {
	TextDocumentRegistrationOptions
	FoldingRangeOptions
	StaticRegistrationOptions
}

// ExecuteCommandOptions execute command options.
type ExecuteCommandOptions struct {
	// Commands is the commands to be executed on the server
	Commands []string `json:"commands"`
}

// CallHierarchyOptions option of CallHierarchy.
//
// @since 3.16.0.
type CallHierarchyOptions struct {
	WorkDoneProgressOptions
}

// CallHierarchyRegistrationOptions registration options of CallHierarchy.
//
// @since 3.16.0.
type CallHierarchyRegistrationOptions struct {
	TextDocumentRegistrationOptions
	CallHierarchyOptions
	StaticRegistrationOptions
}

// LinkedEditingRangeOptions option of linked editing range provider server capabilities.
//
// @since 3.16.0.
type LinkedEditingRangeOptions struct {
	WorkDoneProgressOptions
}

// LinkedEditingRangeRegistrationOptions registration option of linked editing range provider server capabilities.
//
// @since 3.16.0.
type LinkedEditingRangeRegistrationOptions struct {
	TextDocumentRegistrationOptions
	LinkedEditingRangeOptions
	StaticRegistrationOptions
}

// SemanticTokensOptions option of semantic tokens provider server capabilities.
//
// @since 3.16.0.
type SemanticTokensOptions struct {
	WorkDoneProgressOptions
}

// SemanticTokensRegistrationOptions registration option of semantic tokens provider server capabilities.
//
// @since 3.16.0.
type SemanticTokensRegistrationOptions struct {
	TextDocumentRegistrationOptions
	SemanticTokensOptions
	StaticRegistrationOptions
}

// ServerCapabilitiesWorkspace specific server capabilities.
type ServerCapabilitiesWorkspace struct {
	// WorkspaceFolders is the server supports workspace folder.
	//
	// @since 3.6.0.
	WorkspaceFolders *ServerCapabilitiesWorkspaceFolders `json:"workspaceFolders,omitempty"`

	// FileOperations is the server is interested in file notifications/requests.
	//
	// @since 3.16.0.
	FileOperations *ServerCapabilitiesWorkspaceFileOperations `json:"fileOperations,omitempty"`
}

// ServerCapabilitiesWorkspaceFolders is the server supports workspace folder.
//
// @since 3.6.0.
type ServerCapabilitiesWorkspaceFolders struct {
	// Supported is the server has support for workspace folders
	Supported bool `json:"supported,omitempty"`

	// ChangeNotifications whether the server wants to receive workspace folder
	// change notifications.
	//
	// If a strings is provided the string is treated as a ID
	// under which the notification is registered on the client
	// side. The ID can be used to unregister for these events
	// using the `client/unregisterCapability` request.
	ChangeNotifications interface{} `json:"changeNotifications,omitempty"` // string | boolean
}

// ServerCapabilitiesWorkspaceFileOperations is the server is interested in file notifications/requests.
//
// @since 3.16.0.
type ServerCapabilitiesWorkspaceFileOperations struct {
	// DidCreate is the server is interested in receiving didCreateFiles
	// notifications.
	DidCreate *FileOperationRegistrationOptions `json:"didCreate,omitempty"`

	// WillCreate is the server is interested in receiving willCreateFiles requests.
	WillCreate *FileOperationRegistrationOptions `json:"willCreate,omitempty"`

	// DidRename is the server is interested in receiving didRenameFiles
	// notifications.
	DidRename *FileOperationRegistrationOptions `json:"didRename,omitempty"`

	// WillRename is the server is interested in receiving willRenameFiles requests.
	WillRename *FileOperationRegistrationOptions `json:"willRename,omitempty"`

	// DidDelete is the server is interested in receiving didDeleteFiles file
	// notifications.
	DidDelete *FileOperationRegistrationOptions `json:"didDelete,omitempty"`

	// WillDelete is the server is interested in receiving willDeleteFiles file
	// requests.
	WillDelete *FileOperationRegistrationOptions `json:"willDelete,omitempty"`
}

// FileOperationRegistrationOptions is the options to register for file operations.
//
// @since 3.16.0.
type FileOperationRegistrationOptions struct {
	// filters is the actual filters.
	Filters []FileOperationFilter `json:"filters"`
}

// MonikerOptions option of moniker provider server capabilities.
//
// @since 3.16.0.
type MonikerOptions struct {
	WorkDoneProgressOptions
}

// MonikerRegistrationOptions registration option of moniker provider server capabilities.
//
// @since 3.16.0.
type MonikerRegistrationOptions struct {
	TextDocumentRegistrationOptions
	MonikerOptions
}
