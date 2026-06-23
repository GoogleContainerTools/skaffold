// Copyright 2026 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

import (
	"context"
	"strings"

	"go.lsp.dev/jsonrpc2"
)

// Server is the LSP server interface: the set of requests and notifications a
// language server handles. Its method set is the authoritative shape implemented
// by [UnimplementedServer].
type Server interface {
	Initialize(ctx context.Context, params *InitializeParams) (*InitializeResult, error)
	Initialized(ctx context.Context, params *InitializedParams) error
	Shutdown(ctx context.Context) error
	Exit(ctx context.Context) error
	SetTrace(ctx context.Context, params *SetTraceParams) error
	Progress(ctx context.Context, params *ProgressParams) error
	WorkDoneProgressCancel(ctx context.Context, params *WorkDoneProgressCancelParams) error
	DidOpen(ctx context.Context, params *DidOpenTextDocumentParams) error
	DidChange(ctx context.Context, params *DidChangeTextDocumentParams) error
	WillSave(ctx context.Context, params *WillSaveTextDocumentParams) error
	WillSaveWaitUntil(ctx context.Context, params *WillSaveTextDocumentParams) ([]TextEdit, error)
	DidSave(ctx context.Context, params *DidSaveTextDocumentParams) error
	DidClose(ctx context.Context, params *DidCloseTextDocumentParams) error
	DidOpenNotebookDocument(ctx context.Context, params *DidOpenNotebookDocumentParams) error
	DidChangeNotebookDocument(ctx context.Context, params *DidChangeNotebookDocumentParams) error
	DidSaveNotebookDocument(ctx context.Context, params *DidSaveNotebookDocumentParams) error
	DidCloseNotebookDocument(ctx context.Context, params *DidCloseNotebookDocumentParams) error
	Declaration(ctx context.Context, params *DeclarationParams) (DeclarationResult, error)
	Definition(ctx context.Context, params *DefinitionParams) (DefinitionResult, error)
	TypeDefinition(ctx context.Context, params *TypeDefinitionParams) (DefinitionResult, error)
	Implementation(ctx context.Context, params *ImplementationParams) (DefinitionResult, error)
	References(ctx context.Context, params *ReferenceParams) ([]Location, error)
	PrepareCallHierarchy(ctx context.Context, params *CallHierarchyPrepareParams) ([]CallHierarchyItem, error)
	IncomingCalls(ctx context.Context, params *CallHierarchyIncomingCallsParams) ([]CallHierarchyIncomingCall, error)
	OutgoingCalls(ctx context.Context, params *CallHierarchyOutgoingCallsParams) ([]CallHierarchyOutgoingCall, error)
	PrepareTypeHierarchy(ctx context.Context, params *TypeHierarchyPrepareParams) ([]TypeHierarchyItem, error)
	Supertypes(ctx context.Context, params *TypeHierarchySupertypesParams) ([]TypeHierarchyItem, error)
	Subtypes(ctx context.Context, params *TypeHierarchySubtypesParams) ([]TypeHierarchyItem, error)
	DocumentHighlight(ctx context.Context, params *DocumentHighlightParams) ([]DocumentHighlight, error)
	DocumentLink(ctx context.Context, params *DocumentLinkParams) ([]DocumentLink, error)
	DocumentLinkResolve(ctx context.Context, params *DocumentLink) (*DocumentLink, error)
	Hover(ctx context.Context, params *HoverParams) (*Hover, error)
	CodeLens(ctx context.Context, params *CodeLensParams) ([]CodeLens, error)
	CodeLensResolve(ctx context.Context, params *CodeLens) (*CodeLens, error)
	FoldingRanges(ctx context.Context, params *FoldingRangeParams) ([]FoldingRange, error)
	SelectionRange(ctx context.Context, params *SelectionRangeParams) ([]SelectionRange, error)
	DocumentSymbol(ctx context.Context, params *DocumentSymbolParams) (DocumentSymbolResult, error)
	SemanticTokensFull(ctx context.Context, params *SemanticTokensParams) (*SemanticTokens, error)
	SemanticTokensFullDelta(ctx context.Context, params *SemanticTokensDeltaParams) (SemanticTokensDeltaResult, error)
	SemanticTokensRange(ctx context.Context, params *SemanticTokensRangeParams) (*SemanticTokens, error)
	InlineValue(ctx context.Context, params *InlineValueParams) ([]InlineValue, error)
	InlayHint(ctx context.Context, params *InlayHintParams) ([]InlayHint, error)
	InlayHintResolve(ctx context.Context, params *InlayHint) (*InlayHint, error)
	Moniker(ctx context.Context, params *MonikerParams) ([]Moniker, error)
	Completion(ctx context.Context, params *CompletionParams) (CompletionResult, error)
	CompletionResolve(ctx context.Context, params *CompletionItem) (*CompletionItem, error)
	Diagnostic(ctx context.Context, params *DocumentDiagnosticParams) (DocumentDiagnosticReport, error)
	DiagnosticWorkspace(ctx context.Context, params *WorkspaceDiagnosticParams) (*WorkspaceDiagnosticReport, error)
	SignatureHelp(ctx context.Context, params *SignatureHelpParams) (*SignatureHelp, error)
	CodeAction(ctx context.Context, params *CodeActionParams) ([]CommandOrCodeAction, error)
	CodeActionResolve(ctx context.Context, params *CodeAction) (*CodeAction, error)
	DocumentColor(ctx context.Context, params *DocumentColorParams) ([]ColorInformation, error)
	ColorPresentation(ctx context.Context, params *ColorPresentationParams) ([]ColorPresentation, error)
	Formatting(ctx context.Context, params *DocumentFormattingParams) ([]TextEdit, error)
	RangeFormatting(ctx context.Context, params *DocumentRangeFormattingParams) ([]TextEdit, error)
	RangesFormatting(ctx context.Context, params *DocumentRangesFormattingParams) ([]TextEdit, error)
	OnTypeFormatting(ctx context.Context, params *DocumentOnTypeFormattingParams) ([]TextEdit, error)
	Rename(ctx context.Context, params *RenameParams) (*WorkspaceEdit, error)
	PrepareRename(ctx context.Context, params *PrepareRenameParams) (PrepareRenameResult, error)
	LinkedEditingRange(ctx context.Context, params *LinkedEditingRangeParams) (*LinkedEditingRanges, error)
	InlineCompletion(ctx context.Context, params *InlineCompletionParams) (InlineCompletionResult, error)
	Symbols(ctx context.Context, params *WorkspaceSymbolParams) (WorkspaceSymbolResult, error)
	WorkspaceSymbolResolve(ctx context.Context, params *WorkspaceSymbol) (*WorkspaceSymbol, error)
	DidChangeConfiguration(ctx context.Context, params *DidChangeConfigurationParams) error
	DidChangeWorkspaceFolders(ctx context.Context, params *DidChangeWorkspaceFoldersParams) error
	WillCreateFiles(ctx context.Context, params *CreateFilesParams) (*WorkspaceEdit, error)
	WillRenameFiles(ctx context.Context, params *RenameFilesParams) (*WorkspaceEdit, error)
	WillDeleteFiles(ctx context.Context, params *DeleteFilesParams) (*WorkspaceEdit, error)
	DidCreateFiles(ctx context.Context, params *CreateFilesParams) error
	DidRenameFiles(ctx context.Context, params *RenameFilesParams) error
	DidDeleteFiles(ctx context.Context, params *DeleteFilesParams) error
	DidChangeWatchedFiles(ctx context.Context, params *DidChangeWatchedFilesParams) error
	ExecuteCommand(ctx context.Context, params *ExecuteCommandParams) (LSPAny, error)
	TextDocumentContent(ctx context.Context, params *TextDocumentContentParams) (*TextDocumentContentResult, error)
	Request(ctx context.Context, method string, params any) (any, error)
}

