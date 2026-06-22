// Copyright 2026 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

import (
	"context"

	"go.lsp.dev/jsonrpc2"
)

// errNotImplemented is the sentinel returned by every [UnimplementedServer] and
// [UnimplementedClient] method. It carries the JSON-RPC "method not found" code
// so peers observe a well-formed, classified error for un-overridden methods.
var errNotImplemented = jsonrpc2.NewError(jsonrpc2.ErrMethodNotFound.Code, "not implemented")

// UnimplementedServer is an embeddable default implementation of the [Server]
// interface. Every method returns [errNotImplemented] together with the zero
// value of its result, so consumers can embed it and override only the methods
// they support.
type UnimplementedServer struct{}

// compile-time assertion that UnimplementedServer satisfies Server.
var _ Server = UnimplementedServer{}

func (UnimplementedServer) Initialize(context.Context, *InitializeParams) (*InitializeResult, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) Initialized(context.Context, *InitializedParams) error {
	return errNotImplemented
}

func (UnimplementedServer) Shutdown(context.Context) error { return errNotImplemented }

func (UnimplementedServer) Exit(context.Context) error { return errNotImplemented }

func (UnimplementedServer) SetTrace(context.Context, *SetTraceParams) error {
	return errNotImplemented
}

func (UnimplementedServer) Progress(context.Context, *ProgressParams) error {
	return errNotImplemented
}

func (UnimplementedServer) WorkDoneProgressCancel(context.Context, *WorkDoneProgressCancelParams) error {
	return errNotImplemented
}

func (UnimplementedServer) DidOpen(context.Context, *DidOpenTextDocumentParams) error {
	return errNotImplemented
}

func (UnimplementedServer) DidChange(context.Context, *DidChangeTextDocumentParams) error {
	return errNotImplemented
}

func (UnimplementedServer) WillSave(context.Context, *WillSaveTextDocumentParams) error {
	return errNotImplemented
}

func (UnimplementedServer) WillSaveWaitUntil(context.Context, *WillSaveTextDocumentParams) ([]TextEdit, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) DidSave(context.Context, *DidSaveTextDocumentParams) error {
	return errNotImplemented
}

func (UnimplementedServer) DidClose(context.Context, *DidCloseTextDocumentParams) error {
	return errNotImplemented
}

func (UnimplementedServer) DidOpenNotebookDocument(context.Context, *DidOpenNotebookDocumentParams) error {
	return errNotImplemented
}

func (UnimplementedServer) DidChangeNotebookDocument(context.Context, *DidChangeNotebookDocumentParams) error {
	return errNotImplemented
}

func (UnimplementedServer) DidSaveNotebookDocument(context.Context, *DidSaveNotebookDocumentParams) error {
	return errNotImplemented
}

func (UnimplementedServer) DidCloseNotebookDocument(context.Context, *DidCloseNotebookDocumentParams) error {
	return errNotImplemented
}

