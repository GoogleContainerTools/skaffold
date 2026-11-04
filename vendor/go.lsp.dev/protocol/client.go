// Copyright 2026 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

import (
	"context"

	"go.lsp.dev/jsonrpc2"
)

// Client is the LSP client interface: the set of requests and notifications a
// language client handles. Its method set is the authoritative shape implemented
// by [UnimplementedClient].
type Client interface {
	Progress(ctx context.Context, params *ProgressParams) error
	LogTrace(ctx context.Context, params *LogTraceParams) error
	RegisterCapability(ctx context.Context, params *RegistrationParams) error
	UnregisterCapability(ctx context.Context, params *UnregistrationParams) error
	ShowMessage(ctx context.Context, params *ShowMessageParams) error
	ShowMessageRequest(ctx context.Context, params *ShowMessageRequestParams) (*MessageActionItem, error)
	LogMessage(ctx context.Context, params *LogMessageParams) error
	ShowDocument(ctx context.Context, params *ShowDocumentParams) (*ShowDocumentResult, error)
	WorkDoneProgressCreate(ctx context.Context, params *WorkDoneProgressCreateParams) error
	Telemetry(ctx context.Context, params LSPAny) error
	PublishDiagnostics(ctx context.Context, params *PublishDiagnosticsParams) error
	Configuration(ctx context.Context, params *ConfigurationParams) ([]LSPAny, error)
	WorkspaceFolders(ctx context.Context) ([]WorkspaceFolder, error)
	ApplyEdit(ctx context.Context, params *ApplyWorkspaceEditParams) (*ApplyWorkspaceEditResult, error)
	CodeLensRefresh(ctx context.Context) error
	FoldingRangeRefresh(ctx context.Context) error
	SemanticTokensRefresh(ctx context.Context) error
	InlineValueRefresh(ctx context.Context) error
	InlayHintRefresh(ctx context.Context) error
	DiagnosticRefresh(ctx context.Context) error
	TextDocumentContentRefresh(ctx context.Context, params *TextDocumentContentRefreshParams) error
}

// ClientDispatcher returns a [Client] that dispatches LSP requests across conn.
func ClientDispatcher(conn jsonrpc2.Conn) Client {
	return &client{Conn: conn}
}

// ClientHandler returns a [jsonrpc2.Handler] that routes incoming requests to
// client, falling back to handler for unhandled methods.
//
// The fallback handler receives the original borrowed [jsonrpc2.Request]. Like
// any jsonrpc2 handler, it must copy the method/params or call
// [jsonrpc2.Request.Clone] before retaining request data past its return.
func ClientHandler(client Client, handler jsonrpc2.Handler) jsonrpc2.Handler {
	return func(ctx context.Context, req *jsonrpc2.Request) (any, error) {
		if ctx.Err() != nil {
			return nil, ErrRequestCancelled
		}

		result, handled, err := clientDispatch(ctx, client, req)
		if handled || err != nil {
			return result, err
		}

		return handler(ctx, req)
	}
}

// clientDispatch decodes req and invokes the matching [Client] method, reporting
// handled=true when req named a standard client method.
//
//nolint:funlen,gocyclo,cyclop
func clientDispatch(ctx context.Context, client Client, req *jsonrpc2.Request) (result any, handled bool, err error) {
	if ctx.Err() != nil {
		return nil, true, ErrRequestCancelled
	}

	switch req.Method() {
	case MethodProgress: // notification
		var params ProgressParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}

		return nil, true, client.Progress(ctx, &params)

	case MethodLogTrace: // notification
		var params LogTraceParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}

		return nil, true, client.LogTrace(ctx, &params)

	case MethodClientRegisterCapability: // request
		var params RegistrationParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}

		return nil, true, client.RegisterCapability(ctx, &params)

	case MethodClientUnregisterCapability: // request
		var params UnregistrationParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}

		return nil, true, client.UnregisterCapability(ctx, &params)

	case MethodWindowShowMessage: // notification
		var params ShowMessageParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}

		return nil, true, client.ShowMessage(ctx, &params)

	case MethodWindowShowMessageRequest: // request
		var params ShowMessageRequestParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := client.ShowMessageRequest(ctx, &params)

		return resp, true, err

	case MethodWindowLogMessage: // notification
		var params LogMessageParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}

		return nil, true, client.LogMessage(ctx, &params)

	case MethodWindowShowDocument: // request
		var params ShowDocumentParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := client.ShowDocument(ctx, &params)

		return resp, true, err

	case MethodWindowWorkDoneProgressCreate: // request
		var params WorkDoneProgressCreateParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}

		return nil, true, client.WorkDoneProgressCreate(ctx, &params)

	case MethodTelemetryEvent: // notification
		var params LSPAny
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}

		return nil, true, client.Telemetry(ctx, params)

	case MethodTextDocumentPublishDiagnostics: // notification
		var params PublishDiagnosticsParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}

		return nil, true, client.PublishDiagnostics(ctx, &params)

	case MethodWorkspaceConfiguration: // request
		var params ConfigurationParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := client.Configuration(ctx, &params)

		return resp, true, err

	case MethodWorkspaceWorkspaceFolders: // request
		resp, err := client.WorkspaceFolders(ctx)

		return resp, true, err

	case MethodWorkspaceApplyEdit: // request
		var params ApplyWorkspaceEditParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}
		resp, err := client.ApplyEdit(ctx, &params)

		return resp, true, err

	case MethodWorkspaceCodeLensRefresh: // request
		return nil, true, client.CodeLensRefresh(ctx)

	case MethodWorkspaceFoldingRangeRefresh: // request
		return nil, true, client.FoldingRangeRefresh(ctx)

	case MethodWorkspaceSemanticTokensRefresh: // request
		return nil, true, client.SemanticTokensRefresh(ctx)

	case MethodWorkspaceInlineValueRefresh: // request
		return nil, true, client.InlineValueRefresh(ctx)

	case MethodWorkspaceInlayHintRefresh: // request
		return nil, true, client.InlayHintRefresh(ctx)

	case MethodWorkspaceDiagnosticRefresh: // request
		return nil, true, client.DiagnosticRefresh(ctx)

	case MethodWorkspaceTextDocumentContentRefresh: // request
		var params TextDocumentContentRefreshParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, true, replyParseError(err)
		}

		return nil, true, client.TextDocumentContentRefresh(ctx, &params)

	default:
		return nil, false, nil
	}
}