// ServerDispatcher returns a [Server] that dispatches LSP requests across conn.
func ServerDispatcher(conn jsonrpc2.Conn) Server {
	return &server{Conn: conn}
}

// ServerHandler returns a [jsonrpc2.Handler] that routes incoming requests to
// server, falling back to [Server.Request] for non-standard methods.
//
//nolint:unparam,revive // handler is part of the stable signature; intentionally unused
func ServerHandler(server Server, handler jsonrpc2.Handler) jsonrpc2.Handler {
	return func(ctx context.Context, req *jsonrpc2.Request) (any, error) {
		if ctx.Err() != nil {
			return nil, ErrRequestCancelled
		}

		result, handled, err := serverDispatch(ctx, server, req)
		if handled || err != nil {
			return result, err
		}

		// Non-standard requests route to Server.Request; params are passed through
		// as an opaque LSPAny so custom methods stay fully general. Copy the
		// borrowed method before invoking user code.
		method := strings.Clone(req.Method())
		var params LSPAny
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, replyParseError(err)
		}

		return server.Request(ctx, method, params)
	}
}

// serverDispatch decodes req and invokes the matching [Server] method, reporting
// handled=true when req named a standard server method.
//
//nolint:gocognit,funlen,gocyclo,cyclop,maintidx
func serverDispatch(ctx context.Context, server Server, req *jsonrpc2.Request) (result any, handled bool, err error) {
	if ctx.Err() != nil {
		return nil, true, ErrRequestCancelled
	}

	switch req.Method() {
	case MethodInitialize: // request
		var params InitializeParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.Initialize(ctx, &params)

		return resp, true, err

	case MethodInitialized: // notification
		var params InitializedParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}

		return nil, true, server.Initialized(ctx, &params)

	case MethodShutdown: // request
		return nil, true, server.Shutdown(ctx)

	case MethodExit: // notification
		return nil, true, server.Exit(ctx)

	case MethodSetTrace: // notification
		var params SetTraceParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}

		return nil, true, server.SetTrace(ctx, &params)

	case MethodProgress: // notification
		var params ProgressParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}

		return nil, true, server.Progress(ctx, &params)

	case MethodWindowWorkDoneProgressCancel: // notification
		var params WorkDoneProgressCancelParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}

		return nil, true, server.WorkDoneProgressCancel(ctx, &params)

	case MethodTextDocumentDidOpen: // notification
		var params DidOpenTextDocumentParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}

		return nil, true, server.DidOpen(ctx, &params)

	case MethodTextDocumentDidChange: // notification
		var params DidChangeTextDocumentParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}

		return nil, true, server.DidChange(ctx, &params)

	case MethodTextDocumentWillSave: // notification
		var params WillSaveTextDocumentParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}

		return nil, true, server.WillSave(ctx, &params)

	case MethodTextDocumentWillSaveWaitUntil: // request
		var params WillSaveTextDocumentParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.WillSaveWaitUntil(ctx, &params)

		return resp, true, err

	case MethodTextDocumentDidSave: // notification
		var params DidSaveTextDocumentParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}

		return nil, true, server.DidSave(ctx, &params)

	case MethodTextDocumentDidClose: // notification
		var params DidCloseTextDocumentParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}

		return nil, true, server.DidClose(ctx, &params)

	case MethodNotebookDocumentDidOpen: // notification
		var params DidOpenNotebookDocumentParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}

		return nil, true, server.DidOpenNotebookDocument(ctx, &params)

	case MethodNotebookDocumentDidChange: // notification
		var params DidChangeNotebookDocumentParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}

		return nil, true, server.DidChangeNotebookDocument(ctx, &params)

	case MethodNotebookDocumentDidSave: // notification
		var params DidSaveNotebookDocumentParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}

		return nil, true, server.DidSaveNotebookDocument(ctx, &params)

	case MethodNotebookDocumentDidClose: // notification
		var params DidCloseNotebookDocumentParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}

		return nil, true, server.DidCloseNotebookDocument(ctx, &params)

	case MethodTextDocumentDeclaration: // request
		var params DeclarationParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.Declaration(ctx, &params)

		return resp, true, err

	case MethodTextDocumentDefinition: // request
		var params DefinitionParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.Definition(ctx, &params)

		return resp, true, err

	case MethodTextDocumentTypeDefinition: // request
		var params TypeDefinitionParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.TypeDefinition(ctx, &params)

		return resp, true, err

	case MethodTextDocumentImplementation: // request
		var params ImplementationParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.Implementation(ctx, &params)

		return resp, true, err

	case MethodTextDocumentReferences: // request
		var params ReferenceParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.References(ctx, &params)

		return resp, true, err

	case MethodTextDocumentPrepareCallHierarchy: // request
		var params CallHierarchyPrepareParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.PrepareCallHierarchy(ctx, &params)

		return resp, true, err

	case MethodCallHierarchyIncomingCalls: // request
		var params CallHierarchyIncomingCallsParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.IncomingCalls(ctx, &params)

		return resp, true, err

	case MethodCallHierarchyOutgoingCalls: // request
		var params CallHierarchyOutgoingCallsParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.OutgoingCalls(ctx, &params)

		return resp, true, err

	case MethodTextDocumentPrepareTypeHierarchy: // request
		var params TypeHierarchyPrepareParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.PrepareTypeHierarchy(ctx, &params)

		return resp, true, err

	case MethodTypeHierarchySupertypes: // request
		var params TypeHierarchySupertypesParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.Supertypes(ctx, &params)

		return resp, true, err

	case MethodTypeHierarchySubtypes: // request
		var params TypeHierarchySubtypesParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.Subtypes(ctx, &params)

		return resp, true, err

	case MethodTextDocumentDocumentHighlight: // request
		var params DocumentHighlightParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.DocumentHighlight(ctx, &params)

		return resp, true, err

	case MethodTextDocumentDocumentLink: // request
		var params DocumentLinkParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.DocumentLink(ctx, &params)

		return resp, true, err

	case MethodDocumentLinkResolve: // request
		var params DocumentLink
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.DocumentLinkResolve(ctx, &params)

		return resp, true, err

	case MethodTextDocumentHover: // request
		var params HoverParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.Hover(ctx, &params)

		return resp, true, err

	case MethodTextDocumentCodeLens: // request
		var params CodeLensParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.CodeLens(ctx, &params)

		return resp, true, err

	case MethodCodeLensResolve: // request
		var params CodeLens
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.CodeLensResolve(ctx, &params)

		return resp, true, err

	case MethodTextDocumentFoldingRange: // request
		var params FoldingRangeParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.FoldingRanges(ctx, &params)

		return resp, true, err

	case MethodTextDocumentSelectionRange: // request
		var params SelectionRangeParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.SelectionRange(ctx, &params)

		return resp, true, err

	case MethodTextDocumentDocumentSymbol: // request
		var params DocumentSymbolParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.DocumentSymbol(ctx, &params)

		return resp, true, err

	case MethodTextDocumentSemanticTokensFull: // request
		var params SemanticTokensParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.SemanticTokensFull(ctx, &params)

		return resp, true, err

	case MethodTextDocumentSemanticTokensFullDelta: // request
		var params SemanticTokensDeltaParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.SemanticTokensFullDelta(ctx, &params)

		return resp, true, err

	case MethodTextDocumentSemanticTokensRange: // request
		var params SemanticTokensRangeParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.SemanticTokensRange(ctx, &params)

		return resp, true, err

	case MethodTextDocumentInlineValue: // request
		var params InlineValueParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.InlineValue(ctx, &params)

		return resp, true, err

	case MethodTextDocumentInlayHint: // request
		var params InlayHintParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.InlayHint(ctx, &params)

		return resp, true, err

	case MethodInlayHintResolve: // request
		var params InlayHint
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.InlayHintResolve(ctx, &params)

		return resp, true, err

	case MethodTextDocumentMoniker: // request
		var params MonikerParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.Moniker(ctx, &params)

		return resp, true, err

	case MethodTextDocumentCompletion: // request
		var params CompletionParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.Completion(ctx, &params)

		return resp, true, err

	case MethodCompletionItemResolve: // request
		var params CompletionItem
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.CompletionResolve(ctx, &params)

		return resp, true, err

	case MethodTextDocumentDiagnostic: // request
		var params DocumentDiagnosticParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.Diagnostic(ctx, &params)

		return resp, true, err

	case MethodWorkspaceDiagnostic: // request
		var params WorkspaceDiagnosticParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.DiagnosticWorkspace(ctx, &params)

		return resp, true, err

	case MethodTextDocumentSignatureHelp: // request
		var params SignatureHelpParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.SignatureHelp(ctx, &params)

		return resp, true, err

	case MethodTextDocumentCodeAction: // request
		var params CodeActionParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.CodeAction(ctx, &params)

		return resp, true, err

	case MethodCodeActionResolve: // request
		var params CodeAction
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.CodeActionResolve(ctx, &params)

		return resp, true, err

	case MethodTextDocumentDocumentColor: // request
		var params DocumentColorParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.DocumentColor(ctx, &params)

		return resp, true, err

	case MethodTextDocumentColorPresentation: // request
		var params ColorPresentationParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.ColorPresentation(ctx, &params)

		return resp, true, err

	case MethodTextDocumentFormatting: // request
		var params DocumentFormattingParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.Formatting(ctx, &params)

		return resp, true, err

	case MethodTextDocumentRangeFormatting: // request
		var params DocumentRangeFormattingParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.RangeFormatting(ctx, &params)

		return resp, true, err

	case MethodTextDocumentRangesFormatting: // request
		var params DocumentRangesFormattingParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.RangesFormatting(ctx, &params)

		return resp, true, err

	case MethodTextDocumentOnTypeFormatting: // request
		var params DocumentOnTypeFormattingParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.OnTypeFormatting(ctx, &params)

		return resp, true, err

	case MethodTextDocumentRename: // request
		var params RenameParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.Rename(ctx, &params)

		return resp, true, err

	case MethodTextDocumentPrepareRename: // request
		var params PrepareRenameParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.PrepareRename(ctx, &params)

		return resp, true, err

	case MethodTextDocumentLinkedEditingRange: // request
		var params LinkedEditingRangeParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.LinkedEditingRange(ctx, &params)

		return resp, true, err

	case MethodTextDocumentInlineCompletion: // request
		var params InlineCompletionParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.InlineCompletion(ctx, &params)

		return resp, true, err

	case MethodWorkspaceSymbol: // request
		var params WorkspaceSymbolParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.Symbols(ctx, &params)

		return resp, true, err

	case MethodWorkspaceSymbolResolve: // request
		var params WorkspaceSymbol
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.WorkspaceSymbolResolve(ctx, &params)

		return resp, true, err

	case MethodWorkspaceDidChangeConfiguration: // notification
		var params DidChangeConfigurationParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}

		return nil, true, server.DidChangeConfiguration(ctx, &params)

	case MethodWorkspaceDidChangeWorkspaceFolders: // notification
		var params DidChangeWorkspaceFoldersParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}

		return nil, true, server.DidChangeWorkspaceFolders(ctx, &params)

	case MethodWorkspaceWillCreateFiles: // request
		var params CreateFilesParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.WillCreateFiles(ctx, &params)

		return resp, true, err

	case MethodWorkspaceWillRenameFiles: // request
		var params RenameFilesParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.WillRenameFiles(ctx, &params)

		return resp, true, err

	case MethodWorkspaceWillDeleteFiles: // request
		var params DeleteFilesParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.WillDeleteFiles(ctx, &params)

		return resp, true, err

	case MethodWorkspaceDidCreateFiles: // notification
		var params CreateFilesParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}

		return nil, true, server.DidCreateFiles(ctx, &params)

	case MethodWorkspaceDidRenameFiles: // notification
		var params RenameFilesParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}

		return nil, true, server.DidRenameFiles(ctx, &params)

	case MethodWorkspaceDidDeleteFiles: // notification
		var params DeleteFilesParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}

		return nil, true, server.DidDeleteFiles(ctx, &params)

	case MethodWorkspaceDidChangeWatchedFiles: // notification
		var params DidChangeWatchedFilesParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}

		return nil, true, server.DidChangeWatchedFiles(ctx, &params)

	case MethodWorkspaceExecuteCommand: // request
		var params ExecuteCommandParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.ExecuteCommand(ctx, &params)

		return resp, true, err

	case MethodWorkspaceTextDocumentContent: // request
		var params TextDocumentContentParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := server.TextDocumentContent(ctx, &params)

		return resp, true, err

	default:
		return nil, false, nil
	}
}