func (UnimplementedServer) Declaration(context.Context, *DeclarationParams) (DeclarationResult, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) Definition(context.Context, *DefinitionParams) (DefinitionResult, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) TypeDefinition(context.Context, *TypeDefinitionParams) (DefinitionResult, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) Implementation(context.Context, *ImplementationParams) (DefinitionResult, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) References(context.Context, *ReferenceParams) ([]Location, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) PrepareCallHierarchy(context.Context, *CallHierarchyPrepareParams) ([]CallHierarchyItem, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) IncomingCalls(context.Context, *CallHierarchyIncomingCallsParams) ([]CallHierarchyIncomingCall, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) OutgoingCalls(context.Context, *CallHierarchyOutgoingCallsParams) ([]CallHierarchyOutgoingCall, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) PrepareTypeHierarchy(context.Context, *TypeHierarchyPrepareParams) ([]TypeHierarchyItem, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) Supertypes(context.Context, *TypeHierarchySupertypesParams) ([]TypeHierarchyItem, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) Subtypes(context.Context, *TypeHierarchySubtypesParams) ([]TypeHierarchyItem, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) DocumentHighlight(context.Context, *DocumentHighlightParams) ([]DocumentHighlight, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) DocumentLink(context.Context, *DocumentLinkParams) ([]DocumentLink, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) DocumentLinkResolve(context.Context, *DocumentLink) (*DocumentLink, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) Hover(context.Context, *HoverParams) (*Hover, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) CodeLens(context.Context, *CodeLensParams) ([]CodeLens, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) CodeLensResolve(context.Context, *CodeLens) (*CodeLens, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) FoldingRanges(context.Context, *FoldingRangeParams) ([]FoldingRange, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) SelectionRange(context.Context, *SelectionRangeParams) ([]SelectionRange, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) DocumentSymbol(context.Context, *DocumentSymbolParams) (DocumentSymbolResult, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) SemanticTokensFull(context.Context, *SemanticTokensParams) (*SemanticTokens, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) SemanticTokensFullDelta(context.Context, *SemanticTokensDeltaParams) (SemanticTokensDeltaResult, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) SemanticTokensRange(context.Context, *SemanticTokensRangeParams) (*SemanticTokens, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) InlineValue(context.Context, *InlineValueParams) ([]InlineValue, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) InlayHint(context.Context, *InlayHintParams) ([]InlayHint, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) InlayHintResolve(context.Context, *InlayHint) (*InlayHint, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) Moniker(context.Context, *MonikerParams) ([]Moniker, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) Completion(context.Context, *CompletionParams) (CompletionResult, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) CompletionResolve(context.Context, *CompletionItem) (*CompletionItem, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) Diagnostic(context.Context, *DocumentDiagnosticParams) (DocumentDiagnosticReport, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) DiagnosticWorkspace(context.Context, *WorkspaceDiagnosticParams) (*WorkspaceDiagnosticReport, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) SignatureHelp(context.Context, *SignatureHelpParams) (*SignatureHelp, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) CodeAction(context.Context, *CodeActionParams) ([]CommandOrCodeAction, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) CodeActionResolve(context.Context, *CodeAction) (*CodeAction, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) DocumentColor(context.Context, *DocumentColorParams) ([]ColorInformation, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) ColorPresentation(context.Context, *ColorPresentationParams) ([]ColorPresentation, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) Formatting(context.Context, *DocumentFormattingParams) ([]TextEdit, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) RangeFormatting(context.Context, *DocumentRangeFormattingParams) ([]TextEdit, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) RangesFormatting(context.Context, *DocumentRangesFormattingParams) ([]TextEdit, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) OnTypeFormatting(context.Context, *DocumentOnTypeFormattingParams) ([]TextEdit, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) Rename(context.Context, *RenameParams) (*WorkspaceEdit, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) PrepareRename(context.Context, *PrepareRenameParams) (PrepareRenameResult, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) LinkedEditingRange(context.Context, *LinkedEditingRangeParams) (*LinkedEditingRanges, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) InlineCompletion(context.Context, *InlineCompletionParams) (InlineCompletionResult, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) Symbols(context.Context, *WorkspaceSymbolParams) (WorkspaceSymbolResult, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) WorkspaceSymbolResolve(context.Context, *WorkspaceSymbol) (*WorkspaceSymbol, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) DidChangeConfiguration(context.Context, *DidChangeConfigurationParams) error {
	return errNotImplemented
}

func (UnimplementedServer) DidChangeWorkspaceFolders(context.Context, *DidChangeWorkspaceFoldersParams) error {
	return errNotImplemented
}

func (UnimplementedServer) WillCreateFiles(context.Context, *CreateFilesParams) (*WorkspaceEdit, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) WillRenameFiles(context.Context, *RenameFilesParams) (*WorkspaceEdit, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) WillDeleteFiles(context.Context, *DeleteFilesParams) (*WorkspaceEdit, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) DidCreateFiles(context.Context, *CreateFilesParams) error {
	return errNotImplemented
}