// client is the [Client] dispatcher: it issues server->client requests and
// notifications over a jsonrpc2 connection.
type client struct {
	jsonrpc2.Conn
}

// compile-time assertion that *client satisfies Client.
var _ Client = (*client)(nil)

func (c *client) Progress(ctx context.Context, params *ProgressParams) error {
	return c.Conn.Notify(ctx, MethodProgress, params)
}

func (c *client) LogTrace(ctx context.Context, params *LogTraceParams) error {
	return c.Conn.Notify(ctx, MethodLogTrace, params)
}

func (c *client) RegisterCapability(ctx context.Context, params *RegistrationParams) error {
	return Call(ctx, c.Conn, MethodClientRegisterCapability, params, nil)
}

func (c *client) UnregisterCapability(ctx context.Context, params *UnregistrationParams) error {
	return Call(ctx, c.Conn, MethodClientUnregisterCapability, params, nil)
}

func (c *client) ShowMessage(ctx context.Context, params *ShowMessageParams) error {
	return c.Conn.Notify(ctx, MethodWindowShowMessage, params)
}

func (c *client) ShowMessageRequest(ctx context.Context, params *ShowMessageRequestParams) (*MessageActionItem, error) {
	var result *MessageActionItem
	if err := Call(ctx, c.Conn, MethodWindowShowMessageRequest, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *client) LogMessage(ctx context.Context, params *LogMessageParams) error {
	return c.Conn.Notify(ctx, MethodWindowLogMessage, params)
}

func (c *client) ShowDocument(ctx context.Context, params *ShowDocumentParams) (*ShowDocumentResult, error) {
	var result *ShowDocumentResult
	if err := Call(ctx, c.Conn, MethodWindowShowDocument, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *client) WorkDoneProgressCreate(ctx context.Context, params *WorkDoneProgressCreateParams) error {
	return Call(ctx, c.Conn, MethodWindowWorkDoneProgressCreate, params, nil)
}

func (c *client) Telemetry(ctx context.Context, params LSPAny) error {
	return c.Conn.Notify(ctx, MethodTelemetryEvent, params)
}

func (c *client) PublishDiagnostics(ctx context.Context, params *PublishDiagnosticsParams) error {
	return c.Conn.Notify(ctx, MethodTextDocumentPublishDiagnostics, params)
}

func (c *client) Configuration(ctx context.Context, params *ConfigurationParams) ([]LSPAny, error) {
	var result []LSPAny
	if err := Call(ctx, c.Conn, MethodWorkspaceConfiguration, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *client) WorkspaceFolders(ctx context.Context) ([]WorkspaceFolder, error) {
	var result []WorkspaceFolder
	if err := Call(ctx, c.Conn, MethodWorkspaceWorkspaceFolders, nil, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *client) ApplyEdit(ctx context.Context, params *ApplyWorkspaceEditParams) (*ApplyWorkspaceEditResult, error) {
	var result *ApplyWorkspaceEditResult
	if err := Call(ctx, c.Conn, MethodWorkspaceApplyEdit, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *client) CodeLensRefresh(ctx context.Context) error {
	return Call(ctx, c.Conn, MethodWorkspaceCodeLensRefresh, nil, nil)
}

func (c *client) FoldingRangeRefresh(ctx context.Context) error {
	return Call(ctx, c.Conn, MethodWorkspaceFoldingRangeRefresh, nil, nil)
}

func (c *client) SemanticTokensRefresh(ctx context.Context) error {
	return Call(ctx, c.Conn, MethodWorkspaceSemanticTokensRefresh, nil, nil)
}

func (c *client) InlineValueRefresh(ctx context.Context) error {
	return Call(ctx, c.Conn, MethodWorkspaceInlineValueRefresh, nil, nil)
}

func (c *client) InlayHintRefresh(ctx context.Context) error {
	return Call(ctx, c.Conn, MethodWorkspaceInlayHintRefresh, nil, nil)
}

func (c *client) DiagnosticRefresh(ctx context.Context) error {
	return Call(ctx, c.Conn, MethodWorkspaceDiagnosticRefresh, nil, nil)
}

func (c *client) TextDocumentContentRefresh(ctx context.Context, params *TextDocumentContentRefreshParams) error {
	return Call(ctx, c.Conn, MethodWorkspaceTextDocumentContentRefresh, params, nil)
}
