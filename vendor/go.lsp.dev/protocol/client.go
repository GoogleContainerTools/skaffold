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

// ClientDispatcher returns a Client that dispatches LSP requests across the
// given jsonrpc2 connection.
func ClientDispatcher(conn jsonrpc2.Conn, logger *zap.Logger) Client {
	return &client{
		Conn:   conn,
		logger: logger,
	}
}

// ClientHandler handler of LSP client.
func ClientHandler(client Client, handler jsonrpc2.Handler) jsonrpc2.Handler {
	h := func(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
		if ctx.Err() != nil {
			xctx := xcontext.Detach(ctx)

			return reply(xctx, nil, ErrRequestCancelled)
		}

		handled, err := clientDispatch(ctx, client, reply, req)
		if handled || err != nil {
			return err
		}

		return handler(ctx, reply, req)
	}

	return h
}

// clientDispatch implements jsonrpc2.Handler.
//nolint:funlen,cyclop
func clientDispatch(ctx context.Context, client Client, reply jsonrpc2.Replier, req jsonrpc2.Request) (handled bool, err error) {
	if ctx.Err() != nil {
		return true, reply(ctx, nil, ErrRequestCancelled)
	}

	dec := json.NewDecoder(bytes.NewReader(req.Params()))
	logger := LoggerFromContext(ctx)

	switch req.Method() {
	case MethodProgress: // notification
		defer logger.Debug(MethodProgress, zap.Error(err))

		var params ProgressParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		err := client.Progress(ctx, &params)

		return true, reply(ctx, nil, err)

	case MethodWorkDoneProgressCreate: // request
		defer logger.Debug(MethodWorkDoneProgressCreate, zap.Error(err))

		var params WorkDoneProgressCreateParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		err := client.WorkDoneProgressCreate(ctx, &params)

		return true, reply(ctx, nil, err)

	case MethodWindowLogMessage: // notification
		defer logger.Debug(MethodWindowLogMessage, zap.Error(err))

		var params LogMessageParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		err := client.LogMessage(ctx, &params)

		return true, reply(ctx, nil, err)

	case MethodTextDocumentPublishDiagnostics: // notification
		defer logger.Debug(MethodTextDocumentPublishDiagnostics, zap.Error(err))

		var params PublishDiagnosticsParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		err := client.PublishDiagnostics(ctx, &params)

		return true, reply(ctx, nil, err)

	case MethodWindowShowMessage: // notification
		defer logger.Debug(MethodWindowShowMessage, zap.Error(err))

		var params ShowMessageParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		err := client.ShowMessage(ctx, &params)

		return true, reply(ctx, nil, err)

	case MethodWindowShowMessageRequest: // request
		defer logger.Debug(MethodWindowShowMessageRequest, zap.Error(err))

		var params ShowMessageRequestParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := client.ShowMessageRequest(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodTelemetryEvent: // notification
		defer logger.Debug(MethodTelemetryEvent, zap.Error(err))

		var params interface{}
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		err := client.Telemetry(ctx, &params)

		return true, reply(ctx, nil, err)

	case MethodClientRegisterCapability: // request
		defer logger.Debug(MethodClientRegisterCapability, zap.Error(err))

		var params RegistrationParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		err := client.RegisterCapability(ctx, &params)

		return true, reply(ctx, nil, err)

	case MethodClientUnregisterCapability: // request
		defer logger.Debug(MethodClientUnregisterCapability, zap.Error(err))

		var params UnregistrationParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		err := client.UnregisterCapability(ctx, &params)

		return true, reply(ctx, nil, err)

	case MethodWorkspaceApplyEdit: // request
		defer logger.Debug(MethodWorkspaceApplyEdit, zap.Error(err))

		var params ApplyWorkspaceEditParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := client.ApplyEdit(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodWorkspaceConfiguration: // request
		defer logger.Debug(MethodWorkspaceConfiguration, zap.Error(err))

		var params ConfigurationParams
		if err := dec.Decode(&params); err != nil {
			return true, replyParseError(ctx, reply, err)
		}

		resp, err := client.Configuration(ctx, &params)

		return true, reply(ctx, resp, err)

	case MethodWorkspaceWorkspaceFolders: // request
		defer logger.Debug(MethodWorkspaceWorkspaceFolders, zap.Error(err))

		if len(req.Params()) > 0 {
			return true, reply(ctx, nil, fmt.Errorf("expected no params: %w", jsonrpc2.ErrInvalidParams))
		}

		resp, err := client.WorkspaceFolders(ctx)

		return true, reply(ctx, resp, err)

	default:
		return false, nil
	}
}

// Client represents a Language Server Protocol client.
type Client interface {
	Progress(ctx context.Context, params *ProgressParams) (err error)
	WorkDoneProgressCreate(ctx context.Context, params *WorkDoneProgressCreateParams) (err error)
	LogMessage(ctx context.Context, params *LogMessageParams) (err error)
	PublishDiagnostics(ctx context.Context, params *PublishDiagnosticsParams) (err error)
	ShowMessage(ctx context.Context, params *ShowMessageParams) (err error)
	ShowMessageRequest(ctx context.Context, params *ShowMessageRequestParams) (result *MessageActionItem, err error)
	Telemetry(ctx context.Context, params interface{}) (err error)
	RegisterCapability(ctx context.Context, params *RegistrationParams) (err error)
	UnregisterCapability(ctx context.Context, params *UnregistrationParams) (err error)
	ApplyEdit(ctx context.Context, params *ApplyWorkspaceEditParams) (result bool, err error)
	Configuration(ctx context.Context, params *ConfigurationParams) (result []interface{}, err error)
	WorkspaceFolders(ctx context.Context) (result []WorkspaceFolder, err error)
}

// list of client methods.
const (
	// MethodProgress method name of "$/progress".
	MethodProgress = "$/progress"

	// MethodWorkDoneProgressCreate method name of "window/workDoneProgress/create".
	MethodWorkDoneProgressCreate = "window/workDoneProgress/create"

	// MethodWindowShowMessage method name of "window/showMessage".
	MethodWindowShowMessage = "window/showMessage"

	// MethodWindowShowMessageRequest method name of "window/showMessageRequest.
	MethodWindowShowMessageRequest = "window/showMessageRequest"

	// MethodWindowLogMessage method name of "window/logMessage.
	MethodWindowLogMessage = "window/logMessage"

	// MethodTelemetryEvent method name of "telemetry/event.
	MethodTelemetryEvent = "telemetry/event"

	// MethodClientRegisterCapability method name of "client/registerCapability.
	MethodClientRegisterCapability = "client/registerCapability"

	// MethodClientUnregisterCapability method name of "client/unregisterCapability.
	MethodClientUnregisterCapability = "client/unregisterCapability"

	// MethodTextDocumentPublishDiagnostics method name of "textDocument/publishDiagnostics.
	MethodTextDocumentPublishDiagnostics = "textDocument/publishDiagnostics"

	// MethodWorkspaceApplyEdit method name of "workspace/applyEdit.
	MethodWorkspaceApplyEdit = "workspace/applyEdit"

	// MethodWorkspaceConfiguration method name of "workspace/configuration.
	MethodWorkspaceConfiguration = "workspace/configuration"

	// MethodWorkspaceWorkspaceFolders method name of "workspace/workspaceFolders".
	MethodWorkspaceWorkspaceFolders = "workspace/workspaceFolders"
)

// client implements a Language Server Protocol client.
type client struct {
	jsonrpc2.Conn

	logger *zap.Logger
}

// compiler time check whether the Client implements ClientInterface interface.
var _ Client = (*client)(nil)

// Progress is the base protocol offers also support to report progress in a generic fashion.
//
// This mechanism can be used to report any kind of progress including work done progress (usually used to report progress in the user interface using a progress bar) and
// partial result progress to support streaming of results.
//
// @since 3.16.0.
func (c *client) Progress(ctx context.Context, params *ProgressParams) (err error) {
	c.logger.Debug("call " + MethodProgress)
	defer c.logger.Debug("end "+MethodProgress, zap.Error(err))

	return c.Conn.Notify(ctx, MethodProgress, params)
}

// WorkDoneProgressCreate sends the request is sent from the server to the client to ask the client to create a work done progress.
//
// @since 3.16.0.
func (c *client) WorkDoneProgressCreate(ctx context.Context, params *WorkDoneProgressCreateParams) (err error) {
	c.logger.Debug("call " + MethodWorkDoneProgressCreate)
	defer c.logger.Debug("end "+MethodWorkDoneProgressCreate, zap.Error(err))

	return Call(ctx, c.Conn, MethodWorkDoneProgressCreate, params, nil)
}

// LogMessage sends the notification from the server to the client to ask the client to log a particular message.
func (c *client) LogMessage(ctx context.Context, params *LogMessageParams) (err error) {
	c.logger.Debug("call " + MethodWindowLogMessage)
	defer c.logger.Debug("end "+MethodWindowLogMessage, zap.Error(err))

	return c.Conn.Notify(ctx, MethodWindowLogMessage, params)
}

// PublishDiagnostics sends the notification from the server to the client to signal results of validation runs.
//
// Diagnostics are “owned” by the server so it is the server’s responsibility to clear them if necessary. The following rule is used for VS Code servers that generate diagnostics:
//
// - if a language is single file only (for example HTML) then diagnostics are cleared by the server when the file is closed.
// - if a language has a project system (for example C#) diagnostics are not cleared when a file closes. When a project is opened all diagnostics for all files are recomputed (or read from a cache).
//
// When a file changes it is the server’s responsibility to re-compute diagnostics and push them to the client.
// If the computed set is empty it has to push the empty array to clear former diagnostics.
// Newly pushed diagnostics always replace previously pushed diagnostics. There is no merging that happens on the client side.
func (c *client) PublishDiagnostics(ctx context.Context, params *PublishDiagnosticsParams) (err error) {
	c.logger.Debug("call " + MethodTextDocumentPublishDiagnostics)
	defer c.logger.Debug("end "+MethodTextDocumentPublishDiagnostics, zap.Error(err))

	return c.Conn.Notify(ctx, MethodTextDocumentPublishDiagnostics, params)
}

// ShowMessage sends the notification from a server to a client to ask the
// client to display a particular message in the user interface.
func (c *client) ShowMessage(ctx context.Context, params *ShowMessageParams) (err error) {
	return c.Conn.Notify(ctx, MethodWindowShowMessage, params)
}

// ShowMessageRequest sends the request from a server to a client to ask the client to display a particular message in the user interface.
//
// In addition to the show message notification the request allows to pass actions and to wait for an answer from the client.
func (c *client) ShowMessageRequest(ctx context.Context, params *ShowMessageRequestParams) (_ *MessageActionItem, err error) {
	c.logger.Debug("call " + MethodWindowShowMessageRequest)
	defer c.logger.Debug("end "+MethodWindowShowMessageRequest, zap.Error(err))

	var result *MessageActionItem
	if err := Call(ctx, c.Conn, MethodWindowShowMessageRequest, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Telemetry sends the notification from the server to the client to ask the client to log a telemetry event.
func (c *client) Telemetry(ctx context.Context, params interface{}) (err error) {
	c.logger.Debug("call " + MethodTelemetryEvent)
	defer c.logger.Debug("end "+MethodTelemetryEvent, zap.Error(err))

	return c.Conn.Notify(ctx, MethodTelemetryEvent, params)
}

// RegisterCapability sends the request from the server to the client to register for a new capability on the client side.
//
// Not all clients need to support dynamic capability registration.
//
// A client opts in via the dynamicRegistration property on the specific client capabilities.
// A client can even provide dynamic registration for capability A but not for capability B (see TextDocumentClientCapabilities as an example).
func (c *client) RegisterCapability(ctx context.Context, params *RegistrationParams) (err error) {
	c.logger.Debug("call " + MethodClientRegisterCapability)
	defer c.logger.Debug("end "+MethodClientRegisterCapability, zap.Error(err))

	return Call(ctx, c.Conn, MethodClientRegisterCapability, params, nil)
}

// UnregisterCapability sends the request from the server to the client to unregister a previously registered capability.
func (c *client) UnregisterCapability(ctx context.Context, params *UnregistrationParams) (err error) {
	c.logger.Debug("call " + MethodClientUnregisterCapability)
	defer c.logger.Debug("end "+MethodClientUnregisterCapability, zap.Error(err))

	return Call(ctx, c.Conn, MethodClientUnregisterCapability, params, nil)
}

// ApplyEdit sends the request from the server to the client to modify resource on the client side.
func (c *client) ApplyEdit(ctx context.Context, params *ApplyWorkspaceEditParams) (result bool, err error) {
	c.logger.Debug("call " + MethodWorkspaceApplyEdit)
	defer c.logger.Debug("end "+MethodWorkspaceApplyEdit, zap.Error(err))

	if err := Call(ctx, c.Conn, MethodWorkspaceApplyEdit, params, &result); err != nil {
		return false, err
	}

	return result, nil
}

// Configuration sends the request from the server to the client to fetch configuration settings from the client.
//
// The request can fetch several configuration settings in one roundtrip.
// The order of the returned configuration settings correspond to the order of the
// passed ConfigurationItems (e.g. the first item in the response is the result for the first configuration item in the params).
func (c *client) Configuration(ctx context.Context, params *ConfigurationParams) (_ []interface{}, err error) {
	c.logger.Debug("call " + MethodWorkspaceConfiguration)
	defer c.logger.Debug("end "+MethodWorkspaceConfiguration, zap.Error(err))

	var result []interface{}
	if err := Call(ctx, c.Conn, MethodWorkspaceConfiguration, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// WorkspaceFolders sends the request from the server to the client to fetch the current open list of workspace folders.
//
// Returns null in the response if only a single file is open in the tool. Returns an empty array if a workspace is open but no folders are configured.
//
// @since 3.6.0.
func (c *client) WorkspaceFolders(ctx context.Context) (result []WorkspaceFolder, err error) {
	c.logger.Debug("call " + MethodWorkspaceWorkspaceFolders)
	defer c.logger.Debug("end "+MethodWorkspaceWorkspaceFolders, zap.Error(err))

	if err := Call(ctx, c.Conn, MethodWorkspaceWorkspaceFolders, nil, &result); err != nil {
		return nil, err
	}

	return result, nil
}