func (UnimplementedServer) DidRenameFiles(context.Context, *RenameFilesParams) error {
	return errNotImplemented
}

func (UnimplementedServer) DidDeleteFiles(context.Context, *DeleteFilesParams) error {
	return errNotImplemented
}

func (UnimplementedServer) DidChangeWatchedFiles(context.Context, *DidChangeWatchedFilesParams) error {
	return errNotImplemented
}

func (UnimplementedServer) ExecuteCommand(context.Context, *ExecuteCommandParams) (LSPAny, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) TextDocumentContent(context.Context, *TextDocumentContentParams) (*TextDocumentContentResult, error) {
	return nil, errNotImplemented
}

func (UnimplementedServer) Request(context.Context, string, any) (any, error) {
	return nil, errNotImplemented
}

// UnimplementedClient is an embeddable default implementation of the [Client]
// interface. Every method returns [errNotImplemented] together with the zero
// value of its result, so consumers can embed it and override only the methods
// they support.
type UnimplementedClient struct{}

// compile-time assertion that UnimplementedClient satisfies Client.
var _ Client = UnimplementedClient{}

func (UnimplementedClient) Progress(context.Context, *ProgressParams) error {
	return errNotImplemented
}

func (UnimplementedClient) LogTrace(context.Context, *LogTraceParams) error {
	return errNotImplemented
}

func (UnimplementedClient) RegisterCapability(context.Context, *RegistrationParams) error {
	return errNotImplemented
}

func (UnimplementedClient) UnregisterCapability(context.Context, *UnregistrationParams) error {
	return errNotImplemented
}

func (UnimplementedClient) ShowMessage(context.Context, *ShowMessageParams) error {
	return errNotImplemented
}

func (UnimplementedClient) ShowMessageRequest(context.Context, *ShowMessageRequestParams) (*MessageActionItem, error) {
	return nil, errNotImplemented
}

func (UnimplementedClient) LogMessage(context.Context, *LogMessageParams) error {
	return errNotImplemented
}

func (UnimplementedClient) ShowDocument(context.Context, *ShowDocumentParams) (*ShowDocumentResult, error) {
	return nil, errNotImplemented
}

func (UnimplementedClient) WorkDoneProgressCreate(context.Context, *WorkDoneProgressCreateParams) error {
	return errNotImplemented
}

func (UnimplementedClient) Telemetry(context.Context, LSPAny) error {
	return errNotImplemented
}

func (UnimplementedClient) PublishDiagnostics(context.Context, *PublishDiagnosticsParams) error {
	return errNotImplemented
}

func (UnimplementedClient) Configuration(context.Context, *ConfigurationParams) ([]LSPAny, error) {
	return nil, errNotImplemented
}

func (UnimplementedClient) WorkspaceFolders(context.Context) ([]WorkspaceFolder, error) {
	return nil, errNotImplemented
}

func (UnimplementedClient) ApplyEdit(context.Context, *ApplyWorkspaceEditParams) (*ApplyWorkspaceEditResult, error) {
	return nil, errNotImplemented
}

func (UnimplementedClient) CodeLensRefresh(context.Context) error { return errNotImplemented }

func (UnimplementedClient) FoldingRangeRefresh(context.Context) error { return errNotImplemented }

func (UnimplementedClient) SemanticTokensRefresh(context.Context) error { return errNotImplemented }

func (UnimplementedClient) InlineValueRefresh(context.Context) error { return errNotImplemented }

func (UnimplementedClient) InlayHintRefresh(context.Context) error { return errNotImplemented }

func (UnimplementedClient) DiagnosticRefresh(context.Context) error { return errNotImplemented }

func (UnimplementedClient) TextDocumentContentRefresh(context.Context, *TextDocumentContentRefreshParams) error {
	return errNotImplemented
}
