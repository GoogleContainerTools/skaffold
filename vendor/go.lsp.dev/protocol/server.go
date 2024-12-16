// SPDX-FileCopyrightText: 2019 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

import (
	"bytes"
	"context"
	"fmt"

	"github.com/segmentio/encoding/json"
	"go.uber.org/zap"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/pkg/xcontext"
)

// ServerDispatcher returns a Server that dispatches LSP requests across the
// given jsonrpc2 connection.
func ServerDispatcher(conn jsonrpc2.Conn, logger *zap.Logger) Server {
	return &server{
		Conn:   conn,
		logger: logger,
	}
}

// ServerHandler jsonrpc2.Handler of Language Server Prococol Server.
//nolint:unparam
func ServerHandler(server Server, handler jsonrpc2.Handler) jsonrpc2.Handler {
	h := func(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
		if ctx.Err() != nil {
			xctx := xcontext.Detach(ctx)

			return reply(xctx, nil, ErrRequestCancelled)
		}
		handled, err := serverDispatch(ctx, server, reply, req)
		if handled || err != nil {
			return err
		}

		// TODO: This code is wrong, it ignores handler and assumes non standard
		// request handles everything
		// non standard request should just be a layered handler.
		var params interface{}
		if err := json.Unmarshal(req.Params(), &params); err != nil {
			return replyParseError(ctx, reply, err)
		}

		resp, err := server.Request(ctx, req.Method(), params)

		return reply(ctx, resp, err)
	}

	return h
}