// server is the [Server] dispatcher: it issues client->server requests and
// notifications over a jsonrpc2 connection.
type server struct {
	jsonrpc2.Conn
}

// compile-time assertion that *server satisfies Server.
var _ Server = (*server)(nil)

func (s *server) Initialize(ctx context.Context, params *InitializeParams) (*InitializeResult, error) {
	var result *InitializeResult
	if err := Call(ctx, s.Conn, MethodInitialize, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) Initialized(ctx context.Context, params *InitializedParams) error {
	return s.Conn.Notify(ctx, MethodInitialized, params)
}

func (s *server) Shutdown(ctx context.Context) error {
	return Call(ctx, s.Conn, MethodShutdown, nil, nil)
}

func (s *server) Exit(ctx context.Context) error {
	return s.Conn.Notify(ctx, MethodExit, nil)
}

func (s *server) SetTrace(ctx context.Context, params *SetTraceParams) error {
	return s.Conn.Notify(ctx, MethodSetTrace, params)
}

func (s *server) Progress(ctx context.Context, params *ProgressParams) error {
	return s.Conn.Notify(ctx, MethodProgress, params)
}

func (s *server) WorkDoneProgressCancel(ctx context.Context, params *WorkDoneProgressCancelParams) error {
	return s.Conn.Notify(ctx, MethodWindowWorkDoneProgressCancel, params)
}

func (s *server) DidOpen(ctx context.Context, params *DidOpenTextDocumentParams) error {
	return s.Conn.Notify(ctx, MethodTextDocumentDidOpen, params)
}

func (s *server) DidChange(ctx context.Context, params *DidChangeTextDocumentParams) error {
	return s.Conn.Notify(ctx, MethodTextDocumentDidChange, params)
}

func (s *server) WillSave(ctx context.Context, params *WillSaveTextDocumentParams) error {
	return s.Conn.Notify(ctx, MethodTextDocumentWillSave, params)
}

func (s *server) WillSaveWaitUntil(ctx context.Context, params *WillSaveTextDocumentParams) ([]TextEdit, error) {
	var result []TextEdit
	if err := Call(ctx, s.Conn, MethodTextDocumentWillSaveWaitUntil, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) DidSave(ctx context.Context, params *DidSaveTextDocumentParams) error {
	return s.Conn.Notify(ctx, MethodTextDocumentDidSave, params)
}

func (s *server) DidClose(ctx context.Context, params *DidCloseTextDocumentParams) error {
	return s.Conn.Notify(ctx, MethodTextDocumentDidClose, params)
}

func (s *server) DidOpenNotebookDocument(ctx context.Context, params *DidOpenNotebookDocumentParams) error {
	return s.Conn.Notify(ctx, MethodNotebookDocumentDidOpen, params)
}

func (s *server) DidChangeNotebookDocument(ctx context.Context, params *DidChangeNotebookDocumentParams) error {
	return s.Conn.Notify(ctx, MethodNotebookDocumentDidChange, params)
}

func (s *server) DidSaveNotebookDocument(ctx context.Context, params *DidSaveNotebookDocumentParams) error {
	return s.Conn.Notify(ctx, MethodNotebookDocumentDidSave, params)
}

func (s *server) DidCloseNotebookDocument(ctx context.Context, params *DidCloseNotebookDocumentParams) error {
	return s.Conn.Notify(ctx, MethodNotebookDocumentDidClose, params)
}

func (s *server) Declaration(ctx context.Context, params *DeclarationParams) (DeclarationResult, error) {
	var result DeclarationResult
	if err := Call(ctx, s.Conn, MethodTextDocumentDeclaration, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) Definition(ctx context.Context, params *DefinitionParams) (DefinitionResult, error) {
	var result DefinitionResult
	if err := Call(ctx, s.Conn, MethodTextDocumentDefinition, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) TypeDefinition(ctx context.Context, params *TypeDefinitionParams) (DefinitionResult, error) {
	var result DefinitionResult
	if err := Call(ctx, s.Conn, MethodTextDocumentTypeDefinition, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) Implementation(ctx context.Context, params *ImplementationParams) (DefinitionResult, error) {
	var result DefinitionResult
	if err := Call(ctx, s.Conn, MethodTextDocumentImplementation, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) References(ctx context.Context, params *ReferenceParams) ([]Location, error) {
	var result []Location
	if err := Call(ctx, s.Conn, MethodTextDocumentReferences, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) PrepareCallHierarchy(ctx context.Context, params *CallHierarchyPrepareParams) ([]CallHierarchyItem, error) {
	var result []CallHierarchyItem
	if err := Call(ctx, s.Conn, MethodTextDocumentPrepareCallHierarchy, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) IncomingCalls(ctx context.Context, params *CallHierarchyIncomingCallsParams) ([]CallHierarchyIncomingCall, error) {
	var result []CallHierarchyIncomingCall
	if err := Call(ctx, s.Conn, MethodCallHierarchyIncomingCalls, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) OutgoingCalls(ctx context.Context, params *CallHierarchyOutgoingCallsParams) ([]CallHierarchyOutgoingCall, error) {
	var result []CallHierarchyOutgoingCall
	if err := Call(ctx, s.Conn, MethodCallHierarchyOutgoingCalls, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) PrepareTypeHierarchy(ctx context.Context, params *TypeHierarchyPrepareParams) ([]TypeHierarchyItem, error) {
	var result []TypeHierarchyItem
	if err := Call(ctx, s.Conn, MethodTextDocumentPrepareTypeHierarchy, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) Supertypes(ctx context.Context, params *TypeHierarchySupertypesParams) ([]TypeHierarchyItem, error) {
	var result []TypeHierarchyItem
	if err := Call(ctx, s.Conn, MethodTypeHierarchySupertypes, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) Subtypes(ctx context.Context, params *TypeHierarchySubtypesParams) ([]TypeHierarchyItem, error) {
	var result []TypeHierarchyItem
	if err := Call(ctx, s.Conn, MethodTypeHierarchySubtypes, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) DocumentHighlight(ctx context.Context, params *DocumentHighlightParams) ([]DocumentHighlight, error) {
	var result []DocumentHighlight
	if err := Call(ctx, s.Conn, MethodTextDocumentDocumentHighlight, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) DocumentLink(ctx context.Context, params *DocumentLinkParams) ([]DocumentLink, error) {
	var result []DocumentLink
	if err := Call(ctx, s.Conn, MethodTextDocumentDocumentLink, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) DocumentLinkResolve(ctx context.Context, params *DocumentLink) (*DocumentLink, error) {
	var result *DocumentLink
	if err := Call(ctx, s.Conn, MethodDocumentLinkResolve, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) Hover(ctx context.Context, params *HoverParams) (*Hover, error) {
	var result *Hover
	if err := Call(ctx, s.Conn, MethodTextDocumentHover, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) CodeLens(ctx context.Context, params *CodeLensParams) ([]CodeLens, error) {
	var result []CodeLens
	if err := Call(ctx, s.Conn, MethodTextDocumentCodeLens, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) CodeLensResolve(ctx context.Context, params *CodeLens) (*CodeLens, error) {
	var result *CodeLens
	if err := Call(ctx, s.Conn, MethodCodeLensResolve, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) FoldingRanges(ctx context.Context, params *FoldingRangeParams) ([]FoldingRange, error) {
	var result []FoldingRange
	if err := Call(ctx, s.Conn, MethodTextDocumentFoldingRange, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) SelectionRange(ctx context.Context, params *SelectionRangeParams) ([]SelectionRange, error) {
	var result []SelectionRange
	if err := Call(ctx, s.Conn, MethodTextDocumentSelectionRange, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) DocumentSymbol(ctx context.Context, params *DocumentSymbolParams) (DocumentSymbolResult, error) {
	var result DocumentSymbolResult
	if err := Call(ctx, s.Conn, MethodTextDocumentDocumentSymbol, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) SemanticTokensFull(ctx context.Context, params *SemanticTokensParams) (*SemanticTokens, error) {
	var result *SemanticTokens
	if err := Call(ctx, s.Conn, MethodTextDocumentSemanticTokensFull, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) SemanticTokensFullDelta(ctx context.Context, params *SemanticTokensDeltaParams) (SemanticTokensDeltaResult, error) {
	var result SemanticTokensDeltaResult
	if err := Call(ctx, s.Conn, MethodTextDocumentSemanticTokensFullDelta, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) SemanticTokensRange(ctx context.Context, params *SemanticTokensRangeParams) (*SemanticTokens, error) {
	var result *SemanticTokens
	if err := Call(ctx, s.Conn, MethodTextDocumentSemanticTokensRange, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) InlineValue(ctx context.Context, params *InlineValueParams) ([]InlineValue, error) {
	var result []InlineValue
	if err := Call(ctx, s.Conn, MethodTextDocumentInlineValue, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) InlayHint(ctx context.Context, params *InlayHintParams) ([]InlayHint, error) {
	var result []InlayHint
	if err := Call(ctx, s.Conn, MethodTextDocumentInlayHint, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) InlayHintResolve(ctx context.Context, params *InlayHint) (*InlayHint, error) {
	var result *InlayHint
	if err := Call(ctx, s.Conn, MethodInlayHintResolve, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) Moniker(ctx context.Context, params *MonikerParams) ([]Moniker, error) {
	var result []Moniker
	if err := Call(ctx, s.Conn, MethodTextDocumentMoniker, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) Completion(ctx context.Context, params *CompletionParams) (CompletionResult, error) {
	var result CompletionResult
	if err := Call(ctx, s.Conn, MethodTextDocumentCompletion, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) CompletionResolve(ctx context.Context, params *CompletionItem) (*CompletionItem, error) {
	var result *CompletionItem
	if err := Call(ctx, s.Conn, MethodCompletionItemResolve, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) Diagnostic(ctx context.Context, params *DocumentDiagnosticParams) (DocumentDiagnosticReport, error) {
	var result DocumentDiagnosticReport
	if err := Call(ctx, s.Conn, MethodTextDocumentDiagnostic, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) DiagnosticWorkspace(ctx context.Context, params *WorkspaceDiagnosticParams) (*WorkspaceDiagnosticReport, error) {
	var result *WorkspaceDiagnosticReport
	if err := Call(ctx, s.Conn, MethodWorkspaceDiagnostic, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) SignatureHelp(ctx context.Context, params *SignatureHelpParams) (*SignatureHelp, error) {
	var result *SignatureHelp
	if err := Call(ctx, s.Conn, MethodTextDocumentSignatureHelp, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) CodeAction(ctx context.Context, params *CodeActionParams) ([]CommandOrCodeAction, error) {
	var result []CommandOrCodeAction
	if err := Call(ctx, s.Conn, MethodTextDocumentCodeAction, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) CodeActionResolve(ctx context.Context, params *CodeAction) (*CodeAction, error) {
	var result *CodeAction
	if err := Call(ctx, s.Conn, MethodCodeActionResolve, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) DocumentColor(ctx context.Context, params *DocumentColorParams) ([]ColorInformation, error) {
	var result []ColorInformation
	if err := Call(ctx, s.Conn, MethodTextDocumentDocumentColor, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) ColorPresentation(ctx context.Context, params *ColorPresentationParams) ([]ColorPresentation, error) {
	var result []ColorPresentation
	if err := Call(ctx, s.Conn, MethodTextDocumentColorPresentation, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) Formatting(ctx context.Context, params *DocumentFormattingParams) ([]TextEdit, error) {
	var result []TextEdit
	if err := Call(ctx, s.Conn, MethodTextDocumentFormatting, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) RangeFormatting(ctx context.Context, params *DocumentRangeFormattingParams) ([]TextEdit, error) {
	var result []TextEdit
	if err := Call(ctx, s.Conn, MethodTextDocumentRangeFormatting, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) RangesFormatting(ctx context.Context, params *DocumentRangesFormattingParams) ([]TextEdit, error) {
	var result []TextEdit
	if err := Call(ctx, s.Conn, MethodTextDocumentRangesFormatting, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) OnTypeFormatting(ctx context.Context, params *DocumentOnTypeFormattingParams) ([]TextEdit, error) {
	var result []TextEdit
	if err := Call(ctx, s.Conn, MethodTextDocumentOnTypeFormatting, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) Rename(ctx context.Context, params *RenameParams) (*WorkspaceEdit, error) {
	var result *WorkspaceEdit
	if err := Call(ctx, s.Conn, MethodTextDocumentRename, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) PrepareRename(ctx context.Context, params *PrepareRenameParams) (PrepareRenameResult, error) {
	var result PrepareRenameResult
	if err := Call(ctx, s.Conn, MethodTextDocumentPrepareRename, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) LinkedEditingRange(ctx context.Context, params *LinkedEditingRangeParams) (*LinkedEditingRanges, error) {
	var result *LinkedEditingRanges
	if err := Call(ctx, s.Conn, MethodTextDocumentLinkedEditingRange, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) InlineCompletion(ctx context.Context, params *InlineCompletionParams) (InlineCompletionResult, error) {
	var result InlineCompletionResult
	if err := Call(ctx, s.Conn, MethodTextDocumentInlineCompletion, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) Symbols(ctx context.Context, params *WorkspaceSymbolParams) (WorkspaceSymbolResult, error) {
	var result WorkspaceSymbolResult
	if err := Call(ctx, s.Conn, MethodWorkspaceSymbol, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) WorkspaceSymbolResolve(ctx context.Context, params *WorkspaceSymbol) (*WorkspaceSymbol, error) {
	var result *WorkspaceSymbol
	if err := Call(ctx, s.Conn, MethodWorkspaceSymbolResolve, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) DidChangeConfiguration(ctx context.Context, params *DidChangeConfigurationParams) error {
	return s.Conn.Notify(ctx, MethodWorkspaceDidChangeConfiguration, params)
}

func (s *server) DidChangeWorkspaceFolders(ctx context.Context, params *DidChangeWorkspaceFoldersParams) error {
	return s.Conn.Notify(ctx, MethodWorkspaceDidChangeWorkspaceFolders, params)
}

func (s *server) WillCreateFiles(ctx context.Context, params *CreateFilesParams) (*WorkspaceEdit, error) {
	var result *WorkspaceEdit
	if err := Call(ctx, s.Conn, MethodWorkspaceWillCreateFiles, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) WillRenameFiles(ctx context.Context, params *RenameFilesParams) (*WorkspaceEdit, error) {
	var result *WorkspaceEdit
	if err := Call(ctx, s.Conn, MethodWorkspaceWillRenameFiles, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) WillDeleteFiles(ctx context.Context, params *DeleteFilesParams) (*WorkspaceEdit, error) {
	var result *WorkspaceEdit
	if err := Call(ctx, s.Conn, MethodWorkspaceWillDeleteFiles, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) DidCreateFiles(ctx context.Context, params *CreateFilesParams) error {
	return s.Conn.Notify(ctx, MethodWorkspaceDidCreateFiles, params)
}

func (s *server) DidRenameFiles(ctx context.Context, params *RenameFilesParams) error {
	return s.Conn.Notify(ctx, MethodWorkspaceDidRenameFiles, params)
}

func (s *server) DidDeleteFiles(ctx context.Context, params *DeleteFilesParams) error {
	return s.Conn.Notify(ctx, MethodWorkspaceDidDeleteFiles, params)
}

func (s *server) DidChangeWatchedFiles(ctx context.Context, params *DidChangeWatchedFilesParams) error {
	return s.Conn.Notify(ctx, MethodWorkspaceDidChangeWatchedFiles, params)
}

func (s *server) ExecuteCommand(ctx context.Context, params *ExecuteCommandParams) (LSPAny, error) {
	var result LSPAny
	if err := Call(ctx, s.Conn, MethodWorkspaceExecuteCommand, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) TextDocumentContent(ctx context.Context, params *TextDocumentContentParams) (*TextDocumentContentResult, error) {
	var result *TextDocumentContentResult
	if err := Call(ctx, s.Conn, MethodWorkspaceTextDocumentContent, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *server) Request(ctx context.Context, method string, params any) (any, error) {
	var result any
	if err := Call(ctx, s.Conn, method, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}
