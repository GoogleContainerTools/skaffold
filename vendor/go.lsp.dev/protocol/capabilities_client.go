// SPDX-FileCopyrightText: 2021 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

import "strconv"

// ClientCapabilities now define capabilities for dynamic registration, workspace and text document features
// the client supports.
//
// The experimental can be used to pass experimental capabilities under development.
//
// For future compatibility a ClientCapabilities object literal can have more properties set than currently defined.
// Servers receiving a ClientCapabilities object literal with unknown properties should ignore these properties.
//
// A missing property should be interpreted as an absence of the capability.
// If a missing property normally defines sub properties, all missing sub properties should be interpreted
// as an absence of the corresponding capability.
type ClientCapabilities struct {
	// Workspace specific client capabilities.
	Workspace *WorkspaceClientCapabilities `json:"workspace,omitempty"`

	// TextDocument specific client capabilities.
	TextDocument *TextDocumentClientCapabilities `json:"textDocument,omitempty"`

	// Window specific client capabilities.
	Window *WindowClientCapabilities `json:"window,omitempty"`

	// General client capabilities.
	//
	// @since 3.16.0.
	General *GeneralClientCapabilities `json:"general,omitempty"`

	// Experimental client capabilities.
	Experimental interface{} `json:"experimental,omitempty"`
}

// WorkspaceClientCapabilities Workspace specific client capabilities.
type WorkspaceClientCapabilities struct {
	// The client supports applying batch edits to the workspace by supporting
	// the request "workspace/applyEdit".
	ApplyEdit bool `json:"applyEdit,omitempty"`

	// WorkspaceEdit capabilities specific to `WorkspaceEdit`s.
	WorkspaceEdit *WorkspaceClientCapabilitiesWorkspaceEdit `json:"workspaceEdit,omitempty"`

	// DidChangeConfiguration capabilities specific to the `workspace/didChangeConfiguration` notification.
	DidChangeConfiguration *DidChangeConfigurationWorkspaceClientCapabilities `json:"didChangeConfiguration,omitempty"`

	// DidChangeWatchedFiles capabilities specific to the `workspace/didChangeWatchedFiles` notification.
	DidChangeWatchedFiles *DidChangeWatchedFilesWorkspaceClientCapabilities `json:"didChangeWatchedFiles,omitempty"`

	// Symbol capabilities specific to the "workspace/symbol" request.
	Symbol *WorkspaceSymbolClientCapabilities `json:"symbol,omitempty"`

	// ExecuteCommand capabilities specific to the "workspace/executeCommand" request.
	ExecuteCommand *ExecuteCommandClientCapabilities `json:"executeCommand,omitempty"`

	// WorkspaceFolders is the client has support for workspace folders.
	//
	// @since 3.6.0.
	WorkspaceFolders bool `json:"workspaceFolders,omitempty"`

	// Configuration is the client supports "workspace/configuration" requests.
	//
	// @since 3.6.0.
	Configuration bool `json:"configuration,omitempty"`

	// SemanticTokens is the capabilities specific to the semantic token requests scoped to the
	// workspace.
	//
	// @since 3.16.0.
	SemanticTokens *SemanticTokensWorkspaceClientCapabilities `json:"semanticTokens,omitempty"`

	// CodeLens is the Capabilities specific to the code lens requests scoped to the
	// workspace.
	//
	// @since 3.16.0.
	CodeLens *CodeLensWorkspaceClientCapabilities `json:"codeLens,omitempty"`

	// FileOperations is the client has support for file requests/notifications.
	//
	// @since 3.16.0.
	FileOperations *WorkspaceClientCapabilitiesFileOperations `json:"fileOperations,omitempty"`
}

// WorkspaceClientCapabilitiesWorkspaceEdit capabilities specific to "WorkspaceEdit"s.
type WorkspaceClientCapabilitiesWorkspaceEdit struct {
	// DocumentChanges is the client supports versioned document changes in `WorkspaceEdit`s
	DocumentChanges bool `json:"documentChanges,omitempty"`

	// FailureHandling is the failure handling strategy of a client if applying the workspace edit
	// fails.
	//
	// Mostly FailureHandlingKind.
	FailureHandling string `json:"failureHandling,omitempty"`

	// ResourceOperations is the resource operations the client supports. Clients should at least
	// support "create", "rename" and "delete" files and folders.
	ResourceOperations []string `json:"resourceOperations,omitempty"`

	// NormalizesLineEndings whether the client normalizes line endings to the client specific
	// setting.
	// If set to `true` the client will normalize line ending characters
	// in a workspace edit to the client specific new line character(s).
	//
	// @since 3.16.0.
	NormalizesLineEndings bool `json:"normalizesLineEndings,omitempty"`

	// ChangeAnnotationSupport whether the client in general supports change annotations on text edits,
	// create file, rename file and delete file changes.
	//
	// @since 3.16.0.
	ChangeAnnotationSupport *WorkspaceClientCapabilitiesWorkspaceEditChangeAnnotationSupport `json:"changeAnnotationSupport,omitempty"`
}

// FailureHandlingKind is the kind of failure handling .
type FailureHandlingKind string

const (
	// FailureHandlingKindAbort applying the workspace change is simply aborted if one of the changes provided
	// fails. All operations executed before the failing operation stay executed.
	FailureHandlingKindAbort FailureHandlingKind = "abort"

	// FailureHandlingKindTransactional all operations are executed transactional. That means they either all
	// succeed or no changes at all are applied to the workspace.
	FailureHandlingKindTransactional FailureHandlingKind = "transactional"

	// FailureHandlingKindTextOnlyTransactional if the workspace edit contains only textual file changes they are executed transactional.
	// If resource changes (create, rename or delete file) are part of the change the failure
	// handling strategy is abort.
	FailureHandlingKindTextOnlyTransactional FailureHandlingKind = "textOnlyTransactional"

	// FailureHandlingKindUndo the client tries to undo the operations already executed. But there is no
	// guarantee that this is succeeding.
	FailureHandlingKindUndo FailureHandlingKind = "undo"
)

