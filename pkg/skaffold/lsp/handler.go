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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/parser"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	schemautil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/validation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

var h Handler

// Handler is the server handler for the skaffold LSP.  It implements the LSP spec and supports connection over TCP
type Handler struct {
	documentManager *DocumentManager
	lastDiagnostics map[string][]protocol.Diagnostic
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

func sendValidationAndLintDiagnostics(ctx context.Context, opts config.SkaffoldOptions, out io.Writer, req jsonrpc2.Request, createRunner func(ctx context.Context, out io.Writer, opts config.SkaffoldOptions) (runner.Runner, []schemautil.VersionedConfig, *runcontext.RunContext, error)) error {
	isValidConfig := true
	diags, err := validateFiles(ctx, opts, req)
	if err != nil {
		return err
	}
	sendDiagnostics(ctx, diags)

	// there was a validation error found, config is invalid
	if len(diags) > 0 {
		isValidConfig = false
	}

	if isValidConfig {
		_, _, runCtx, err := createRunner(ctx, out, opts)
		if err != nil {
			return err
		}
		// TODO(aaron-prindle) files should be linted even if skaffold.yaml file is invalid (for the lint rules that is possible for)
		// currently this only lints when the config is valid
		diags, err = lintFiles(ctx, runCtx, opts, req)
		if err != nil {
			return err
		}
		sendDiagnostics(ctx, diags)
	}

	return nil
}

func GetHandler(conn jsonrpc2.Conn, out io.Writer, opts config.SkaffoldOptions, createRunner func(ctx context.Context, out io.Writer, opts config.SkaffoldOptions) (runner.Runner, []schemautil.VersionedConfig, *runcontext.RunContext, error)) jsonrpc2.Handler {
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
			// TODO(aaron-prindle) does workspace changing send a new 'initialize' or is there workspaceChange msg?  Need to make sure that is handled...
			// and we don't keep initialize workspace always
			err := os.Chdir(uriToFilename(uri.URI(params.WorkspaceFolders[0].URI)))
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
			return sendValidationAndLintDiagnostics(ctx, opts, out, req, createRunner)
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
				return sendValidationAndLintDiagnostics(ctx, opts, out, req, createRunner)
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
				return sendValidationAndLintDiagnostics(ctx, opts, out, req, createRunner)
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
				return sendValidationAndLintDiagnostics(ctx, opts, out, req, createRunner)
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

func convertErrorWithLocationsToResults(errs []validation.ErrorWithLocation) []lint.Result {
	results := []lint.Result{}
	for _, e := range errs {
		results = append(results,
			lint.Result{
				Rule: &lint.Rule{
					RuleID:   lint.ValidationError,
					Severity: protocol.DiagnosticSeverityError,
				},
				// TODO(aaron-prindle) currently there is dupe line and file information in the Explanation field, need to remove this for LSP
				Explanation: e.Error.Error(),
				AbsFilePath: e.Location.SourceFile,
				StartLine:   e.Location.StartLine,
				EndLine:     e.Location.EndLine,
				StartColumn: e.Location.StartColumn,
				EndColumn:   e.Location.EndColumn,
			})
	}
	return results
}

func sendDiagnostics(ctx context.Context, diags map[string][]protocol.Diagnostic) {
	// copy map to not mutate input
	tmpDiags := map[string][]protocol.Diagnostic{}
	for k, v := range diags {
		tmpDiags[k] = v
	}

	for k := range h.lastDiagnostics {
		if _, ok := tmpDiags[k]; !ok {
			tmpDiags[k] = []protocol.Diagnostic{}
		}
	}

	if len(tmpDiags) > 0 {
		fmt.Fprintf(os.Stderr, "publishing diagnostics (%d).\n", len(tmpDiags))
		for k, v := range tmpDiags {
			fmt.Fprintf(os.Stderr, "> %s\n", k)
			h.conn.Notify(ctx, protocol.MethodTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
				URI:         uri.File(k),
				Diagnostics: v,
			})
		}
	}
	h.lastDiagnostics = tmpDiags
}

func validateFiles(ctx context.Context,
	opts config.SkaffoldOptions, req jsonrpc2.Request) (map[string][]protocol.Diagnostic, error) {
	// TODO(aaron-prindle) currently lint checks only filesystem, instead need to check VFS w/ documentManager info (validation uses VFS currently NOT lint)

	// TODO(aaron-prindle) if invalid yaml and parser fails, need to handle that as well as a validation error vs server erroring
	// OR just show nothing in this case as that would make sense vs all RED text, perhaps should just error
	cfgs, err := parser.GetConfigSet(ctx, opts)
	if err != nil {
		return nil, err
	}
	vopts := validation.GetValidationOpts(opts)
	vopts.CheckDeploySource = true
	errs := validation.ProcessForLSP(cfgs, vopts)
	results := convertErrorWithLocationsToResults(errs)

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
	return tmpDiags, nil
}

func lintFiles(ctx context.Context, runCtx docker.Config,
	opts config.SkaffoldOptions, req jsonrpc2.Request) (map[string][]protocol.Diagnostic, error) {
	// TODO(aaron-prindle) currently lint checks only filesystem, instead need to check VFS w/ documentManager info
	// need to make sure something like k8a-manifest.yaml comes from afero VFS and not os FS always
	results, err := lint.GetAllLintResults(ctx, lint.Options{
		Filename:     opts.ConfigurationFile,
		RepoCacheDir: opts.RepoCacheDir,
		OutFormat:    lint.PlainTextOutput,
		Modules:      opts.ConfigurationFilter,
		Profiles:     opts.Profiles,
	}, runCtx)

	if err != nil {
		return nil, err
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
	return tmpDiags, nil
}
