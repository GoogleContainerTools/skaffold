// SPDX-FileCopyrightText: 2019 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

//go:build gojay
// +build gojay

package protocol

import (
	"bytes"
	"context"
	"fmt"

	"github.com/francoispqt/gojay"
	"go.uber.org/zap"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/pkg/xcontext"

	"go.lsp.dev/protocol/internal/gojaypool"
)

// ServerHandler jsonrpc2.Handler of Language Server Prococol Server.
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
		if err := gojay.Unsafe.Unmarshal(req.Params(), &params); err != nil {
			return replyParseError(ctx, reply, err)
		}

		resp, err := server.Request(ctx, req.Method(), params)
		return reply(ctx, resp, err)
	}

	return h
}

// serverDispatch implements jsonrpc2.Handler.
//nolint:funlen,gocognit
func serverDispatch(ctx context.Context, server Server, reply jsonrpc2.Replier, req jsonrpc2.Request) (handled bool, err error) {
	if ctx.Err() != nil {
		return true, reply(ctx, nil, ErrRequestCancelled)
	}

	dec := gojaypool.BorrowSizedDecoder(bytes.NewReader(req.Params()), len(req.Params()))
	defer dec.Release()
	logger := LoggerFromContext(ctx)

	switch req.Method() {
	case MethodInitialize: // request
		defer logger.Debug(MethodInitialize, zap.Error(err))

		var params InitializeParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.Initialize(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodInitialized: // notification
		defer logger.Debug(MethodInitialized, zap.Error(err))

		var params InitializedParams
		if err := dec.DecodeObject(&params); err != nil {
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
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		err := server.WorkDoneProgressCancel(ctx, &params)
		return true, reply(ctx, nil, err)

	case MethodLogTrace: // notification
		defer logger.Debug(MethodLogTrace, zap.Error(err))

		var params LogTraceParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		err := server.LogTrace(ctx, &params)
		return true, reply(ctx, nil, err)

	case MethodSetTrace: // notification
		defer logger.Debug(MethodSetTrace, zap.Error(err))

		var params SetTraceParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		err := server.SetTrace(ctx, &params)
		return true, reply(ctx, nil, err)

	case MethodTextDocumentCodeAction: // request
		defer logger.Debug(MethodTextDocumentCodeAction, zap.Error(err))

		var params CodeActionParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.CodeAction(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodTextDocumentCodeLens: // request
		defer logger.Debug(MethodTextDocumentCodeLens, zap.Error(err))

		var params CodeLensParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.CodeLens(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodCodeLensResolve: // request
		defer logger.Debug(MethodCodeLensResolve, zap.Error(err))

		var params CodeLens
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.CodeLensResolve(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodTextDocumentColorPresentation: // request
		defer logger.Debug(MethodTextDocumentColorPresentation, zap.Error(err))

		var params ColorPresentationParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.ColorPresentation(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodTextDocumentCompletion: // request
		defer logger.Debug(MethodTextDocumentCompletion, zap.Error(err))

		var params CompletionParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.Completion(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodCompletionItemResolve: // request
		defer logger.Debug(MethodCompletionItemResolve, zap.Error(err))

		var params CompletionItem
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.CompletionResolve(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodTextDocumentDeclaration: // request
		defer logger.Debug(MethodTextDocumentDeclaration, zap.Error(err))

		var params DeclarationParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.Declaration(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodTextDocumentDefinition: // request
		defer logger.Debug(MethodTextDocumentDefinition, zap.Error(err))

		var params DefinitionParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.Definition(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodTextDocumentDidChange: // notification
		defer logger.Debug(MethodTextDocumentDidChange, zap.Error(err))

		var params DidChangeTextDocumentParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		err := server.DidChange(ctx, &params)
		return true, reply(ctx, nil, err)

	case MethodWorkspaceDidChangeConfiguration: // notification
		defer logger.Debug(MethodWorkspaceDidChangeConfiguration, zap.Error(err))

		var params DidChangeConfigurationParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		err := server.DidChangeConfiguration(ctx, &params)
		return true, reply(ctx, nil, err)

	case MethodWorkspaceDidChangeWatchedFiles: // notification
		defer logger.Debug(MethodWorkspaceDidChangeWatchedFiles, zap.Error(err))

		var params DidChangeWatchedFilesParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		err := server.DidChangeWatchedFiles(ctx, &params)
		return true, reply(ctx, nil, err)

	case MethodWorkspaceDidChangeWorkspaceFolders: // notification
		defer logger.Debug(MethodWorkspaceDidChangeWorkspaceFolders, zap.Error(err))

		var params DidChangeWorkspaceFoldersParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		err := server.DidChangeWorkspaceFolders(ctx, &params)
		return true, reply(ctx, nil, err)

	case MethodTextDocumentDidClose: // notification
		defer logger.Debug(MethodTextDocumentDidClose, zap.Error(err))

		var params DidCloseTextDocumentParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		err := server.DidClose(ctx, &params)
		return true, reply(ctx, nil, err)

	case MethodTextDocumentDidOpen: // notification
		defer logger.Debug(MethodTextDocumentDidOpen, zap.Error(err))

		var params DidOpenTextDocumentParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		err := server.DidOpen(ctx, &params)
		return true, reply(ctx, nil, err)

	case MethodTextDocumentDidSave: // notification
		defer logger.Debug(MethodTextDocumentDidSave, zap.Error(err))

		var params DidSaveTextDocumentParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		err := server.DidSave(ctx, &params)
		return true, reply(ctx, nil, err)

	case MethodTextDocumentDocumentColor: // request
		defer logger.Debug(MethodTextDocumentDocumentColor, zap.Error(err))

		var params DocumentColorParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.DocumentColor(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodTextDocumentDocumentHighlight: // request
		defer logger.Debug(MethodTextDocumentDocumentHighlight, zap.Error(err))

		var params DocumentHighlightParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.DocumentHighlight(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodTextDocumentDocumentLink: // request
		defer logger.Debug(MethodTextDocumentDocumentLink, zap.Error(err))

		var params DocumentLinkParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.DocumentLink(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodDocumentLinkResolve: // request
		defer logger.Debug(MethodDocumentLinkResolve, zap.Error(err))

		var params DocumentLink
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.DocumentLinkResolve(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodTextDocumentDocumentSymbol: // request
		defer logger.Debug(MethodTextDocumentDocumentSymbol, zap.Error(err))

		var params DocumentSymbolParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.DocumentSymbol(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodWorkspaceExecuteCommand: // request
		defer logger.Debug(MethodWorkspaceExecuteCommand, zap.Error(err))

		var params ExecuteCommandParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.ExecuteCommand(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodTextDocumentFoldingRange: // request
		defer logger.Debug(MethodTextDocumentFoldingRange, zap.Error(err))

		var params FoldingRangeParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.FoldingRanges(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodTextDocumentFormatting: // request
		defer logger.Debug(MethodTextDocumentFormatting, zap.Error(err))

		var params DocumentFormattingParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.Formatting(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodTextDocumentHover: // request
		defer logger.Debug(MethodTextDocumentHover, zap.Error(err))

		var params HoverParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.Hover(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodTextDocumentImplementation: // request
		defer logger.Debug(MethodTextDocumentImplementation, zap.Error(err))

		var params ImplementationParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.Implementation(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodTextDocumentOnTypeFormatting: // request
		defer logger.Debug(MethodTextDocumentOnTypeFormatting, zap.Error(err))

		var params DocumentOnTypeFormattingParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.OnTypeFormatting(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodTextDocumentPrepareRename: // request
		defer logger.Debug(MethodTextDocumentPrepareRename, zap.Error(err))

		var params PrepareRenameParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.PrepareRename(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodTextDocumentRangeFormatting: // request
		defer logger.Debug(MethodTextDocumentRangeFormatting, zap.Error(err))

		var params DocumentRangeFormattingParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.RangeFormatting(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodTextDocumentReferences: // request
		defer logger.Debug(MethodTextDocumentReferences, zap.Error(err))

		var params ReferenceParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.References(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodTextDocumentRename: // request
		defer logger.Debug(MethodTextDocumentRename, zap.Error(err))

		var params RenameParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.Rename(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodTextDocumentSignatureHelp: // request
		defer logger.Debug(MethodTextDocumentSignatureHelp, zap.Error(err))

		var params SignatureHelpParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.SignatureHelp(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodWorkspaceSymbol: // request
		defer logger.Debug(MethodWorkspaceSymbol, zap.Error(err))

		var params WorkspaceSymbolParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.Symbols(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodTextDocumentTypeDefinition: // request
		defer logger.Debug(MethodTextDocumentTypeDefinition, zap.Error(err))

		var params TypeDefinitionParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.TypeDefinition(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodTextDocumentWillSave: // notification
		defer logger.Debug(MethodTextDocumentWillSave, zap.Error(err))

		var params WillSaveTextDocumentParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		err := server.WillSave(ctx, &params)
		return true, reply(ctx, nil, err)

	case MethodTextDocumentWillSaveWaitUntil: // request
		defer logger.Debug(MethodTextDocumentWillSaveWaitUntil, zap.Error(err))

		var params WillSaveTextDocumentParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.WillSaveWaitUntil(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodShowDocument: // request
		defer logger.Debug(MethodShowDocument, zap.Error(err))

		var params ShowDocumentParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.ShowDocument(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodWillCreateFiles: // request
		defer logger.Debug(MethodWillCreateFiles, zap.Error(err))

		var params CreateFilesParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.WillCreateFiles(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodDidCreateFiles: // notification
		defer logger.Debug(MethodDidCreateFiles, zap.Error(err))

		var params CreateFilesParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		err := server.DidCreateFiles(ctx, &params)
		return true, reply(ctx, nil, err)

	case MethodWillRenameFiles: // request
		defer logger.Debug(MethodWillRenameFiles, zap.Error(err))

		var params RenameFilesParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.WillRenameFiles(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodDidRenameFiles: // notification
		defer logger.Debug(MethodDidRenameFiles, zap.Error(err))

		var params RenameFilesParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		err := server.DidRenameFiles(ctx, &params)
		return true, reply(ctx, nil, err)

	case MethodWillDeleteFiles: // request
		defer logger.Debug(MethodWillDeleteFiles, zap.Error(err))

		var params DeleteFilesParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.WillDeleteFiles(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodDidDeleteFiles: // notification
		defer logger.Debug(MethodDidDeleteFiles, zap.Error(err))

		var params DeleteFilesParams
		if err := dec.DecodeObject(&params); err != nil {
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
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.PrepareCallHierarchy(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodCallHierarchyIncomingCalls: // request
		defer logger.Debug(MethodCallHierarchyIncomingCalls, zap.Error(err))

		var params CallHierarchyIncomingCallsParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.IncomingCalls(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodCallHierarchyOutgoingCalls: // request
		defer logger.Debug(MethodCallHierarchyOutgoingCalls, zap.Error(err))

		var params CallHierarchyOutgoingCallsParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.OutgoingCalls(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodSemanticTokensFull: // request
		defer logger.Debug(MethodSemanticTokensFull, zap.Error(err))

		var params SemanticTokensParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.SemanticTokensFull(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodSemanticTokensFullDelta: // request
		defer logger.Debug(MethodSemanticTokensFullDelta, zap.Error(err))

		var params SemanticTokensDeltaParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.SemanticTokensFullDelta(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodSemanticTokensRange: // request
		defer logger.Debug(MethodSemanticTokensRange, zap.Error(err))

		var params SemanticTokensRangeParams
		if err := dec.DecodeObject(&params); err != nil {
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
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.LinkedEditingRange(ctx, &params)
		return true, reply(ctx, resp, err)

	case MethodMoniker: // request
		defer logger.Debug(MethodMoniker, zap.Error(err))

		var params MonikerParams
		if err := dec.DecodeObject(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}
		resp, err := server.Moniker(ctx, &params)
		return true, reply(ctx, resp, err)

	default:
		return false, nil
	}
}