// WorkspaceClientCapabilitiesWorkspaceEditChangeAnnotationSupport is the ChangeAnnotationSupport of WorkspaceClientCapabilitiesWorkspaceEdit.
//
// @since 3.16.0.
type WorkspaceClientCapabilitiesWorkspaceEditChangeAnnotationSupport struct {
	// GroupsOnLabel whether the client groups edits with equal labels into tree nodes,
	// for instance all edits labeled with "Changes in Strings" would
	// be a tree node.
	GroupsOnLabel bool `json:"groupsOnLabel,omitempty"`
}

// DidChangeConfigurationWorkspaceClientCapabilities capabilities specific to the "workspace/didChangeConfiguration" notification.
//
// @since 3.16.0.
type DidChangeConfigurationWorkspaceClientCapabilities struct {
	// DynamicRegistration whether the did change configuration notification supports dynamic registration.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// DidChangeWatchedFilesWorkspaceClientCapabilities capabilities specific to the "workspace/didChangeWatchedFiles" notification.
//
// @since 3.16.0.
type DidChangeWatchedFilesWorkspaceClientCapabilities struct {
	// Did change watched files notification supports dynamic registration.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// WorkspaceSymbolClientCapabilities capabilities specific to the `workspace/symbol` request.
//
// WorkspaceSymbolClientCapabilities is the workspace symbol request is sent from the client to the server to
// list project-wide symbols matching the query string.
type WorkspaceSymbolClientCapabilities struct {
	// DynamicRegistration is the Symbol request supports dynamic registration.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`

	// SymbolKindCapabilities is the specific capabilities for the SymbolKindCapabilities in the "workspace/symbol" request.
	SymbolKind *SymbolKindCapabilities `json:"symbolKind,omitempty"`

	// TagSupport is the client supports tags on `SymbolInformation`.
	// Clients supporting tags have to handle unknown tags gracefully.
	//
	// @since 3.16.0
	TagSupport *TagSupportCapabilities `json:"tagSupport,omitempty"`
}

type SymbolKindCapabilities struct {
	// ValueSet is the symbol kind values the client supports. When this
	// property exists the client also guarantees that it will
	// handle values outside its set gracefully and falls back
	// to a default value when unknown.
	//
	// If this property is not present the client only supports
	// the symbol kinds from `File` to `Array` as defined in
	// the initial version of the protocol.
	ValueSet []SymbolKind `json:"valueSet,omitempty"`
}

type TagSupportCapabilities struct {
	// ValueSet is the tags supported by the client.
	ValueSet []SymbolTag `json:"valueSet,omitempty"`
}

// ExecuteCommandClientCapabilities capabilities specific to the "workspace/executeCommand" request.
type ExecuteCommandClientCapabilities struct {
	// DynamicRegistration Execute command supports dynamic registration.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// SemanticTokensWorkspaceClientCapabilities capabilities specific to the "workspace/semanticToken" request.
//
// @since 3.16.0.
type SemanticTokensWorkspaceClientCapabilities struct {
	// RefreshSupport whether the client implementation supports a refresh request sent from
	// the server to the client.
	//
	// Note that this event is global and will force the client to refresh all
	// semantic tokens currently shown. It should be used with absolute care
	// and is useful for situation where a server for example detect a project
	// wide change that requires such a calculation.
	RefreshSupport bool `json:"refreshSupport,omitempty"`
}

// CodeLensWorkspaceClientCapabilities capabilities specific to the "workspace/codeLens" request.
//
// @since 3.16.0.
type CodeLensWorkspaceClientCapabilities struct {
	// RefreshSupport whether the client implementation supports a refresh request sent from the
	// server to the client.
	//
	// Note that this event is global and will force the client to refresh all
	// code lenses currently shown. It should be used with absolute care and is
	// useful for situation where a server for example detect a project wide
	// change that requires such a calculation.
	RefreshSupport bool `json:"refreshSupport,omitempty"`
}

// WorkspaceClientCapabilitiesFileOperations capabilities specific to the fileOperations.
//
// @since 3.16.0.
type WorkspaceClientCapabilitiesFileOperations struct {
	// DynamicRegistration whether the client supports dynamic registration for file
	// requests/notifications.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`

	// DidCreate is the client has support for sending didCreateFiles notifications.
	DidCreate bool `json:"didCreate,omitempty"`

	// WillCreate is the client has support for sending willCreateFiles requests.
	WillCreate bool `json:"willCreate,omitempty"`

	// DidRename is the client has support for sending didRenameFiles notifications.
	DidRename bool `json:"didRename,omitempty"`

	// WillRename is the client has support for sending willRenameFiles requests.
	WillRename bool `json:"willRename,omitempty"`

	// DidDelete is the client has support for sending didDeleteFiles notifications.
	DidDelete bool `json:"didDelete,omitempty"`

	// WillDelete is the client has support for sending willDeleteFiles requests.
	WillDelete bool `json:"willDelete,omitempty"`
}

// TextDocumentClientCapabilities Text document specific client capabilities.
type TextDocumentClientCapabilities struct {
	// Synchronization defines which synchronization capabilities the client supports.
	Synchronization *TextDocumentSyncClientCapabilities `json:"synchronization,omitempty"`

	// Completion Capabilities specific to the "textDocument/completion".
	Completion *CompletionTextDocumentClientCapabilities `json:"completion,omitempty"`

	// Hover capabilities specific to the "textDocument/hover".
	Hover *HoverTextDocumentClientCapabilities `json:"hover,omitempty"`

	// SignatureHelp capabilities specific to the "textDocument/signatureHelp".
	SignatureHelp *SignatureHelpTextDocumentClientCapabilities `json:"signatureHelp,omitempty"`

	// Declaration capabilities specific to the "textDocument/declaration".
	Declaration *DeclarationTextDocumentClientCapabilities `json:"declaration,omitempty"`

	// Definition capabilities specific to the "textDocument/definition".
	//
	// @since 3.14.0.
	Definition *DefinitionTextDocumentClientCapabilities `json:"definition,omitempty"`

	// TypeDefinition capabilities specific to the "textDocument/typeDefinition".
	//
	// @since 3.6.0.
	TypeDefinition *TypeDefinitionTextDocumentClientCapabilities `json:"typeDefinition,omitempty"`

	// Implementation capabilities specific to the "textDocument/implementation".
	//
	// @since 3.6.0.
	Implementation *ImplementationTextDocumentClientCapabilities `json:"implementation,omitempty"`

	// References capabilities specific to the "textDocument/references".
	References *ReferencesTextDocumentClientCapabilities `json:"references,omitempty"`

	// DocumentHighlight capabilities specific to the "textDocument/documentHighlight".
	DocumentHighlight *DocumentHighlightClientCapabilities `json:"documentHighlight,omitempty"`

	// DocumentSymbol capabilities specific to the "textDocument/documentSymbol".
	DocumentSymbol *DocumentSymbolClientCapabilities `json:"documentSymbol,omitempty"`

	// CodeAction capabilities specific to the "textDocument/codeAction".
	CodeAction *CodeActionClientCapabilities `json:"codeAction,omitempty"`

	// CodeLens capabilities specific to the "textDocument/codeLens".
	CodeLens *CodeLensClientCapabilities `json:"codeLens,omitempty"`

	// DocumentLink capabilities specific to the "textDocument/documentLink".
	DocumentLink *DocumentLinkClientCapabilities `json:"documentLink,omitempty"`

	// ColorProvider capabilities specific to the "textDocument/documentColor" and the
	// "textDocument/colorPresentation" request.
	//
	// @since 3.6.0.
	ColorProvider *DocumentColorClientCapabilities `json:"colorProvider,omitempty"`

	// Formatting Capabilities specific to the "textDocument/formatting" request.
	Formatting *DocumentFormattingClientCapabilities `json:"formatting,omitempty"`

	// RangeFormatting Capabilities specific to the "textDocument/rangeFormatting" request.
	RangeFormatting *DocumentRangeFormattingClientCapabilities `json:"rangeFormatting,omitempty"`

	// OnTypeFormatting Capabilities specific to the "textDocument/onTypeFormatting" request.
	OnTypeFormatting *DocumentOnTypeFormattingClientCapabilities `json:"onTypeFormatting,omitempty"`

	// PublishDiagnostics capabilities specific to "textDocument/publishDiagnostics".
	PublishDiagnostics *PublishDiagnosticsClientCapabilities `json:"publishDiagnostics,omitempty"`

	// Rename capabilities specific to the "textDocument/rename".
	Rename *RenameClientCapabilities `json:"rename,omitempty"`

	// FoldingRange capabilities specific to "textDocument/foldingRange" requests.
	//
	// @since 3.10.0.
	FoldingRange *FoldingRangeClientCapabilities `json:"foldingRange,omitempty"`

	// SelectionRange capabilities specific to "textDocument/selectionRange" requests.
	//
	// @since 3.15.0.
	SelectionRange *SelectionRangeClientCapabilities `json:"selectionRange,omitempty"`

	// CallHierarchy capabilities specific to the various call hierarchy requests.
	//
	// @since 3.16.0.
	CallHierarchy *CallHierarchyClientCapabilities `json:"callHierarchy,omitempty"`

	// SemanticTokens capabilities specific to the various semantic token requests.
	//
	// @since 3.16.0.
	SemanticTokens *SemanticTokensClientCapabilities `json:"semanticTokens,omitempty"`

	// LinkedEditingRange capabilities specific to the "textDocument/linkedEditingRange" request.
	//
	// @since 3.16.0.
	LinkedEditingRange *LinkedEditingRangeClientCapabilities `json:"linkedEditingRange,omitempty"`

	// Moniker capabilities specific to the "textDocument/moniker" request.
	//
	// @since 3.16.0.
	Moniker *MonikerClientCapabilities `json:"moniker,omitempty"`
}

// TextDocumentSyncClientCapabilities defines which synchronization capabilities the client supports.
type TextDocumentSyncClientCapabilities struct {
	// DynamicRegistration whether text document synchronization supports dynamic registration.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`

	// WillSave is the client supports sending will save notifications.
	WillSave bool `json:"willSave,omitempty"`

	// WillSaveWaitUntil is the client supports sending a will save request and
	// waits for a response providing text edits which will
	// be applied to the document before it is saved.
	WillSaveWaitUntil bool `json:"willSaveWaitUntil,omitempty"`

	// DidSave is the client supports did save notifications.
	DidSave bool `json:"didSave,omitempty"`
}

// CompletionTextDocumentClientCapabilities Capabilities specific to the "textDocument/completion".
type CompletionTextDocumentClientCapabilities struct {
	// Whether completion supports dynamic registration.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`

	// The client supports the following `CompletionItem` specific
	// capabilities.
	CompletionItem *CompletionTextDocumentClientCapabilitiesItem `json:"completionItem,omitempty"`

	CompletionItemKind *CompletionTextDocumentClientCapabilitiesItemKind `json:"completionItemKind,omitempty"`

	// ContextSupport is the client supports to send additional context information for a
	// `textDocument/completion` request.
	ContextSupport bool `json:"contextSupport,omitempty"`
}

// CompletionTextDocumentClientCapabilitiesItem is the client supports the following "CompletionItem" specific
// capabilities.
type CompletionTextDocumentClientCapabilitiesItem struct {
	// SnippetSupport client supports snippets as insert text.
	//
	// A snippet can define tab stops and placeholders with `$1`, `$2`
	// and `${3:foo}`. `$0` defines the final tab stop, it defaults to
	// the end of the snippet. Placeholders with equal identifiers are linked,
	// that is typing in one will update others too.
	SnippetSupport bool `json:"snippetSupport,omitempty"`

	// CommitCharactersSupport client supports commit characters on a completion item.
	CommitCharactersSupport bool `json:"commitCharactersSupport,omitempty"`

	// DocumentationFormat client supports the follow content formats for the documentation
	// property. The order describes the preferred format of the client.
	DocumentationFormat []MarkupKind `json:"documentationFormat,omitempty"`

	// DeprecatedSupport client supports the deprecated property on a completion item.
	DeprecatedSupport bool `json:"deprecatedSupport,omitempty"`

	// PreselectSupport client supports the preselect property on a completion item.
	PreselectSupport bool `json:"preselectSupport,omitempty"`

	// TagSupport is the client supports the tag property on a completion item.
	//
	// Clients supporting tags have to handle unknown tags gracefully.
	// Clients especially need to preserve unknown tags when sending
	// a completion item back to the server in a resolve call.
	//
	// @since 3.15.0.
	TagSupport *CompletionTextDocumentClientCapabilitiesItemTagSupport `json:"tagSupport,omitempty"`

	// InsertReplaceSupport client supports insert replace edit to control different behavior if
	// a completion item is inserted in the text or should replace text.
	//
	// @since 3.16.0.
	InsertReplaceSupport bool `json:"insertReplaceSupport,omitempty"`

	// ResolveSupport indicates which properties a client can resolve lazily on a
	// completion item. Before version 3.16.0 only the predefined properties
	// `documentation` and `details` could be resolved lazily.
	//
	// @since 3.16.0.
	ResolveSupport *CompletionTextDocumentClientCapabilitiesItemResolveSupport `json:"resolveSupport,omitempty"`

	// InsertTextModeSupport is the client supports the `insertTextMode` property on
	// a completion item to override the whitespace handling mode
	// as defined by the client (see `insertTextMode`).
	//
	// @since 3.16.0.
	InsertTextModeSupport *CompletionTextDocumentClientCapabilitiesItemInsertTextModeSupport `json:"insertTextModeSupport,omitempty"`
}

// CompletionTextDocumentClientCapabilitiesItemTagSupport specific capabilities for the "TagSupport" in the "textDocument/completion" request.
//
// @since 3.15.0.
type CompletionTextDocumentClientCapabilitiesItemTagSupport struct {
	// ValueSet is the tags supported by the client.
	//
	// @since 3.15.0.
	ValueSet []CompletionItemTag `json:"valueSet,omitempty"`
}

// CompletionTextDocumentClientCapabilitiesItemResolveSupport specific capabilities for the ResolveSupport in the CompletionTextDocumentClientCapabilitiesItem.
//
// @since 3.16.0.
type CompletionTextDocumentClientCapabilitiesItemResolveSupport struct {
	// Properties is the properties that a client can resolve lazily.
	Properties []string `json:"properties"`
}

// CompletionTextDocumentClientCapabilitiesItemInsertTextModeSupport specific capabilities for the InsertTextModeSupport in the CompletionTextDocumentClientCapabilitiesItem.
//
// @since 3.16.0.
type CompletionTextDocumentClientCapabilitiesItemInsertTextModeSupport struct {
	// ValueSet is the tags supported by the client.
	//
	// @since 3.16.0.
	ValueSet []InsertTextMode `json:"valueSet,omitempty"`
}

// CompletionTextDocumentClientCapabilitiesItemKind specific capabilities for the "CompletionItemKind" in the "textDocument/completion" request.
type CompletionTextDocumentClientCapabilitiesItemKind struct {
	// The completion item kind values the client supports. When this
	// property exists the client also guarantees that it will
	// handle values outside its set gracefully and falls back
	// to a default value when unknown.
	//
	// If this property is not present the client only supports
	// the completion items kinds from `Text` to `Reference` as defined in
	// the initial version of the protocol.
	//
	ValueSet []CompletionItemKind `json:"valueSet,omitempty"`
}

// HoverTextDocumentClientCapabilities capabilities specific to the "textDocument/hover".
type HoverTextDocumentClientCapabilities struct {
	// DynamicRegistration whether hover supports dynamic registration.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`

	// ContentFormat is the client supports the follow content formats for the content
	// property. The order describes the preferred format of the client.
	ContentFormat []MarkupKind `json:"contentFormat,omitempty"`
}

// SignatureHelpTextDocumentClientCapabilities capabilities specific to the "textDocument/signatureHelp".
type SignatureHelpTextDocumentClientCapabilities struct {
	// DynamicRegistration whether signature help supports dynamic registration.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`

	// SignatureInformation is the client supports the following "SignatureInformation"
	// specific properties.
	SignatureInformation *TextDocumentClientCapabilitiesSignatureInformation `json:"signatureInformation,omitempty"`

	// ContextSupport is the client supports to send additional context information for a "textDocument/signatureHelp" request.
	//
	// A client that opts into contextSupport will also support the "retriggerCharacters" on "SignatureHelpOptions".
	//
	// @since 3.15.0.
	ContextSupport bool `json:"contextSupport,omitempty"`
}

// TextDocumentClientCapabilitiesSignatureInformation is the client supports the following "SignatureInformation"
// specific properties.
type TextDocumentClientCapabilitiesSignatureInformation struct {
	// DocumentationFormat is the client supports the follow content formats for the documentation
	// property. The order describes the preferred format of the client.
	DocumentationFormat []MarkupKind `json:"documentationFormat,omitempty"`

	// ParameterInformation is the Client capabilities specific to parameter information.
	ParameterInformation *TextDocumentClientCapabilitiesParameterInformation `json:"parameterInformation,omitempty"`

	// ActiveParameterSupport is the client supports the `activeParameter` property on
	// `SignatureInformation` literal.
	//
	// @since 3.16.0.
	ActiveParameterSupport bool `json:"activeParameterSupport,omitempty"`
}

// TextDocumentClientCapabilitiesParameterInformation is the client capabilities specific to parameter information.
type TextDocumentClientCapabilitiesParameterInformation struct {
	// LabelOffsetSupport is the client supports processing label offsets instead of a
	// simple label string.
	//
	// @since 3.14.0.
	LabelOffsetSupport bool `json:"labelOffsetSupport,omitempty"`
}

// DeclarationTextDocumentClientCapabilities capabilities specific to the "textDocument/declaration".
type DeclarationTextDocumentClientCapabilities struct {
	// DynamicRegistration whether declaration supports dynamic registration. If this is set to `true`
	// the client supports the new `(TextDocumentRegistrationOptions & StaticRegistrationOptions)`
	// return value for the corresponding server capability as well.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`

	// LinkSupport is the client supports additional metadata in the form of declaration links.
	//
	// @since 3.14.0.
	LinkSupport bool `json:"linkSupport,omitempty"`
}

// DefinitionTextDocumentClientCapabilities capabilities specific to the "textDocument/definition".
//
// @since 3.14.0.
type DefinitionTextDocumentClientCapabilities struct {
	// DynamicRegistration whether definition supports dynamic registration.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`

	// LinkSupport is the client supports additional metadata in the form of definition links.
	LinkSupport bool `json:"linkSupport,omitempty"`
}

// TypeDefinitionTextDocumentClientCapabilities capabilities specific to the "textDocument/typeDefinition".
//
// @since 3.6.0.
type TypeDefinitionTextDocumentClientCapabilities struct {
	// DynamicRegistration whether typeDefinition supports dynamic registration. If this is set to `true`
	// the client supports the new "(TextDocumentRegistrationOptions & StaticRegistrationOptions)"
	// return value for the corresponding server capability as well.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`

	// LinkSupport is the client supports additional metadata in the form of definition links.
	//
	// @since 3.14.0
	LinkSupport bool `json:"linkSupport,omitempty"`
}

// ImplementationTextDocumentClientCapabilities capabilities specific to the "textDocument/implementation".
//
// @since 3.6.0.
type ImplementationTextDocumentClientCapabilities struct {
	// DynamicRegistration whether implementation supports dynamic registration. If this is set to `true`
	// the client supports the new "(TextDocumentRegistrationOptions & StaticRegistrationOptions)"
	// return value for the corresponding server capability as well.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`

	// LinkSupport is the client supports additional metadata in the form of definition links.
	//
	// @since 3.14.0
	LinkSupport bool `json:"linkSupport,omitempty"`
}

// ReferencesTextDocumentClientCapabilities capabilities specific to the "textDocument/references".
type ReferencesTextDocumentClientCapabilities struct {
	// DynamicRegistration whether references supports dynamic registration.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// DocumentHighlightClientCapabilities capabilities specific to the "textDocument/documentHighlight".
type DocumentHighlightClientCapabilities struct {
	// DynamicRegistration Whether document highlight supports dynamic registration.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// DocumentSymbolClientCapabilities capabilities specific to the "textDocument/documentSymbol".
type DocumentSymbolClientCapabilities struct {
	// DynamicRegistration whether document symbol supports dynamic registration.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`

	// SymbolKind specific capabilities for the "SymbolKindCapabilities".
	SymbolKind *SymbolKindCapabilities `json:"symbolKind,omitempty"`

	// HierarchicalDocumentSymbolSupport is the client support hierarchical document symbols.
	HierarchicalDocumentSymbolSupport bool `json:"hierarchicalDocumentSymbolSupport,omitempty"`

	// TagSupport is the client supports tags on "SymbolInformation". Tags are supported on
	// "DocumentSymbol" if "HierarchicalDocumentSymbolSupport" is set to true.
	// Clients supporting tags have to handle unknown tags gracefully.
	//
	// @since 3.16.0.
	TagSupport *DocumentSymbolClientCapabilitiesTagSupport `json:"tagSupport,omitempty"`

	// LabelSupport is the client supports an additional label presented in the UI when
	// registering a document symbol provider.
	//
	// @since 3.16.0.
	LabelSupport bool `json:"labelSupport,omitempty"`
}

// DocumentSymbolClientCapabilitiesTagSupport TagSupport in the DocumentSymbolClientCapabilities.
//
// @since 3.16.0.
type DocumentSymbolClientCapabilitiesTagSupport struct {
	// ValueSet is the tags supported by the client.
	ValueSet []SymbolTag `json:"valueSet"`
}

// CodeActionClientCapabilities capabilities specific to the "textDocument/codeAction".
type CodeActionClientCapabilities struct {
	// DynamicRegistration whether code action supports dynamic registration.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`

	// CodeActionLiteralSupport is the client support code action literals as a valid
	// response of the "textDocument/codeAction" request.
	//
	// @since 3.8.0
	CodeActionLiteralSupport *CodeActionClientCapabilitiesLiteralSupport `json:"codeActionLiteralSupport,omitempty"`

	// IsPreferredSupport whether code action supports the "isPreferred" property.
	//
	// @since 3.15.0.
	IsPreferredSupport bool `json:"isPreferredSupport,omitempty"`

	// DisabledSupport whether code action supports the `disabled` property.
	//
	// @since 3.16.0.
	DisabledSupport bool `json:"disabledSupport,omitempty"`

	// DataSupport whether code action supports the `data` property which is
	// preserved between a `textDocument/codeAction` and a
	// `codeAction/resolve` request.
	//
	// @since 3.16.0.
	DataSupport bool `json:"dataSupport,omitempty"`

	// ResolveSupport whether the client supports resolving additional code action
	// properties via a separate `codeAction/resolve` request.
	//
	// @since 3.16.0.
	ResolveSupport *CodeActionClientCapabilitiesResolveSupport `json:"resolveSupport,omitempty"`

	// HonorsChangeAnnotations whether the client honors the change annotations in
	// text edits and resource operations returned via the
	// `CodeAction#edit` property by for example presenting
	// the workspace edit in the user interface and asking
	// for confirmation.
	//
	// @since 3.16.0.
	HonorsChangeAnnotations bool `json:"honorsChangeAnnotations,omitempty"`
}

// CodeActionClientCapabilitiesLiteralSupport is the client support code action literals as a valid response of the "textDocument/codeAction" request.
type CodeActionClientCapabilitiesLiteralSupport struct {
	// CodeActionKind is the code action kind is support with the following value
	// set.
	CodeActionKind *CodeActionClientCapabilitiesKind `json:"codeActionKind"`
}

// CodeActionClientCapabilitiesKind is the code action kind is support with the following value set.
type CodeActionClientCapabilitiesKind struct {
	// ValueSet is the code action kind values the client supports. When this
	// property exists the client also guarantees that it will
	// handle values outside its set gracefully and falls back
	// to a default value when unknown.
	ValueSet []CodeActionKind `json:"valueSet"`
}

// CodeActionClientCapabilitiesResolveSupport ResolveSupport in the CodeActionClientCapabilities.
//
// @since 3.16.0.
type CodeActionClientCapabilitiesResolveSupport struct {
	// Properties is the properties that a client can resolve lazily.
	Properties []string `json:"properties"`
}

// CodeLensClientCapabilities capabilities specific to the "textDocument/codeLens".
type CodeLensClientCapabilities struct {
	// DynamicRegistration Whether code lens supports dynamic registration.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// DocumentLinkClientCapabilities capabilities specific to the "textDocument/documentLink".
type DocumentLinkClientCapabilities struct {
	// DynamicRegistration whether document link supports dynamic registration.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`

	// TooltipSupport whether the client supports the "tooltip" property on "DocumentLink".
	//
	// @since 3.15.0.
	TooltipSupport bool `json:"tooltipSupport,omitempty"`
}

// DocumentColorClientCapabilities capabilities specific to the "textDocument/documentColor" and the
// "textDocument/colorPresentation" request.
//
// @since 3.6.0.
type DocumentColorClientCapabilities struct {
	// DynamicRegistration whether colorProvider supports dynamic registration. If this is set to `true`
	// the client supports the new "(ColorProviderOptions & TextDocumentRegistrationOptions & StaticRegistrationOptions)"
	// return value for the corresponding server capability as well.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// DocumentFormattingClientCapabilities capabilities specific to the "textDocument/formatting".
type DocumentFormattingClientCapabilities struct {
	// DynamicRegistration whether code lens supports dynamic registration.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// DocumentRangeFormattingClientCapabilities capabilities specific to the "textDocument/rangeFormatting".
type DocumentRangeFormattingClientCapabilities struct {
	// DynamicRegistration whether code lens supports dynamic registration.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// DocumentOnTypeFormattingClientCapabilities capabilities specific to the "textDocument/onTypeFormatting".
type DocumentOnTypeFormattingClientCapabilities struct {
	// DynamicRegistration whether code lens supports dynamic registration.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// PublishDiagnosticsClientCapabilities capabilities specific to "textDocument/publishDiagnostics".
type PublishDiagnosticsClientCapabilities struct {
	// RelatedInformation whether the clients accepts diagnostics with related information.
	RelatedInformation bool `json:"relatedInformation,omitempty"`

	// TagSupport clients supporting tags have to handle unknown tags gracefully.
	//
	// @since 3.15.0.
	TagSupport *PublishDiagnosticsClientCapabilitiesTagSupport `json:"tagSupport,omitempty"`

	// VersionSupport whether the client interprets the version property of the
	// "textDocument/publishDiagnostics" notification`s parameter.
	//
	// @since 3.15.0.
	VersionSupport bool `json:"versionSupport,omitempty"`

	// CodeDescriptionSupport client supports a codeDescription property
	//
	// @since 3.16.0.
	CodeDescriptionSupport bool `json:"codeDescriptionSupport,omitempty"`

	// DataSupport whether code action supports the `data` property which is
	// preserved between a `textDocument/publishDiagnostics` and
	// `textDocument/codeAction` request.
	//
	// @since 3.16.0.
	DataSupport bool `json:"dataSupport,omitempty"`
}

// PublishDiagnosticsClientCapabilitiesTagSupport is the client capacity of TagSupport.
//
// @since 3.15.0.
type PublishDiagnosticsClientCapabilitiesTagSupport struct {
	// ValueSet is the tags supported by the client.
	ValueSet []DiagnosticTag `json:"valueSet"`
}

// RenameClientCapabilities capabilities specific to the "textDocument/rename".
type RenameClientCapabilities struct {
	// DynamicRegistration whether rename supports dynamic registration.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`

	// PrepareSupport is the client supports testing for validity of rename operations
	// before execution.
	PrepareSupport bool `json:"prepareSupport,omitempty"`

	// PrepareSupportDefaultBehavior client supports the default behavior result
	// (`{ defaultBehavior: boolean }`).
	//
	// The value indicates the default behavior used by the
	// client.
	//
	// @since 3.16.0.
	PrepareSupportDefaultBehavior PrepareSupportDefaultBehavior `json:"prepareSupportDefaultBehavior,omitempty"`

	// HonorsChangeAnnotations whether th client honors the change annotations in
	// text edits and resource operations returned via the
	// rename request's workspace edit by for example presenting
	// the workspace edit in the user interface and asking
	// for confirmation.
	//
	// @since 3.16.0.
	HonorsChangeAnnotations bool `json:"honorsChangeAnnotations,omitempty"`
}

// PrepareSupportDefaultBehavior default behavior of PrepareSupport.
//
// @since 3.16.0.
type PrepareSupportDefaultBehavior float64

// list of PrepareSupportDefaultBehavior.
const (
	// PrepareSupportDefaultBehaviorIdentifier is the client's default behavior is to select the identifier
	// according the to language's syntax rule.
	PrepareSupportDefaultBehaviorIdentifier PrepareSupportDefaultBehavior = 1
)

// String returns a string representation of the PrepareSupportDefaultBehavior.
func (k PrepareSupportDefaultBehavior) String() string {
	switch k {
	case PrepareSupportDefaultBehaviorIdentifier:
		return "Identifier"
	default:
		return strconv.FormatFloat(float64(k), 'f', -10, 64)
	}
}

// FoldingRangeClientCapabilities capabilities specific to "textDocument/foldingRange" requests.
//
// @since 3.10.0.
type FoldingRangeClientCapabilities struct {
	// DynamicRegistration whether implementation supports dynamic registration for folding range providers. If this is set to `true`
	// the client supports the new "(FoldingRangeProviderOptions & TextDocumentRegistrationOptions & StaticRegistrationOptions)"
	// return value for the corresponding server capability as well.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`

	// RangeLimit is the maximum number of folding ranges that the client prefers to receive per document. The value serves as a
	// hint, servers are free to follow the limit.
	RangeLimit uint32 `json:"rangeLimit,omitempty"`

	// LineFoldingOnly if set, the client signals that it only supports folding complete lines. If set, client will
	// ignore specified "startCharacter" and "endCharacter" properties in a FoldingRange.
	LineFoldingOnly bool `json:"lineFoldingOnly,omitempty"`
}

// SelectionRangeClientCapabilities capabilities specific to "textDocument/selectionRange" requests.
//
// @since 3.16.0.
type SelectionRangeClientCapabilities struct {
	// DynamicRegistration whether implementation supports dynamic registration for selection range providers. If this is set to `true`
	// the client supports the new "(SelectionRangeProviderOptions & TextDocumentRegistrationOptions & StaticRegistrationOptions)"
	// return value for the corresponding server capability as well.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// CallHierarchyClientCapabilities capabilities specific to "textDocument/callHierarchy" requests.
//
// @since 3.16.0.
type CallHierarchyClientCapabilities struct {
	// DynamicRegistration whether implementation supports dynamic registration. If this is set to
	// `true` the client supports the new `(TextDocumentRegistrationOptions &
	// StaticRegistrationOptions)` return value for the corresponding server
	// capability as well.}
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// SemanticTokensClientCapabilities capabilities specific to the "textDocument.semanticTokens" request.
//
// @since 3.16.0.
type SemanticTokensClientCapabilities struct {
	// DynamicRegistration whether implementation supports dynamic registration. If this is set to
	// `true` the client supports the new `(TextDocumentRegistrationOptions &
	// StaticRegistrationOptions)` return value for the corresponding server
	// capability as well.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`

	// Requests which requests the client supports and might send to the server
	// depending on the server's capability. Please note that clients might not
	// show semantic tokens or degrade some of the user experience if a range
	// or full request is advertised by the client but not provided by the
	// server. If for example the client capability `requests.full` and
	// `request.range` are both set to true but the server only provides a
	// range provider the client might not render a minimap correctly or might
	// even decide to not show any semantic tokens at all.
	Requests SemanticTokensWorkspaceClientCapabilitiesRequests `json:"requests"`

	// TokenTypes is the token types that the client supports.
	TokenTypes []string `json:"tokenTypes"`

	// TokenModifiers is the token modifiers that the client supports.
	TokenModifiers []string `json:"tokenModifiers"`

	// Formats is the formats the clients supports.
	Formats []TokenFormat `json:"formats"`

	// OverlappingTokenSupport whether the client supports tokens that can overlap each other.
	OverlappingTokenSupport bool `json:"overlappingTokenSupport,omitempty"`

	// MultilineTokenSupport whether the client supports tokens that can span multiple lines.
	MultilineTokenSupport bool `json:"multilineTokenSupport,omitempty"`
}

// SemanticTokensWorkspaceClientCapabilitiesRequests capabilities specific to the "textDocument/semanticTokens/xxx" request.
//
// @since 3.16.0.
type SemanticTokensWorkspaceClientCapabilitiesRequests struct {
	// Range is the client will send the "textDocument/semanticTokens/range" request
	// if the server provides a corresponding handler.
	Range bool `json:"range,omitempty"`

	// Full is the client will send the "textDocument/semanticTokens/full" request
	// if the server provides a corresponding handler. The client will send the
	// `textDocument/semanticTokens/full/delta` request if the server provides a
	// corresponding handler.
	Full interface{} `json:"full,omitempty"`
}

// LinkedEditingRangeClientCapabilities capabilities specific to "textDocument/linkedEditingRange" requests.
//
// @since 3.16.0.
type LinkedEditingRangeClientCapabilities struct {
	// DynamicRegistration whether implementation supports dynamic registration.
	// If this is set to `true` the client supports the new
	// `(TextDocumentRegistrationOptions & StaticRegistrationOptions)`
	// return value for the corresponding server capability as well.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// MonikerClientCapabilities capabilities specific to the "textDocument/moniker" request.
//
// @since 3.16.0.
type MonikerClientCapabilities struct {
	// DynamicRegistration whether implementation supports dynamic registration. If this is set to
	// `true` the client supports the new `(TextDocumentRegistrationOptions &
	// StaticRegistrationOptions)` return value for the corresponding server
	// capability as well.// DynamicRegistration whether implementation supports dynamic registration. If this is set to
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// WindowClientCapabilities represents a WindowClientCapabilities specific client capabilities.
//
// @since 3.15.0.
type WindowClientCapabilities struct {
	// WorkDoneProgress whether client supports handling progress notifications. If set servers are allowed to
	// report in "workDoneProgress" property in the request specific server capabilities.
	//
	// @since 3.15.0.
	WorkDoneProgress bool `json:"workDoneProgress,omitempty"`

	// ShowMessage capabilities specific to the showMessage request.
	//
	// @since 3.16.0.
	ShowMessage *ShowMessageRequestClientCapabilities `json:"showMessage,omitempty"`

	// ShowDocument client capabilities for the show document request.
	//
	// @since 3.16.0.
	ShowDocument *ShowDocumentClientCapabilities `json:"showDocument,omitempty"`
}

// ShowMessageRequestClientCapabilities show message request client capabilities.
//
// @since 3.16.0.
type ShowMessageRequestClientCapabilities struct {
	// MessageActionItem capabilities specific to the "MessageActionItem" type.
	MessageActionItem *ShowMessageRequestClientCapabilitiesMessageActionItem `json:"messageActionItem,omitempty"`
}

// ShowMessageRequestClientCapabilitiesMessageActionItem capabilities specific to the "MessageActionItem" type.
//
// @since 3.16.0.
type ShowMessageRequestClientCapabilitiesMessageActionItem struct {
	// AdditionalPropertiesSupport whether the client supports additional attributes which
	// are preserved and sent back to the server in the
	// request's response.
	AdditionalPropertiesSupport bool `json:"additionalPropertiesSupport,omitempty"`
}

// ShowDocumentClientCapabilities client capabilities for the show document request.
//
// @since 3.16.0.
type ShowDocumentClientCapabilities struct {
	// Support is the client has support for the show document
	// request.
	Support bool `json:"support"`
}

// GeneralClientCapabilities represents a General specific client capabilities.
//
// @since 3.16.0.
type GeneralClientCapabilities struct {
	// RegularExpressions is the client capabilities specific to regular expressions.
	//
	// @since 3.16.0.
	RegularExpressions *RegularExpressionsClientCapabilities `json:"regularExpressions,omitempty"`

	// Markdown client capabilities specific to the client's markdown parser.
	//
	// @since 3.16.0.
	Markdown *MarkdownClientCapabilities `json:"markdown,omitempty"`
}

// RegularExpressionsClientCapabilities represents a client capabilities specific to regular expressions.
//
// The following features from the ECMAScript 2020 regular expression specification are NOT mandatory for a client:
//
//  Assertions
// Lookahead assertion, Negative lookahead assertion, lookbehind assertion, negative lookbehind assertion.
//  Character classes
// Matching control characters using caret notation (e.g. "\cX") and matching UTF-16 code units (e.g. "\uhhhh").
//  Group and ranges
// Named capturing groups.
//  Unicode property escapes
// None of the features needs to be supported.
//
// The only regular expression flag that a client needs to support is "i" to specify a case insensitive search.
//
// @since 3.16.0.
type RegularExpressionsClientCapabilities struct {
	// Engine is the engine's name.
	//
	// Well known engine name is "ECMAScript".
	//  https://tc39.es/ecma262/#sec-regexp-regular-expression-objects
	//  https://developer.mozilla.org/en-US/docs/Web/JavaScript/Guide/Regular_Expressions
	Engine string `json:"engine"`

	// Version is the engine's version.
	//
	// Well known engine version is "ES2020".
	//  https://tc39.es/ecma262/#sec-regexp-regular-expression-objects
	//  https://developer.mozilla.org/en-US/docs/Web/JavaScript/Guide/Regular_Expressions
	Version string `json:"version,omitempty"`
}

// MarkdownClientCapabilities represents a client capabilities specific to the used markdown parser.
//
// @since 3.16.0.
type MarkdownClientCapabilities struct {
	// Parser is the name of the parser.
	Parser string `json:"parser"`

	// version is the version of the parser.
	Version string `json:"version,omitempty"`
}
