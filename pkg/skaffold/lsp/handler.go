/*
Copyright 2021 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime/debug"

	"github.com/spf13/afero"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/lint"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext/v2"
	schemautil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

var h Handler

// Handler is the server handler for the skaffold LSP.  It implements the LSP spec and supports connection over TCP
type Handler struct {
	documentManager *DocumentManager
	diagnostics     map[string][]protocol.Diagnostic
	initialized     bool
	conn            jsonrpc2.Conn
}

// NewHandler initializes a new Handler object
func NewHandler(conn jsonrpc2.Conn) *Handler {
	return &Handler{
		initialized:     false,
		documentManager: NewDocumentManager(afero.NewMemMapFs()),
		conn:            conn,
	}
}

func GetHandler(conn jsonrpc2.Conn, out io.Writer, opts config.SkaffoldOptions, createRunner func(ctx context.Context, out io.Writer, opts config.SkaffoldOptions) (runner.Runner, []schemautil.VersionedConfig, *runcontext.RunContext, error)) jsonrpc2.Handler {
	var runCtx *runcontext.RunContext
	h = *NewHandler(conn)
	util.Fs = afero.NewCacheOnReadFs(util.Fs, h.documentManager.memMapFs, 0)

	return func(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
		// Recover if a panic occurs in the handlers
		defer func() {
			err := recover()
			if err != nil {
				log.Entry(ctx).Errorf("recovered from panic at %s: %v\n", req.Method(), err)
				log.Entry(ctx).Errorf("stacktrace from panic: \n" + string(debug.Stack()))
			}
		}()
		log.Entry(ctx).Debugf("req.Method():  %q\n", req.Method())
		switch req.Method() {
		case protocol.MethodInitialize:
			var params protocol.InitializeParams
			json.Unmarshal(req.Params(), &params)
			log.Entry(ctx).Debugf("InitializeParams: %+v\n", params)
			log.Entry(ctx).Debugf("InitializeParams.Capabilities.TextDocument: %+v\n", params.Capabilities.TextDocument)
			// TODO(aaron-prindle) currently this only supports workspaces of length one (or the first of the list of workspaces)
			// This used to be a single workspace field/value before lsp spec changes and I only know how to open one workspace per
			// session in VSCode atm so this should be ok initially
			if len(params.WorkspaceFolders) == 0 {
				return fmt.Errorf("expected WorkspaceFolders to have at least one value, got 0")
			}
			err := os.Chdir(uriToFilename(uri.URI(params.WorkspaceFolders[0].URI)))
			if err != nil {
				return err
			}

			_, _, runCtx, err = createRunner(ctx, out, opts)
			if err != nil {
				return err
			}

			// TODO(aaron-prindle) might need some checks to verify the initialize requests supports these,
			// right now assuming VS Code w/ supported methods - seems like an ok assumption for now
			if err := reply(ctx, protocol.InitializeResult{
				Capabilities: protocol.ServerCapabilities{
					TextDocumentSync: protocol.TextDocumentSyncOptions{
						Change:    protocol.TextDocumentSyncKindFull,
						OpenClose: true,
						Save: &protocol.SaveOptions{
							IncludeText: true,
						},
					},
				},
			}, nil); err != nil {
				return err
			}
			h.initialized = true
			return nil
		case protocol.MethodInitialized:
			var params protocol.InitializedParams
			json.Unmarshal(req.Params(), &params)
			log.Entry(ctx).Debugf("InitializedParams: %+v\n", params)
			if err := lintFilesAndSendDiagnostics(ctx, runCtx, opts, req); err != nil {
				return err
			}
			return nil
		}
		if !h.initialized {
			reply(ctx, nil, jsonrpc2.Errorf(jsonrpc2.ServerNotInitialized, "not initialized yet"))
			return nil
		}

		switch req.Method() {
		case protocol.MethodTextDocumentDidOpen:
			var params protocol.DidOpenTextDocumentParams
			json.Unmarshal(req.Params(), &params)
			log.Entry(ctx).Debugf("DidOpenTextDocumentParams: %+v\n", params)
			documentURI := uriToFilename(params.TextDocument.URI)
			if documentURI != "" {
				if err := h.updateDocument(ctx, documentURI, params.TextDocument.Text); err != nil {
					return err
				}
				if err := lintFilesAndSendDiagnostics(ctx, runCtx, opts, req); err != nil {
					return err
				}
			}
		case protocol.MethodTextDocumentDidChange:
			var params protocol.DidChangeTextDocumentParams
			json.Unmarshal(req.Params(), &params)
			log.Entry(ctx).Debugf("DidChangeTextDocumentParams: %+v\n", params)
			documentURI := uriToFilename(params.TextDocument.URI)
			if documentURI != "" && len(params.ContentChanges) > 0 {
				if err := h.updateDocument(ctx, documentURI, params.ContentChanges[0].Text); err != nil {
					return err
				}
				if err := lintFilesAndSendDiagnostics(ctx, runCtx, opts, req); err != nil {
					return err
				}
			}
		case protocol.MethodTextDocumentDidSave:
			var params protocol.DidSaveTextDocumentParams
			json.Unmarshal(req.Params(), &params)
			log.Entry(ctx).Debugf("DidSaveTextDocumentParams: %+v\n", params)
			documentURI := uriToFilename(params.TextDocument.URI)
			if documentURI != "" {
				if err := h.updateDocument(ctx, documentURI, params.Text); err != nil {
					return err
				}
				if err := lintFilesAndSendDiagnostics(ctx, runCtx, opts, req); err != nil {
					return err
				}
			}
		// TODO(aaron-prindle) implement additional methods here - eg: lsp.MethodTextDocumentHover, etc.
		default:
			return nil
		}
		return nil
	}
}

func (h *Handler) updateDocument(ctx context.Context, documentURI, content string) error {
	h.documentManager.UpdateDocument(documentURI, content)
	log.Entry(ctx).Debugf("updated document for %q with %d chars\n", documentURI, len(content))
	return nil
}

func lintFilesAndSendDiagnostics(ctx context.Context, runCtx docker.Config,
	opts config.SkaffoldOptions, req jsonrpc2.Request) error {
	results, err := lint.GetAllLintResults(ctx, lint.Options{
		Filename:     opts.ConfigurationFile,
		RepoCacheDir: opts.RepoCacheDir,
		OutFormat:    lint.PlainTextOutput,
		Modules:      opts.ConfigurationFilter,
		Profiles:     opts.Profiles,
	}, runCtx)

	if err != nil {
		return err
	}

	var params protocol.TextDocumentPositionParams
	json.Unmarshal(req.Params(), &params)
	tmpDiags := make(map[string][]protocol.Diagnostic)
	for _, result := range results {
		diag := protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{Line: uint32(result.StartLine - 1), Character: uint32(result.StartColumn - 1)},
				// TODO(aaron-prindle) should implement and pass end range from lint as well (currently a hack and just flags to end of line vs end of flagged text)
				End: protocol.Position{Line: uint32(result.StartLine), Character: 0},
			},
			Severity: result.Rule.Severity,
			Code:     result.Rule.RuleID,
			Source:   result.AbsFilePath,
			Message:  result.Explanation,
		}
		if _, ok := tmpDiags[result.AbsFilePath]; ok {
			tmpDiags[result.AbsFilePath] = append(tmpDiags[result.AbsFilePath], diag)
			continue
		}
		tmpDiags[result.AbsFilePath] = []protocol.Diagnostic{diag}
	}
	h.diagnostics = tmpDiags

	if h.diagnostics != nil && len(h.diagnostics) > 0 {
		fmt.Fprintf(os.Stderr, "publishing diagnostics (%d).\n", len(h.diagnostics))
		for k, v := range h.diagnostics {
			fmt.Fprintf(os.Stderr, "> %s\n", k)
			h.conn.Notify(ctx, protocol.MethodTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
				URI:         uri.File(k),
				Diagnostics: v,
			})
		}
		h.diagnostics = map[string][]protocol.Diagnostic{}
	}
	return nil
}
