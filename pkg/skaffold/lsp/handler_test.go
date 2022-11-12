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
	"io"
	"net"
	"os"
	"reflect"
	"testing"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

type callTest struct {
	description string
	method      string
	params      interface{}
	expected    interface{}
}

var callTests = []callTest{
	{
		description: "verify lsp 'initialize' method returns expected results",
		method:      protocol.MethodInitialize,
		params: protocol.InitializeParams{
			WorkspaceFolders: []protocol.WorkspaceFolder{
				{
					URI:  "overwritten by test",
					Name: "test name",
				},
			},
			Capabilities: protocol.ClientCapabilities{
				TextDocument: &protocol.TextDocumentClientCapabilities{
					// TODO(aaron-prindle) make sure the values here make sense (similar to VSCode, hit more edge cases (missing capability, versioning, old fields), etc.)
					PublishDiagnostics: &protocol.PublishDiagnosticsClientCapabilities{
						RelatedInformation: true,
						TagSupport: &protocol.PublishDiagnosticsClientCapabilitiesTagSupport{
							ValueSet: []protocol.DiagnosticTag{protocol.DiagnosticTagDeprecated},
						},
						VersionSupport:         true,
						CodeDescriptionSupport: true,
						DataSupport:            true,
					},
				},
			},
		},
		expected: protocol.InitializeResult{
			Capabilities: protocol.ServerCapabilities{
				TextDocumentSync: protocol.TextDocumentSyncOptions{
					Change:    protocol.TextDocumentSyncKindFull,
					OpenClose: true,
					Save: &protocol.SaveOptions{
						IncludeText: true,
					},
				},
			},
		},
	},
	// TODO(aaron-prindle) add error cases and full set of functionality (textDocument/* reqs, etc.)
}

func TestRequest(t *testing.T) {
	ctx := context.Background()
	a, b, done := prepare(ctx, t)
	defer done()
	workdir, err := os.Getwd()
	if err != nil {
		t.Fatalf("error getting working directory: %v", err)
	}
	for _, test := range callTests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			// need to marshal and unmarshal to get proper formatting for matching as structs not represented identically w/o this for verification
			expectedData, err := json.Marshal(test.expected)
			if err != nil {
				t.Fatalf("marshalling expected response to json failed: %v", err)
			}
			var expected protocol.InitializeResult
			json.Unmarshal(expectedData, &expected)
			test.expected = expected

			params := test.params
			if v, ok := test.params.(protocol.InitializeParams); ok {
				v.WorkspaceFolders[0].URI = "file://" + workdir
				params = v
			}

			results := test.newResults()
			if _, err := a.Call(ctx, test.method, params, results); err != nil {
				t.Fatalf("%v call failed: %v", test.method, err)
			}

			test.verifyResults(t.T, results)

			if _, err := b.Call(ctx, test.method, params, results); err != nil {
				t.Fatalf("%v call failed: %v", test.method, err)
			}
			test.verifyResults(t.T, results)
		})
	}
}

func (test *callTest) newResults() interface{} {
	switch e := test.expected.(type) {
	case []interface{}:
		var r []interface{}
		for _, v := range e {
			r = append(r, reflect.New(reflect.TypeOf(v)).Interface())
		}
		return r

	case nil:
		return nil

	default:
		return reflect.New(reflect.TypeOf(test.expected)).Interface()
	}
}

func (test *callTest) verifyResults(t *testing.T, results interface{}) {
	t.Helper()

	if results == nil {
		return
	}

	val := reflect.Indirect(reflect.ValueOf(results)).Interface()
	if !reflect.DeepEqual(val, test.expected) {
		t.Errorf("%v results are incorrect, got %+v expect %+v", test.method, val, test.expected)
	}
}

func prepare(ctx context.Context, t *testing.T) (a, b jsonrpc2.Conn, done func()) {
	t.Helper()

	// make a wait group that can be used to wait for the system to shut down
	aPipe, bPipe := net.Pipe()
	a = run(ctx, aPipe)
	b = run(ctx, bPipe)
	done = func() {
		a.Close()
		b.Close()
		<-a.Done()
		<-b.Done()
	}

	return a, b, done
}

func run(ctx context.Context, nc io.ReadWriteCloser) jsonrpc2.Conn {
	stream := jsonrpc2.NewStream(nc)
	conn := jsonrpc2.NewConn(stream)
	conn.Go(ctx, GetHandler(conn, nil, config.SkaffoldOptions{}, func(ctx context.Context, out io.Writer, opts config.SkaffoldOptions) (runner.Runner, []util.VersionedConfig, *runcontext.RunContext, error) {
		return nil, nil, nil, nil
	}))

	return conn
}