// serverDispatch implements jsonrpc2.Handler.
//nolint:gocognit,funlen,gocyclo,cyclop
func serverDispatch(ctx context.Context, server Server, reply jsonrpc2.Replier, req jsonrpc2.Request) (handled bool, err error) {
	if ctx.Err() != nil {
		return true, reply(ctx, nil, ErrRequestCancelled)
	}

	dec := json.NewDecoder(bytes.NewReader(req.Params()))
	logger := LoggerFromContext(ctx)

	switch req.Method() {
	case MethodInitialize: // request
		defer logger.Debug(MethodInitialize, zap.Error(err))

		var params InitializeParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.Initialize(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodInitialized: // notification
		defer logger.Debug(MethodInitialized, zap.Error(err))

		var params InitializedParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		err := server.Initialized(ctx, &params)

		return true, reply(ctx, nil, err)

	case MethodShutdown: // request
		defer logger.Debug(MethodShutdown, zap.Error(err))

		if len(req.Params()) > 0 {
			return true, reply(ctx, nil, fmt.Errorf("expected no params: %w", jsonrpc2.ErrInvalidParams))
		}

		err := server.Shutdown(ctx)

		return true, reply(ctx, nil, err)

	case MethodExit: // notification
		defer logger.Debug(MethodExit, zap.Error(err))

		if len(req.Params()) > 0 {
			return true, reply(ctx, nil, fmt.Errorf("expected no params: %w", jsonrpc2.ErrInvalidParams))
		}

		err := server.Exit(ctx)

		return true, reply(ctx, nil, err)

	case MethodWorkDoneProgressCancel: // notification
		defer logger.Debug(MethodWorkDoneProgressCancel, zap.Error(err))

		var params WorkDoneProgressCancelParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		err := server.WorkDoneProgressCancel(ctx, &params)

		return true, reply(ctx, nil, err)

	case MethodLogTrace: // notification
		defer logger.Debug(MethodLogTrace, zap.Error(err))

		var params LogTraceParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		err := server.LogTrace(ctx, &params)

		return true, reply(ctx, nil, err)

	case MethodSetTrace: // notification
		defer logger.Debug(MethodSetTrace, zap.Error(err))

		var params SetTraceParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		err := server.SetTrace(ctx, &params)

		return true, reply(ctx, nil, err)

	case MethodTextDocumentCodeAction: // request
		defer logger.Debug(MethodTextDocumentCodeAction, zap.Error(err))

		var params CodeActionParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.CodeAction(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodTextDocumentCodeLens: // request
		defer logger.Debug(MethodTextDocumentCodeLens, zap.Error(err))

		var params CodeLensParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.CodeLens(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodCodeLensResolve: // request
		defer logger.Debug(MethodCodeLensResolve, zap.Error(err))

		var params CodeLens
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.CodeLensResolve(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodTextDocumentColorPresentation: // request
		defer logger.Debug(MethodTextDocumentColorPresentation, zap.Error(err))

		var params ColorPresentationParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.ColorPresentation(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodTextDocumentCompletion: // request
		defer logger.Debug(MethodTextDocumentCompletion, zap.Error(err))

		var params CompletionParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.Completion(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodCompletionItemResolve: // request
		defer logger.Debug(MethodCompletionItemResolve, zap.Error(err))

		var params CompletionItem
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.CompletionResolve(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodTextDocumentDeclaration: // request
		defer logger.Debug(MethodTextDocumentDeclaration, zap.Error(err))

		var params DeclarationParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.Declaration(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodTextDocumentDefinition: // request
		defer logger.Debug(MethodTextDocumentDefinition, zap.Error(err))

		var params DefinitionParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.Definition(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodTextDocumentDidChange: // notification
		defer logger.Debug(MethodTextDocumentDidChange, zap.Error(err))

		var params DidChangeTextDocumentParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		err := server.DidChange(ctx, &params)

		return true, reply(ctx, nil, err)

	case MethodWorkspaceDidChangeConfiguration: // notification
		defer logger.Debug(MethodWorkspaceDidChangeConfiguration, zap.Error(err))

		var params DidChangeConfigurationParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		err := server.DidChangeConfiguration(ctx, &params)

		return true, reply(ctx, nil, err)

	case MethodWorkspaceDidChangeWatchedFiles: // notification
		defer logger.Debug(MethodWorkspaceDidChangeWatchedFiles, zap.Error(err))

		var params DidChangeWatchedFilesParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		err := server.DidChangeWatchedFiles(ctx, &params)

		return true, reply(ctx, nil, err)

	case MethodWorkspaceDidChangeWorkspaceFolders: // notification
		defer logger.Debug(MethodWorkspaceDidChangeWorkspaceFolders, zap.Error(err))

		var params DidChangeWorkspaceFoldersParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		err := server.DidChangeWorkspaceFolders(ctx, &params)

		return true, reply(ctx, nil, err)

	case MethodTextDocumentDidClose: // notification
		defer logger.Debug(MethodTextDocumentDidClose, zap.Error(err))

		var params DidCloseTextDocumentParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		err := server.DidClose(ctx, &params)

		return true, reply(ctx, nil, err)

	case MethodTextDocumentDidOpen: // notification
		defer logger.Debug(MethodTextDocumentDidOpen, zap.Error(err))

		var params DidOpenTextDocumentParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		err := server.DidOpen(ctx, &params)

		return true, reply(ctx, nil, err)

	case MethodTextDocumentDidSave: // notification
		defer logger.Debug(MethodTextDocumentDidSave, zap.Error(err))

		var params DidSaveTextDocumentParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		err := server.DidSave(ctx, &params)

		return true, reply(ctx, nil, err)

	case MethodTextDocumentDocumentColor: // request
		defer logger.Debug(MethodTextDocumentDocumentColor, zap.Error(err))

		var params DocumentColorParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.DocumentColor(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodTextDocumentDocumentHighlight: // request
		defer logger.Debug(MethodTextDocumentDocumentHighlight, zap.Error(err))

		var params DocumentHighlightParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.DocumentHighlight(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodTextDocumentDocumentLink: // request
		defer logger.Debug(MethodTextDocumentDocumentLink, zap.Error(err))

		var params DocumentLinkParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.DocumentLink(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodDocumentLinkResolve: // request
		defer logger.Debug(MethodDocumentLinkResolve, zap.Error(err))

		var params DocumentLink
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.DocumentLinkResolve(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodTextDocumentDocumentSymbol: // request
		defer logger.Debug(MethodTextDocumentDocumentSymbol, zap.Error(err))

		var params DocumentSymbolParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.DocumentSymbol(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodWorkspaceExecuteCommand: // request
		defer logger.Debug(MethodWorkspaceExecuteCommand, zap.Error(err))

		var params ExecuteCommandParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.ExecuteCommand(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodTextDocumentFoldingRange: // request
		defer logger.Debug(MethodTextDocumentFoldingRange, zap.Error(err))

		var params FoldingRangeParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.FoldingRanges(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodTextDocumentFormatting: // request
		defer logger.Debug(MethodTextDocumentFormatting, zap.Error(err))

		var params DocumentFormattingParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.Formatting(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodTextDocumentHover: // request
		defer logger.Debug(MethodTextDocumentHover, zap.Error(err))

		var params HoverParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.Hover(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodTextDocumentImplementation: // request
		defer logger.Debug(MethodTextDocumentImplementation, zap.Error(err))

		var params ImplementationParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.Implementation(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodTextDocumentOnTypeFormatting: // request
		defer logger.Debug(MethodTextDocumentOnTypeFormatting, zap.Error(err))

		var params DocumentOnTypeFormattingParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.OnTypeFormatting(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodTextDocumentPrepareRename: // request
		defer logger.Debug(MethodTextDocumentPrepareRename, zap.Error(err))

		var params PrepareRenameParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.PrepareRename(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodTextDocumentRangeFormatting: // request
		defer logger.Debug(MethodTextDocumentRangeFormatting, zap.Error(err))

		var params DocumentRangeFormattingParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.RangeFormatting(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodTextDocumentReferences: // request
		defer logger.Debug(MethodTextDocumentReferences, zap.Error(err))

		var params ReferenceParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.References(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodTextDocumentRename: // request
		defer logger.Debug(MethodTextDocumentRename, zap.Error(err))

		var params RenameParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.Rename(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodTextDocumentSignatureHelp: // request
		defer logger.Debug(MethodTextDocumentSignatureHelp, zap.Error(err))

		var params SignatureHelpParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.SignatureHelp(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodWorkspaceSymbol: // request
		defer logger.Debug(MethodWorkspaceSymbol, zap.Error(err))

		var params WorkspaceSymbolParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.Symbols(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodTextDocumentTypeDefinition: // request
		defer logger.Debug(MethodTextDocumentTypeDefinition, zap.Error(err))

		var params TypeDefinitionParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.TypeDefinition(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodTextDocumentWillSave: // notification
		defer logger.Debug(MethodTextDocumentWillSave, zap.Error(err))

		var params WillSaveTextDocumentParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		err := server.WillSave(ctx, &params)

		return true, reply(ctx, nil, err)

	case MethodTextDocumentWillSaveWaitUntil: // request
		defer logger.Debug(MethodTextDocumentWillSaveWaitUntil, zap.Error(err))

		var params WillSaveTextDocumentParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.WillSaveWaitUntil(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodShowDocument: // request
		defer logger.Debug(MethodShowDocument, zap.Error(err))

		var params ShowDocumentParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.ShowDocument(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodWillCreateFiles: // request
		defer logger.Debug(MethodWillCreateFiles, zap.Error(err))

		var params CreateFilesParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.WillCreateFiles(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodDidCreateFiles: // notification
		defer logger.Debug(MethodDidCreateFiles, zap.Error(err))

		var params CreateFilesParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		err := server.DidCreateFiles(ctx, &params)

		return true, reply(ctx, nil, err)

	case MethodWillRenameFiles: // request
		defer logger.Debug(MethodWillRenameFiles, zap.Error(err))

		var params RenameFilesParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.WillRenameFiles(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodDidRenameFiles: // notification
		defer logger.Debug(MethodDidRenameFiles, zap.Error(err))

		var params RenameFilesParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		err := server.DidRenameFiles(ctx, &params)

		return true, reply(ctx, nil, err)

	case MethodWillDeleteFiles: // request
		defer logger.Debug(MethodWillDeleteFiles, zap.Error(err))

		var params DeleteFilesParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.WillDeleteFiles(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodDidDeleteFiles: // notification
		defer logger.Debug(MethodDidDeleteFiles, zap.Error(err))

		var params DeleteFilesParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		err := server.DidDeleteFiles(ctx, &params)

		return true, reply(ctx, nil, err)

	case MethodCodeLensRefresh: // request
		defer logger.Debug(MethodCodeLensRefresh, zap.Error(err))

		if len(req.Params()) > 0 {
			return true, reply(ctx, nil, fmt.Errorf("expected no params: %w", jsonrpc2.ErrInvalidParams))
		}

		err := server.CodeLensRefresh(ctx)

		return true, reply(ctx, nil, err)

	case MethodTextDocumentPrepareCallHierarchy: // request
		defer logger.Debug(MethodTextDocumentPrepareCallHierarchy, zap.Error(err))

		var params CallHierarchyPrepareParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.PrepareCallHierarchy(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodCallHierarchyIncomingCalls: // request
		defer logger.Debug(MethodCallHierarchyIncomingCalls, zap.Error(err))

		var params CallHierarchyIncomingCallsParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.IncomingCalls(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodCallHierarchyOutgoingCalls: // request
		defer logger.Debug(MethodCallHierarchyOutgoingCalls, zap.Error(err))

		var params CallHierarchyOutgoingCallsParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.OutgoingCalls(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodSemanticTokensFull: // request
		defer logger.Debug(MethodSemanticTokensFull, zap.Error(err))

		var params SemanticTokensParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.SemanticTokensFull(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodSemanticTokensFullDelta: // request
		defer logger.Debug(MethodSemanticTokensFullDelta, zap.Error(err))

		var params SemanticTokensDeltaParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.SemanticTokensFullDelta(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodSemanticTokensRange: // request
		defer logger.Debug(MethodSemanticTokensRange, zap.Error(err))

		var params SemanticTokensRangeParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.SemanticTokensRange(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodSemanticTokensRefresh: // request
		defer logger.Debug(MethodSemanticTokensRefresh, zap.Error(err))

		if len(req.Params()) > 0 {
			return true, reply(ctx, nil, fmt.Errorf("expected no params: %w", jsonrpc2.ErrInvalidParams))
		}

		err := server.SemanticTokensRefresh(ctx)

		return true, reply(ctx, nil, err)

	case MethodLinkedEditingRange: // request
		defer logger.Debug(MethodLinkedEditingRange, zap.Error(err))

		var params LinkedEditingRangeParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.LinkedEditingRange(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodMoniker: // request
		defer logger.Debug(MethodMoniker, zap.Error(err))

		var params MonikerParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := server.Moniker(ctx, &params)

		return true, reply(ctx, resp, err)

	default:
		return false, nil
	}
}

// Server represents a Language Server Protocol server.
type Server interface {
	Initialize(ctx context.Context, params *InitializeParams) (result *InitializeResult, err error)
	Initialized(ctx context.Context, params *InitializedParams) (err error)
	Shutdown(ctx context.Context) (err error)
	Exit(ctx context.Context) (err error)
	WorkDoneProgressCancel(ctx context.Context, params *WorkDoneProgressCancelParams) (err error)
	LogTrace(ctx context.Context, params *LogTraceParams) (err error)
	SetTrace(ctx context.Context, params *SetTraceParams) (err error)
	CodeAction(ctx context.Context, params *CodeActionParams) (result []CodeAction, err error)
	CodeLens(ctx context.Context, params *CodeLensParams) (result []CodeLens, err error)
	CodeLensResolve(ctx context.Context, params *CodeLens) (result *CodeLens, err error)
	ColorPresentation(ctx context.Context, params *ColorPresentationParams) (result []ColorPresentation, err error)
	Completion(ctx context.Context, params *CompletionParams) (result *CompletionList, err error)
	CompletionResolve(ctx context.Context, params *CompletionItem) (result *CompletionItem, err error)
	Declaration(ctx context.Context, params *DeclarationParams) (result []Location /* Declaration | DeclarationLink[] | null */, err error)
	Definition(ctx context.Context, params *DefinitionParams) (result []Location /* Definition | DefinitionLink[] | null */, err error)
	DidChange(ctx context.Context, params *DidChangeTextDocumentParams) (err error)
	DidChangeConfiguration(ctx context.Context, params *DidChangeConfigurationParams) (err error)
	DidChangeWatchedFiles(ctx context.Context, params *DidChangeWatchedFilesParams) (err error)
	DidChangeWorkspaceFolders(ctx context.Context, params *DidChangeWorkspaceFoldersParams) (err error)
	DidClose(ctx context.Context, params *DidCloseTextDocumentParams) (err error)
	DidOpen(ctx context.Context, params *DidOpenTextDocumentParams) (err error)
	DidSave(ctx context.Context, params *DidSaveTextDocumentParams) (err error)
	DocumentColor(ctx context.Context, params *DocumentColorParams) (result []ColorInformation, err error)
	DocumentHighlight(ctx context.Context, params *DocumentHighlightParams) (result []DocumentHighlight, err error)
	DocumentLink(ctx context.Context, params *DocumentLinkParams) (result []DocumentLink, err error)
	DocumentLinkResolve(ctx context.Context, params *DocumentLink) (result *DocumentLink, err error)
	DocumentSymbol(ctx context.Context, params *DocumentSymbolParams) (result []interface{} /* []SymbolInformation | []DocumentSymbol */, err error)
	ExecuteCommand(ctx context.Context, params *ExecuteCommandParams) (result interface{}, err error)
	FoldingRanges(ctx context.Context, params *FoldingRangeParams) (result []FoldingRange, err error)
	Formatting(ctx context.Context, params *DocumentFormattingParams) (result []TextEdit, err error)
	Hover(ctx context.Context, params *HoverParams) (result *Hover, err error)
	Implementation(ctx context.Context, params *ImplementationParams) (result []Location, err error)
	OnTypeFormatting(ctx context.Context, params *DocumentOnTypeFormattingParams) (result []TextEdit, err error)
	PrepareRename(ctx context.Context, params *PrepareRenameParams) (result *Range, err error)
	RangeFormatting(ctx context.Context, params *DocumentRangeFormattingParams) (result []TextEdit, err error)
	References(ctx context.Context, params *ReferenceParams) (result []Location, err error)
	Rename(ctx context.Context, params *RenameParams) (result *WorkspaceEdit, err error)
	SignatureHelp(ctx context.Context, params *SignatureHelpParams) (result *SignatureHelp, err error)
	Symbols(ctx context.Context, params *WorkspaceSymbolParams) (result []SymbolInformation, err error)
	TypeDefinition(ctx context.Context, params *TypeDefinitionParams) (result []Location, err error)
	WillSave(ctx context.Context, params *WillSaveTextDocumentParams) (err error)
	WillSaveWaitUntil(ctx context.Context, params *WillSaveTextDocumentParams) (result []TextEdit, err error)
	ShowDocument(ctx context.Context, params *ShowDocumentParams) (result *ShowDocumentResult, err error)
	WillCreateFiles(ctx context.Context, params *CreateFilesParams) (result *WorkspaceEdit, err error)
	DidCreateFiles(ctx context.Context, params *CreateFilesParams) (err error)
	WillRenameFiles(ctx context.Context, params *RenameFilesParams) (result *WorkspaceEdit, err error)
	DidRenameFiles(ctx context.Context, params *RenameFilesParams) (err error)
	WillDeleteFiles(ctx context.Context, params *DeleteFilesParams) (result *WorkspaceEdit, err error)
	DidDeleteFiles(ctx context.Context, params *DeleteFilesParams) (err error)
	CodeLensRefresh(ctx context.Context) (err error)
	PrepareCallHierarchy(ctx context.Context, params *CallHierarchyPrepareParams) (result []CallHierarchyItem, err error)
	IncomingCalls(ctx context.Context, params *CallHierarchyIncomingCallsParams) (result []CallHierarchyIncomingCall, err error)
	OutgoingCalls(ctx context.Context, params *CallHierarchyOutgoingCallsParams) (result []CallHierarchyOutgoingCall, err error)
	SemanticTokensFull(ctx context.Context, params *SemanticTokensParams) (result *SemanticTokens, err error)
	SemanticTokensFullDelta(ctx context.Context, params *SemanticTokensDeltaParams) (result interface{} /* SemanticTokens | SemanticTokensDelta */, err error)
	SemanticTokensRange(ctx context.Context, params *SemanticTokensRangeParams) (result *SemanticTokens, err error)
	SemanticTokensRefresh(ctx context.Context) (err error)
	LinkedEditingRange(ctx context.Context, params *LinkedEditingRangeParams) (result *LinkedEditingRanges, err error)
	Moniker(ctx context.Context, params *MonikerParams) (result []Moniker, err error)
	Request(ctx context.Context, method string, params interface{}) (result interface{}, err error)
}

// list of server methods.
const (
	// MethodCancelRequest method name of "$/cancelRequest".
	MethodCancelRequest = "$/cancelRequest"

	// MethodInitialize method name of "initialize".
	MethodInitialize = "initialize"

	// MethodInitialized method name of "initialized".
	MethodInitialized = "initialized"

	// MethodShutdown method name of "shutdown".
	MethodShutdown = "shutdown"

	// MethodExit method name of "exit".
	MethodExit = "exit"

	// MethodWorkDoneProgressCancel method name of "window/workDoneProgress/cancel".
	MethodWorkDoneProgressCancel = "window/workDoneProgress/cancel"

	// MethodLogTrace method name of "$/logTrace".
	MethodLogTrace = "$/logTrace"

	// MethodSetTrace method name of "$/setTrace".
	MethodSetTrace = "$/setTrace"

	// MethodTextDocumentCodeAction method name of "textDocument/codeAction".
	MethodTextDocumentCodeAction = "textDocument/codeAction"

	// MethodTextDocumentCodeLens method name of "textDocument/codeLens".
	MethodTextDocumentCodeLens = "textDocument/codeLens"

	// MethodCodeLensResolve method name of "codeLens/resolve".
	MethodCodeLensResolve = "codeLens/resolve"

	// MethodTextDocumentColorPresentation method name of "textDocument/colorPresentation".
	MethodTextDocumentColorPresentation = "textDocument/colorPresentation"

	// MethodTextDocumentCompletion method name of "textDocument/completion".
	MethodTextDocumentCompletion = "textDocument/completion"

	// MethodCompletionItemResolve method name of "completionItem/resolve".
	MethodCompletionItemResolve = "completionItem/resolve"

	// MethodTextDocumentDeclaration method name of "textDocument/declaration".
	MethodTextDocumentDeclaration = "textDocument/declaration"

	// MethodTextDocumentDefinition method name of "textDocument/definition".
	MethodTextDocumentDefinition = "textDocument/definition"

	// MethodTextDocumentDidChange method name of "textDocument/didChange".
	MethodTextDocumentDidChange = "textDocument/didChange"

	// MethodWorkspaceDidChangeConfiguration method name of "workspace/didChangeConfiguration".
	MethodWorkspaceDidChangeConfiguration = "workspace/didChangeConfiguration"

	// MethodWorkspaceDidChangeWatchedFiles method name of "workspace/didChangeWatchedFiles".
	MethodWorkspaceDidChangeWatchedFiles = "workspace/didChangeWatchedFiles"

	// MethodWorkspaceDidChangeWorkspaceFolders method name of "workspace/didChangeWorkspaceFolders".
	MethodWorkspaceDidChangeWorkspaceFolders = "workspace/didChangeWorkspaceFolders"

	// MethodTextDocumentDidClose method name of "textDocument/didClose".
	MethodTextDocumentDidClose = "textDocument/didClose"

	// MethodTextDocumentDidOpen method name of "textDocument/didOpen".
	MethodTextDocumentDidOpen = "textDocument/didOpen"

	// MethodTextDocumentDidSave method name of "textDocument/didSave".
	MethodTextDocumentDidSave = "textDocument/didSave"

	// MethodTextDocumentDocumentColor method name of"textDocument/documentColor".
	MethodTextDocumentDocumentColor = "textDocument/documentColor"

	// MethodTextDocumentDocumentHighlight method name of "textDocument/documentHighlight".
	MethodTextDocumentDocumentHighlight = "textDocument/documentHighlight"

	// MethodTextDocumentDocumentLink method name of "textDocument/documentLink".
	MethodTextDocumentDocumentLink = "textDocument/documentLink"

	// MethodDocumentLinkResolve method name of "documentLink/resolve".
	MethodDocumentLinkResolve = "documentLink/resolve"

	// MethodTextDocumentDocumentSymbol method name of "textDocument/documentSymbol".
	MethodTextDocumentDocumentSymbol = "textDocument/documentSymbol"

	// MethodWorkspaceExecuteCommand method name of "workspace/executeCommand".
	MethodWorkspaceExecuteCommand = "workspace/executeCommand"

	// MethodTextDocumentFoldingRange method name of "textDocument/foldingRange".
	MethodTextDocumentFoldingRange = "textDocument/foldingRange"

	// MethodTextDocumentFormatting method name of "textDocument/formatting".
	MethodTextDocumentFormatting = "textDocument/formatting"

	// MethodTextDocumentHover method name of "textDocument/hover".
	MethodTextDocumentHover = "textDocument/hover"

	// MethodTextDocumentImplementation method name of "textDocument/implementation".
	MethodTextDocumentImplementation = "textDocument/implementation"

	// MethodTextDocumentOnTypeFormatting method name of "textDocument/onTypeFormatting".
	MethodTextDocumentOnTypeFormatting = "textDocument/onTypeFormatting"

	// MethodTextDocumentPrepareRename method name of "textDocument/prepareRename".
	MethodTextDocumentPrepareRename = "textDocument/prepareRename"

	// MethodTextDocumentRangeFormatting method name of "textDocument/rangeFormatting".
	MethodTextDocumentRangeFormatting = "textDocument/rangeFormatting"

	// MethodTextDocumentReferences method name of "textDocument/references".
	MethodTextDocumentReferences = "textDocument/references"

	// MethodTextDocumentRename method name of "textDocument/rename".
	MethodTextDocumentRename = "textDocument/rename"

	// MethodTextDocumentSignatureHelp method name of "textDocument/signatureHelp".
	MethodTextDocumentSignatureHelp = "textDocument/signatureHelp"

	// MethodWorkspaceSymbol method name of "workspace/symbol".
	MethodWorkspaceSymbol = "workspace/symbol"

	// MethodTextDocumentTypeDefinition method name of "textDocument/typeDefinition".
	MethodTextDocumentTypeDefinition = "textDocument/typeDefinition"

	// MethodTextDocumentWillSave method name of "textDocument/willSave".
	MethodTextDocumentWillSave = "textDocument/willSave"

	// MethodTextDocumentWillSaveWaitUntil method name of "textDocument/willSaveWaitUntil".
	MethodTextDocumentWillSaveWaitUntil = "textDocument/willSaveWaitUntil"

	// MethodShowDocument method name of "window/showDocument".
	MethodShowDocument = "window/showDocument"

	// MethodWillCreateFiles method name of "workspace/willCreateFiles".
	MethodWillCreateFiles = "workspace/willCreateFiles"

	// MethodDidCreateFiles method name of "workspace/didCreateFiles".
	MethodDidCreateFiles = "workspace/didCreateFiles"

	// MethodWillRenameFiles method name of "workspace/willRenameFiles".
	MethodWillRenameFiles = "workspace/willRenameFiles"

	// MethodDidRenameFiles method name of "workspace/didRenameFiles".
	MethodDidRenameFiles = "workspace/didRenameFiles"

	// MethodWillDeleteFiles method name of "workspace/willDeleteFiles".
	MethodWillDeleteFiles = "workspace/willDeleteFiles"

	// MethodDidDeleteFiles method name of "workspace/didDeleteFiles".
	MethodDidDeleteFiles = "workspace/didDeleteFiles"

	// MethodCodeLensRefresh method name of "workspace/codeLens/refresh".
	MethodCodeLensRefresh = "workspace/codeLens/refresh"

	// MethodTextDocumentPrepareCallHierarchy method name of "textDocument/prepareCallHierarchy".
	MethodTextDocumentPrepareCallHierarchy = "textDocument/prepareCallHierarchy"

	// MethodCallHierarchyIncomingCalls method name of "callHierarchy/incomingCalls".
	MethodCallHierarchyIncomingCalls = "callHierarchy/incomingCalls"

	// MethodCallHierarchyOutgoingCalls method name of "callHierarchy/outgoingCalls".
	MethodCallHierarchyOutgoingCalls = "callHierarchy/outgoingCalls"

	// MethodSemanticTokensFull method name of "textDocument/semanticTokens/full".
	MethodSemanticTokensFull = "textDocument/semanticTokens/full"

	// MethodSemanticTokensFullDelta method name of "textDocument/semanticTokens/full/delta".
	MethodSemanticTokensFullDelta = "textDocument/semanticTokens/full/delta"

	// MethodSemanticTokensRange method name of "textDocument/semanticTokens/range".
	MethodSemanticTokensRange = "textDocument/semanticTokens/range"

	// MethodSemanticTokensRefresh method name of "workspace/semanticTokens/refresh".
	MethodSemanticTokensRefresh = "workspace/semanticTokens/refresh"

	// MethodLinkedEditingRange method name of "textDocument/linkedEditingRange".
	MethodLinkedEditingRange = "textDocument/linkedEditingRange"

	// MethodMoniker method name of "textDocument/moniker".
	MethodMoniker = "textDocument/moniker"
)

// server implements a Language Server Protocol server.
type server struct {
	jsonrpc2.Conn

	logger *zap.Logger
}

var _ Server = (*server)(nil)

// Initialize sents the request as the first request from the client to the server.
//
// If the server receives a request or notification before the initialize request it should act as follows:
//
// - For a request the response should be an error with code: -32002. The message can be picked by the server.
// - Notifications should be dropped, except for the exit notification. This will allow the exit of a server without an initialize request.
//
// Until the server has responded to the initialize request with an InitializeResult, the client
// must not send any additional requests or notifications to the server.
// In addition the server is not allowed to send any requests or notifications to the client until
// it has responded with an InitializeResult, with the exception that during the initialize request
// the server is allowed to send the notifications window/showMessage, window/logMessage and telemetry/event
// as well as the window/showMessageRequest request to the client.
func (s *server) Initialize(ctx context.Context, params *InitializeParams) (_ *InitializeResult, err error) {
	s.logger.Debug("call " + MethodInitialize)
	defer s.logger.Debug("end "+MethodInitialize, zap.Error(err))

	var result *InitializeResult
	if err := Call(ctx, s.Conn, MethodInitialize, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Initialized sends the notification from the client to the server after the client received the result of the
// initialize request but before the client is sending any other request or notification to the server.
//
// The server can use the initialized notification for example to dynamically register capabilities.
// The initialized notification may only be sent once.
func (s *server) Initialized(ctx context.Context, params *InitializedParams) (err error) {
	s.logger.Debug("notify " + MethodInitialized)
	defer s.logger.Debug("end "+MethodInitialized, zap.Error(err))

	return s.Conn.Notify(ctx, MethodInitialized, params)
}

// Shutdown sents the request from the client to the server.
//
// It asks the server to shut down, but to not exit (otherwise the response might not be delivered correctly to the client).
// There is a separate exit notification that asks the server to exit.
//
// Clients must not sent any notifications other than `exit` or requests to a server to which they have sent a shutdown requests.
// If a server receives requests after a shutdown request those requests should be errored with `InvalidRequest`.
func (s *server) Shutdown(ctx context.Context) (err error) {
	s.logger.Debug("call " + MethodShutdown)
	defer s.logger.Debug("end "+MethodShutdown, zap.Error(err))

	return Call(ctx, s.Conn, MethodShutdown, nil, nil)
}

// Exit a notification to ask the server to exit its process.
//
// The server should exit with success code 0 if the shutdown request has been received before; otherwise with error code 1.
func (s *server) Exit(ctx context.Context) (err error) {
	s.logger.Debug("notify " + MethodExit)
	defer s.logger.Debug("end "+MethodExit, zap.Error(err))

	return s.Conn.Notify(ctx, MethodExit, nil)
}

// LogTrace a notification to log the trace of the server’s execution.
//
// The amount and content of these notifications depends on the current trace configuration.
//
// If trace is "off", the server should not send any logTrace notification. If trace is "message",
// the server should not add the "verbose" field in the LogTraceParams.
//
// @since 3.16.0.
func (s *server) LogTrace(ctx context.Context, params *LogTraceParams) (err error) {
	s.logger.Debug("notify " + MethodLogTrace)
	defer s.logger.Debug("end "+MethodLogTrace, zap.Error(err))

	return s.Conn.Notify(ctx, MethodLogTrace, params)
}

// SetTrace a notification that should be used by the client to modify the trace setting of the server.
//
// @since 3.16.0.
func (s *server) SetTrace(ctx context.Context, params *SetTraceParams) (err error) {
	s.logger.Debug("notify " + MethodSetTrace)
	defer s.logger.Debug("end "+MethodSetTrace, zap.Error(err))

	return s.Conn.Notify(ctx, MethodSetTrace, params)
}

// WorkDoneProgressCancel is the sends notification from the client to the server to cancel a progress initiated on the
// server side using the "window/workDoneProgress/create".
func (s *server) WorkDoneProgressCancel(ctx context.Context, params *WorkDoneProgressCancelParams) (err error) {
	s.logger.Debug("call " + MethodWorkDoneProgressCancel)
	defer s.logger.Debug("end "+MethodWorkDoneProgressCancel, zap.Error(err))

	return s.Conn.Notify(ctx, MethodWorkDoneProgressCancel, params)
}

// CodeAction sends the request is from the client to the server to compute commands for a given text document and range.
//
// These commands are typically code fixes to either fix problems or to beautify/refactor code. The result of a `textDocument/codeAction`
// request is an array of `Command` literals which are typically presented in the user interface.
//
// To ensure that a server is useful in many clients the commands specified in a code actions should be handled by the
// server and not by the client (see `workspace/executeCommand` and `ServerCapabilities.executeCommandProvider`).
// If the client supports providing edits with a code action then the mode should be used.
func (s *server) CodeAction(ctx context.Context, params *CodeActionParams) (result []CodeAction, err error) {
	s.logger.Debug("call " + MethodTextDocumentCodeAction)
	defer s.logger.Debug("end "+MethodTextDocumentCodeAction, zap.Error(err))

	if err := Call(ctx, s.Conn, MethodTextDocumentCodeAction, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// CodeLens sends the request from the client to the server to compute code lenses for a given text document.
func (s *server) CodeLens(ctx context.Context, params *CodeLensParams) (result []CodeLens, err error) {
	s.logger.Debug("call " + MethodTextDocumentCodeLens)
	defer s.logger.Debug("end "+MethodTextDocumentCodeLens, zap.Error(err))

	if err := Call(ctx, s.Conn, MethodTextDocumentCodeLens, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// CodeLensResolve sends the request from the client to the server to resolve the command for a given code lens item.
func (s *server) CodeLensResolve(ctx context.Context, params *CodeLens) (_ *CodeLens, err error) {
	s.logger.Debug("call " + MethodCodeLensResolve)
	defer s.logger.Debug("end "+MethodCodeLensResolve, zap.Error(err))

	var result *CodeLens
	if err := Call(ctx, s.Conn, MethodCodeLensResolve, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// ColorPresentation sends the request from the client to the server to obtain a list of presentations for a color value at a given location.
//
// Clients can use the result to
//
// - modify a color reference.
// - show in a color picker and let users pick one of the presentations.
func (s *server) ColorPresentation(ctx context.Context, params *ColorPresentationParams) (result []ColorPresentation, err error) {
	s.logger.Debug("call " + MethodTextDocumentColorPresentation)
	defer s.logger.Debug("end "+MethodTextDocumentColorPresentation, zap.Error(err))

	if err := Call(ctx, s.Conn, MethodTextDocumentColorPresentation, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Completion sends the request from the client to the server to compute completion items at a given cursor position.
//
// Completion items are presented in the IntelliSense user interface.
// If computing full completion items is expensive, servers can additionally provide a handler for the completion item resolve request (‘completionItem/resolve’).
//
// This request is sent when a completion item is selected in the user interface.
// A typical use case is for example: the ‘textDocument/completion’ request doesn’t fill in the documentation property
// for returned completion items since it is expensive to compute. When the item is selected in the user interface then
// a ‘completionItem/resolve’ request is sent with the selected completion item as a parameter.
//
// The returned completion item should have the documentation property filled in. The request can delay the computation of
// the `detail` and `documentation` properties. However, properties that are needed for the initial sorting and filtering,
// like `sortText`, `filterText`, `insertText`, and `textEdit` must be provided in the `textDocument/completion` response and must not be changed during resolve.
func (s *server) Completion(ctx context.Context, params *CompletionParams) (_ *CompletionList, err error) {
	s.logger.Debug("call " + MethodTextDocumentCompletion)
	defer s.logger.Debug("end "+MethodTextDocumentCompletion, zap.Error(err))

	var result *CompletionList
	if err := Call(ctx, s.Conn, MethodTextDocumentCompletion, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// CompletionResolve sends the request from the client to the server to resolve additional information for a given completion item.
func (s *server) CompletionResolve(ctx context.Context, params *CompletionItem) (_ *CompletionItem, err error) {
	s.logger.Debug("call " + MethodCompletionItemResolve)
	defer s.logger.Debug("end "+MethodCompletionItemResolve, zap.Error(err))

	var result *CompletionItem
	if err := Call(ctx, s.Conn, MethodCompletionItemResolve, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Declaration sends the request from the client to the server to resolve the declaration location of a symbol at a given text document position.
//
// The result type LocationLink[] got introduce with version 3.14.0 and depends in the corresponding client capability `clientCapabilities.textDocument.declaration.linkSupport`.
//
// @since 3.14.0.
func (s *server) Declaration(ctx context.Context, params *DeclarationParams) (result []Location, err error) {
	s.logger.Debug("call " + MethodTextDocumentDeclaration)
	defer s.logger.Debug("end "+MethodTextDocumentDeclaration, zap.Error(err))

	if err := Call(ctx, s.Conn, MethodTextDocumentDeclaration, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Definition sends the request from the client to the server to resolve the definition location of a symbol at a given text document position.
//
// The result type `[]LocationLink` got introduce with version 3.14.0 and depends in the corresponding client capability `clientCapabilities.textDocument.definition.linkSupport`.
//
// @since 3.14.0.
func (s *server) Definition(ctx context.Context, params *DefinitionParams) (result []Location, err error) {
	s.logger.Debug("call " + MethodTextDocumentDefinition)
	defer s.logger.Debug("end "+MethodTextDocumentDefinition, zap.Error(err))

	if err := Call(ctx, s.Conn, MethodTextDocumentDefinition, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// DidChange sends the notification from the client to the server to signal changes to a text document.
//
// In 2.0 the shape of the params has changed to include proper version numbers and language ids.
func (s *server) DidChange(ctx context.Context, params *DidChangeTextDocumentParams) (err error) {
	s.logger.Debug("notify " + MethodTextDocumentDidChange)
	defer s.logger.Debug("end "+MethodTextDocumentDidChange, zap.Error(err))

	return s.Conn.Notify(ctx, MethodTextDocumentDidChange, params)
}

// DidChangeConfiguration sends the notification from the client to the server to signal the change of configuration settings.
func (s *server) DidChangeConfiguration(ctx context.Context, params *DidChangeConfigurationParams) (err error) {
	s.logger.Debug("call " + MethodWorkspaceDidChangeConfiguration)
	defer s.logger.Debug("end "+MethodWorkspaceDidChangeConfiguration, zap.Error(err))

	return s.Conn.Notify(ctx, MethodWorkspaceDidChangeConfiguration, params)
}

// DidChangeWatchedFiles sends the notification from the client to the server when the client detects changes to files watched by the language client.
//
// It is recommended that servers register for these file events using the registration mechanism.
// In former implementations clients pushed file events without the server actively asking for it.
func (s *server) DidChangeWatchedFiles(ctx context.Context, params *DidChangeWatchedFilesParams) (err error) {
	s.logger.Debug("call " + MethodWorkspaceDidChangeWatchedFiles)
	defer s.logger.Debug("end "+MethodWorkspaceDidChangeWatchedFiles, zap.Error(err))

	return s.Conn.Notify(ctx, MethodWorkspaceDidChangeWatchedFiles, params)
}

// DidChangeWorkspaceFolders sents the notification from the client to the server to inform the server about workspace folder configuration changes.
//
// The notification is sent by default if both ServerCapabilities/workspace/workspaceFolders and ClientCapabilities/workspace/workspaceFolders are true;
// or if the server has registered itself to receive this notification.
// To register for the workspace/didChangeWorkspaceFolders send a client/registerCapability request from the server to the client.
//
// The registration parameter must have a registrations item of the following form, where id is a unique id used to unregister the capability (the example uses a UUID).
func (s *server) DidChangeWorkspaceFolders(ctx context.Context, params *DidChangeWorkspaceFoldersParams) (err error) {
	s.logger.Debug("call " + MethodWorkspaceDidChangeWorkspaceFolders)
	defer s.logger.Debug("end "+MethodWorkspaceDidChangeWorkspaceFolders, zap.Error(err))

	return s.Conn.Notify(ctx, MethodWorkspaceDidChangeWorkspaceFolders, params)
}

// DidClose sends the notification from the client to the server when the document got closed in the client.
//
// The document’s truth now exists where the document’s Uri points to (e.g. if the document’s Uri is a file Uri the truth now exists on disk).
// As with the open notification the close notification is about managing the document’s content.
// Receiving a close notification doesn’t mean that the document was open in an editor before.
//
// A close notification requires a previous open notification to be sent.
// Note that a server’s ability to fulfill requests is independent of whether a text document is open or closed.
func (s *server) DidClose(ctx context.Context, params *DidCloseTextDocumentParams) (err error) {
	s.logger.Debug("call " + MethodTextDocumentDidClose)
	defer s.logger.Debug("end "+MethodTextDocumentDidClose, zap.Error(err))

	return s.Conn.Notify(ctx, MethodTextDocumentDidClose, params)
}

// DidOpen sends the open notification from the client to the server to signal newly opened text documents.
//
// The document’s truth is now managed by the client and the server must not try to read the document’s truth using the document’s Uri.
// Open in this sense means it is managed by the client. It doesn’t necessarily mean that its content is presented in an editor.
//
// An open notification must not be sent more than once without a corresponding close notification send before.
// This means open and close notification must be balanced and the max open count for a particular textDocument is one.
// Note that a server’s ability to fulfill requests is independent of whether a text document is open or closed.
func (s *server) DidOpen(ctx context.Context, params *DidOpenTextDocumentParams) (err error) {
	s.logger.Debug("call " + MethodTextDocumentDidOpen)
	defer s.logger.Debug("end "+MethodTextDocumentDidOpen, zap.Error(err))

	return s.Conn.Notify(ctx, MethodTextDocumentDidOpen, params)
}

// DidSave sends the notification from the client to the server when the document was saved in the client.
func (s *server) DidSave(ctx context.Context, params *DidSaveTextDocumentParams) (err error) {
	s.logger.Debug("call " + MethodTextDocumentDidSave)
	defer s.logger.Debug("end "+MethodTextDocumentDidSave, zap.Error(err))

	return s.Conn.Notify(ctx, MethodTextDocumentDidSave, params)
}

// DocumentColor sends the request from the client to the server to list all color references found in a given text document.
//
// Along with the range, a color value in RGB is returned.
//
// Clients can use the result to decorate color references in an editor.
// For example:
//
// - Color boxes showing the actual color next to the reference
// - Show a color picker when a color reference is edited.
func (s *server) DocumentColor(ctx context.Context, params *DocumentColorParams) (result []ColorInformation, err error) {
	s.logger.Debug("call " + MethodTextDocumentDocumentColor)
	defer s.logger.Debug("end "+MethodTextDocumentDocumentColor, zap.Error(err))

	if err := Call(ctx, s.Conn, MethodTextDocumentDocumentColor, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// DocumentHighlight sends the request is from the client to the server to resolve a document highlights for a given text document position.
//
// For programming languages this usually highlights all references to the symbol scoped to this file.
// However we kept ‘textDocument/documentHighlight’ and ‘textDocument/references’ separate requests since the first one is allowed to be more fuzzy.
//
// Symbol matches usually have a `DocumentHighlightKind` of `Read` or `Write` whereas fuzzy or textual matches use `Text` as the kind.
func (s *server) DocumentHighlight(ctx context.Context, params *DocumentHighlightParams) (result []DocumentHighlight, err error) {
	s.logger.Debug("call " + MethodTextDocumentDocumentHighlight)
	defer s.logger.Debug("end "+MethodTextDocumentDocumentHighlight, zap.Error(err))

	if err := Call(ctx, s.Conn, MethodTextDocumentDocumentHighlight, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// DocumentLink sends the request from the client to the server to request the location of links in a document.
func (s *server) DocumentLink(ctx context.Context, params *DocumentLinkParams) (result []DocumentLink, err error) {
	s.logger.Debug("call " + MethodTextDocumentDocumentLink)
	defer s.logger.Debug("end "+MethodTextDocumentDocumentLink, zap.Error(err))

	if err := Call(ctx, s.Conn, MethodTextDocumentDocumentLink, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// DocumentLinkResolve sends the request from the client to the server to resolve the target of a given document link.
func (s *server) DocumentLinkResolve(ctx context.Context, params *DocumentLink) (_ *DocumentLink, err error) {
	s.logger.Debug("call " + MethodDocumentLinkResolve)
	defer s.logger.Debug("end "+MethodDocumentLinkResolve, zap.Error(err))

	var result *DocumentLink
	if err := Call(ctx, s.Conn, MethodDocumentLinkResolve, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// DocumentSymbol sends the request from the client to the server to return a flat list of all symbols found in a given text document.
//
// Neither the symbol’s location range nor the symbol’s container name should be used to infer a hierarchy.
func (s *server) DocumentSymbol(ctx context.Context, params *DocumentSymbolParams) (result []interface{}, err error) {
	s.logger.Debug("call " + MethodTextDocumentDocumentSymbol)
	defer s.logger.Debug("end "+MethodTextDocumentDocumentSymbol, zap.Error(err))

	if err := Call(ctx, s.Conn, MethodTextDocumentDocumentSymbol, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// ExecuteCommand sends the request from the client to the server to trigger command execution on the server.
//
// In most cases the server creates a `WorkspaceEdit` structure and applies the changes to the workspace using the
// request `workspace/applyEdit` which is sent from the server to the client.
func (s *server) ExecuteCommand(ctx context.Context, params *ExecuteCommandParams) (result interface{}, err error) {
	s.logger.Debug("call " + MethodWorkspaceExecuteCommand)
	defer s.logger.Debug("end "+MethodWorkspaceExecuteCommand, zap.Error(err))

	if err := Call(ctx, s.Conn, MethodWorkspaceExecuteCommand, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// FoldingRanges sends the request from the client to the server to return all folding ranges found in a given text document.
//
// @since version 3.10.0.
func (s *server) FoldingRanges(ctx context.Context, params *FoldingRangeParams) (result []FoldingRange, err error) {
	s.logger.Debug("call " + MethodTextDocumentFoldingRange)
	defer s.logger.Debug("end "+MethodTextDocumentFoldingRange, zap.Error(err))

	if err := Call(ctx, s.Conn, MethodTextDocumentFoldingRange, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Formatting sends the request from the client to the server to format a whole document.
func (s *server) Formatting(ctx context.Context, params *DocumentFormattingParams) (result []TextEdit, err error) {
	s.logger.Debug("call " + MethodTextDocumentFormatting)
	defer s.logger.Debug("end "+MethodTextDocumentFormatting, zap.Error(err))

	if err := Call(ctx, s.Conn, MethodTextDocumentFormatting, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Hover sends the request is from the client to the server to request hover information at a given text document position.
func (s *server) Hover(ctx context.Context, params *HoverParams) (_ *Hover, err error) {
	s.logger.Debug("call " + MethodTextDocumentHover)
	defer s.logger.Debug("end "+MethodTextDocumentHover, zap.Error(err))

	var result *Hover
	if err := Call(ctx, s.Conn, MethodTextDocumentHover, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Implementation sends the request from the client to the server to resolve the implementation location of a symbol at a given text document position.
//
// The result type `[]LocationLink` got introduce with version 3.14.0 and depends in the corresponding client capability `clientCapabilities.implementation.typeDefinition.linkSupport`.
func (s *server) Implementation(ctx context.Context, params *ImplementationParams) (result []Location, err error) {
	s.logger.Debug("call " + MethodTextDocumentImplementation)
	defer s.logger.Debug("end "+MethodTextDocumentImplementation, zap.Error(err))

	if err := Call(ctx, s.Conn, MethodTextDocumentImplementation, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// OnTypeFormatting sends the request from the client to the server to format parts of the document during typing.
func (s *server) OnTypeFormatting(ctx context.Context, params *DocumentOnTypeFormattingParams) (result []TextEdit, err error) {
	s.logger.Debug("call " + MethodTextDocumentOnTypeFormatting)
	defer s.logger.Debug("end "+MethodTextDocumentOnTypeFormatting, zap.Error(err))

	if err := Call(ctx, s.Conn, MethodTextDocumentOnTypeFormatting, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// PrepareRename sends the request from the client to the server to setup and test the validity of a rename operation at a given location.
//
// @since version 3.12.0.
func (s *server) PrepareRename(ctx context.Context, params *PrepareRenameParams) (result *Range, err error) {
	s.logger.Debug("call " + MethodTextDocumentPrepareRename)
	defer s.logger.Debug("end "+MethodTextDocumentPrepareRename, zap.Error(err))

	if err := Call(ctx, s.Conn, MethodTextDocumentPrepareRename, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// RangeFormatting sends the request from the client to the server to format a given range in a document.
func (s *server) RangeFormatting(ctx context.Context, params *DocumentRangeFormattingParams) (result []TextEdit, err error) {
	s.logger.Debug("call " + MethodTextDocumentRangeFormatting)
	defer s.logger.Debug("end "+MethodTextDocumentRangeFormatting, zap.Error(err))

	if err := Call(ctx, s.Conn, MethodTextDocumentRangeFormatting, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// References sends the request from the client to the server to resolve project-wide references for the symbol denoted by the given text document position.
func (s *server) References(ctx context.Context, params *ReferenceParams) (result []Location, err error) {
	s.logger.Debug("call " + MethodTextDocumentReferences)
	defer s.logger.Debug("end "+MethodTextDocumentReferences, zap.Error(err))

	if err := Call(ctx, s.Conn, MethodTextDocumentReferences, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Rename sends the request from the client to the server to perform a workspace-wide rename of a symbol.
func (s *server) Rename(ctx context.Context, params *RenameParams) (result *WorkspaceEdit, err error) {
	s.logger.Debug("call " + MethodTextDocumentRename)
	defer s.logger.Debug("end "+MethodTextDocumentRename, zap.Error(err))

	if err := Call(ctx, s.Conn, MethodTextDocumentRename, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// SignatureHelp sends the request from the client to the server to request signature information at a given cursor position.
func (s *server) SignatureHelp(ctx context.Context, params *SignatureHelpParams) (_ *SignatureHelp, err error) {
	s.logger.Debug("call " + MethodTextDocumentSignatureHelp)
	defer s.logger.Debug("end "+MethodTextDocumentSignatureHelp, zap.Error(err))

	var result *SignatureHelp
	if err := Call(ctx, s.Conn, MethodTextDocumentSignatureHelp, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Symbols sends the request from the client to the server to list project-wide symbols matching the query string.
func (s *server) Symbols(ctx context.Context, params *WorkspaceSymbolParams) (result []SymbolInformation, err error) {
	s.logger.Debug("call " + MethodWorkspaceSymbol)
	defer s.logger.Debug("end "+MethodWorkspaceSymbol, zap.Error(err))

	if err := Call(ctx, s.Conn, MethodWorkspaceSymbol, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// TypeDefinition sends the request from the client to the server to resolve the type definition location of a symbol at a given text document position.
//
// The result type `[]LocationLink` got introduce with version 3.14.0 and depends in the corresponding client capability `clientCapabilities.textDocument.typeDefinition.linkSupport`.
//
// @since version 3.6.0.
func (s *server) TypeDefinition(ctx context.Context, params *TypeDefinitionParams) (result []Location, err error) {
	s.logger.Debug("call " + MethodTextDocumentTypeDefinition)
	defer s.logger.Debug("end "+MethodTextDocumentTypeDefinition, zap.Error(err))

	if err := Call(ctx, s.Conn, MethodTextDocumentTypeDefinition, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// WillSave sends the notification from the client to the server before the document is actually saved.
func (s *server) WillSave(ctx context.Context, params *WillSaveTextDocumentParams) (err error) {
	s.logger.Debug("call " + MethodTextDocumentWillSave)
	defer s.logger.Debug("end "+MethodTextDocumentWillSave, zap.Error(err))

	return s.Conn.Notify(ctx, MethodTextDocumentWillSave, params)
}

// WillSaveWaitUntil sends the request from the client to the server before the document is actually saved.
//
// The request can return an array of TextEdits which will be applied to the text document before it is saved.
// Please note that clients might drop results if computing the text edits took too long or if a server constantly fails on this request.
// This is done to keep the save fast and reliable.
func (s *server) WillSaveWaitUntil(ctx context.Context, params *WillSaveTextDocumentParams) (result []TextEdit, err error) {
	s.logger.Debug("call " + MethodTextDocumentWillSaveWaitUntil)
	defer s.logger.Debug("end "+MethodTextDocumentWillSaveWaitUntil, zap.Error(err))

	if err := Call(ctx, s.Conn, MethodTextDocumentWillSaveWaitUntil, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// ShowDocument sends the request from a server to a client to ask the client to display a particular document in the user interface.
//
// @since 3.16.0.
func (s *server) ShowDocument(ctx context.Context, params *ShowDocumentParams) (result *ShowDocumentResult, err error) {
	s.logger.Debug("call " + MethodShowDocument)
	defer s.logger.Debug("end "+MethodShowDocument, zap.Error(err))

	if err := Call(ctx, s.Conn, MethodShowDocument, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// WillCreateFiles sends the will create files request is sent from the client to the server before files are actually created as long as the creation is triggered from within the client.
//
// The request can return a WorkspaceEdit which will be applied to workspace before the files are created.
//
// Please note that clients might drop results if computing the edit took too long or if a server constantly fails on this request. This is done to keep creates fast and reliable.
//
// @since 3.16.0.
func (s *server) WillCreateFiles(ctx context.Context, params *CreateFilesParams) (result *WorkspaceEdit, err error) {
	s.logger.Debug("call " + MethodWillCreateFiles)
	defer s.logger.Debug("end "+MethodWillCreateFiles, zap.Error(err))

	if err := Call(ctx, s.Conn, MethodWillCreateFiles, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// DidCreateFiles sends the did create files notification is sent from the client to the server when files were created from within the client.
//
// @since 3.16.0.
func (s *server) DidCreateFiles(ctx context.Context, params *CreateFilesParams) (err error) {
	s.logger.Debug("call " + MethodDidCreateFiles)
	defer s.logger.Debug("end "+MethodDidCreateFiles, zap.Error(err))

	return s.Conn.Notify(ctx, MethodDidCreateFiles, params)
}

// WillRenameFiles sends the will rename files request is sent from the client to the server before files are actually renamed as long as the rename is triggered from within the client.
//
// The request can return a WorkspaceEdit which will be applied to workspace before the files are renamed.
//
// Please note that clients might drop results if computing the edit took too long or if a server constantly fails on this request. This is done to keep renames fast and reliable.
//
// @since 3.16.0.
func (s *server) WillRenameFiles(ctx context.Context, params *RenameFilesParams) (result *WorkspaceEdit, err error) {
	s.logger.Debug("call " + MethodWillRenameFiles)
	defer s.logger.Debug("end "+MethodWillRenameFiles, zap.Error(err))

	if err := Call(ctx, s.Conn, MethodWillRenameFiles, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// DidRenameFiles sends the did rename files notification is sent from the client to the server when files were renamed from within the client.
//
// @since 3.16.0.
func (s *server) DidRenameFiles(ctx context.Context, params *RenameFilesParams) (err error) {
	s.logger.Debug("call " + MethodDidRenameFiles)
	defer s.logger.Debug("end "+MethodDidRenameFiles, zap.Error(err))

	return s.Conn.Notify(ctx, MethodDidRenameFiles, params)
}

// WillDeleteFiles sends the will delete files request is sent from the client to the server before files are actually deleted as long as the deletion is triggered from within the client.
//
// The request can return a WorkspaceEdit which will be applied to workspace before the files are deleted.
//
// Please note that clients might drop results if computing the edit took too long or if a server constantly fails on this request. This is done to keep deletes fast and reliable.
//
// @since 3.16.0.
func (s *server) WillDeleteFiles(ctx context.Context, params *DeleteFilesParams) (result *WorkspaceEdit, err error) {
	s.logger.Debug("call " + MethodWillDeleteFiles)
	defer s.logger.Debug("end "+MethodWillDeleteFiles, zap.Error(err))

	if err := Call(ctx, s.Conn, MethodWillDeleteFiles, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// DidDeleteFiles sends the did delete files notification is sent from the client to the server when files were deleted from within the client.
//
// @since 3.16.0.
func (s *server) DidDeleteFiles(ctx context.Context, params *DeleteFilesParams) (err error) {
	s.logger.Debug("call " + MethodDidDeleteFiles)
	defer s.logger.Debug("end "+MethodDidDeleteFiles, zap.Error(err))

	return s.Conn.Notify(ctx, MethodDidDeleteFiles, params)
}

// CodeLensRefresh sent from the server to the client.
//
// Servers can use it to ask clients to refresh the code lenses currently shown in editors.
// As a result the client should ask the server to recompute the code lenses for these editors.
// This is useful if a server detects a configuration change which requires a re-calculation of all code lenses.
//
// Note that the client still has the freedom to delay the re-calculation of the code lenses if for example an editor is currently not visible.
//
// @since 3.16.0.
func (s *server) CodeLensRefresh(ctx context.Context) (err error) {
	s.logger.Debug("call " + MethodCodeLensRefresh)
	defer s.logger.Debug("end "+MethodCodeLensRefresh, zap.Error(err))

	return Call(ctx, s.Conn, MethodCodeLensRefresh, nil, nil)
}

// PrepareCallHierarchy sent from the client to the server to return a call hierarchy for the language element of given text document positions.
//
// The call hierarchy requests are executed in two steps:
//  1. first a call hierarchy item is resolved for the given text document position
//  2. for a call hierarchy item the incoming or outgoing call hierarchy items are resolved.
//
// @since 3.16.0.
func (s *server) PrepareCallHierarchy(ctx context.Context, params *CallHierarchyPrepareParams) (result []CallHierarchyItem, err error) {
	s.logger.Debug("call " + MethodTextDocumentPrepareCallHierarchy)
	defer s.logger.Debug("end "+MethodTextDocumentPrepareCallHierarchy, zap.Error(err))

	if err := Call(ctx, s.Conn, MethodTextDocumentPrepareCallHierarchy, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// IncomingCalls is the request is sent from the client to the server to resolve incoming calls for a given call hierarchy item.
//
// The request doesn’t define its own client and server capabilities. It is only issued if a server registers for the "textDocument/prepareCallHierarchy" request.
//
// @since 3.16.0.
func (s *server) IncomingCalls(ctx context.Context, params *CallHierarchyIncomingCallsParams) (result []CallHierarchyIncomingCall, err error) {
	s.logger.Debug("call " + MethodCallHierarchyIncomingCalls)
	defer s.logger.Debug("end "+MethodCallHierarchyIncomingCalls, zap.Error(err))

	if err := Call(ctx, s.Conn, MethodCallHierarchyIncomingCalls, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// OutgoingCalls is the request is sent from the client to the server to resolve outgoing calls for a given call hierarchy item.
//
// The request doesn’t define its own client and server capabilities. It is only issued if a server registers for the "textDocument/prepareCallHierarchy" request.
//
// @since 3.16.0.
func (s *server) OutgoingCalls(ctx context.Context, params *CallHierarchyOutgoingCallsParams) (result []CallHierarchyOutgoingCall, err error) {
	s.logger.Debug("call " + MethodCallHierarchyOutgoingCalls)
	defer s.logger.Debug("end "+MethodCallHierarchyOutgoingCalls, zap.Error(err))

	if err := Call(ctx, s.Conn, MethodCallHierarchyOutgoingCalls, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// SemanticTokensFull is the request is sent from the client to the server to resolve semantic tokens for a given file.
//
// Semantic tokens are used to add additional color information to a file that depends on language specific symbol information.
//
// A semantic token request usually produces a large result. The protocol therefore supports encoding tokens with numbers.
//
// @since 3.16.0.
func (s *server) SemanticTokensFull(ctx context.Context, params *SemanticTokensParams) (result *SemanticTokens, err error) {
	s.logger.Debug("call " + MethodSemanticTokensFull)
	defer s.logger.Debug("end "+MethodSemanticTokensFull, zap.Error(err))

	if err := Call(ctx, s.Conn, MethodSemanticTokensFull, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// SemanticTokensFullDelta is the request is sent from the client to the server to resolve semantic token delta for a given file.
//
// Semantic tokens are used to add additional color information to a file that depends on language specific symbol information.
//
// A semantic token request usually produces a large result. The protocol therefore supports encoding tokens with numbers.
//
// @since 3.16.0.
func (s *server) SemanticTokensFullDelta(ctx context.Context, params *SemanticTokensDeltaParams) (result interface{}, err error) {
	s.logger.Debug("call " + MethodSemanticTokensFullDelta)
	defer s.logger.Debug("end "+MethodSemanticTokensFullDelta, zap.Error(err))

	if err := Call(ctx, s.Conn, MethodSemanticTokensFullDelta, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// SemanticTokensRange is the request is sent from the client to the server to resolve semantic token delta for a given file.
//
// When a user opens a file it can be beneficial to only compute the semantic tokens for the visible range (faster rendering of the tokens in the user interface).
// If a server can compute these tokens faster than for the whole file it can provide a handler for the "textDocument/semanticTokens/range" request to handle this case special.
//
// Please note that if a client also announces that it will send the "textDocument/semanticTokens/range" server should implement this request as well to allow for flicker free scrolling and semantic coloring of a minimap.
//
// @since 3.16.0.
func (s *server) SemanticTokensRange(ctx context.Context, params *SemanticTokensRangeParams) (result *SemanticTokens, err error) {
	s.logger.Debug("call " + MethodSemanticTokensRange)
	defer s.logger.Debug("end "+MethodSemanticTokensRange, zap.Error(err))

	if err := Call(ctx, s.Conn, MethodSemanticTokensRange, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// SemanticTokensRefresh is sent from the server to the client. Servers can use it to ask clients to refresh the editors for which this server provides semantic tokens.
//
// As a result the client should ask the server to recompute the semantic tokens for these editors.
// This is useful if a server detects a project wide configuration change which requires a re-calculation of all semantic tokens.
//
// Note that the client still has the freedom to delay the re-calculation of the semantic tokens if for example an editor is currently not visible.
//
// @since 3.16.0.
func (s *server) SemanticTokensRefresh(ctx context.Context) (err error) {
	s.logger.Debug("call " + MethodSemanticTokensRefresh)
	defer s.logger.Debug("end "+MethodSemanticTokensRefresh, zap.Error(err))

	return Call(ctx, s.Conn, MethodSemanticTokensRefresh, nil, nil)
}

// LinkedEditingRange is the linked editing request is sent from the client to the server to return for a given position in a document the range of the symbol at the position and all ranges that have the same content.
//
// Optionally a word pattern can be returned to describe valid contents.
//
// A rename to one of the ranges can be applied to all other ranges if the new content is valid. If no result-specific word pattern is provided, the word pattern from the client’s language configuration is used.
//
// @since 3.16.0.
func (s *server) LinkedEditingRange(ctx context.Context, params *LinkedEditingRangeParams) (result *LinkedEditingRanges, err error) {
	s.logger.Debug("call " + MethodLinkedEditingRange)
	defer s.logger.Debug("end "+MethodLinkedEditingRange, zap.Error(err))

	if err := Call(ctx, s.Conn, MethodLinkedEditingRange, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Moniker is the request is sent from the client to the server to get the symbol monikers for a given text document position.
//
// An array of Moniker types is returned as response to indicate possible monikers at the given location.
//
// If no monikers can be calculated, an empty array or null should be returned.
//
// @since 3.16.0.
func (s *server) Moniker(ctx context.Context, params *MonikerParams) (result []Moniker, err error) {
	s.logger.Debug("call " + MethodMoniker)
	defer s.logger.Debug("end "+MethodMoniker, zap.Error(err))

	if err := Call(ctx, s.Conn, MethodMoniker, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Request sends a request from the client to the server that non-compliant with the Language Server Protocol specifications.
func (s *server) Request(ctx context.Context, method string, params interface{}) (interface{}, error) {
	s.logger.Debug("call " + method)
	defer s.logger.Debug("end " + method)

	var result interface{}
	if err := Call(ctx, s.Conn, method, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}
