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

package cmd

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"

	"github.com/spf13/cobra"
	"go.lsp.dev/jsonrpc2"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/lsp"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
)

var port int

func NewCmdLSP() *cobra.Command {
	return NewCmd("lsp").
		WithDescription("Starts skaffold LSP language server that provides lint suggestions to IDEs").
		WithCommonFlags().
		WithFlags([]*Flag{
			{Value: &port, Name: "port", DefValue: 4389,
				Usage: "port number that the skaffold lsp will expose for connections, '4389' is the default port value"}}).
		Hidden().
		NoArgs(doLSP)
}

func doLSP(ctx context.Context, out io.Writer) error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			if sig == os.Interrupt {
				os.Exit(0)
			}
		}
	}()
	// TODO(aaron-prindle) plumb context based termination as well on os.Kill and os.Interrupt
	addr := "localhost:" + strconv.Itoa(port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Entry(ctx).Fatalf("Could not bind to address %s: %v", addr, err)
	}
	defer listener.Close()

	fmt.Fprintf(out, "skaffold lsp listening for TCP connections on: %v\n", addr)
	connectionCount := 0
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Entry(ctx).Fatalf("encountered error attempting to accept tcp connection: %v", err)
		}
		connectionCount++
		log.Entry(ctx).Infof("skaffold lsp received incoming connection #%d\n", connectionCount)
		stream := jsonrpc2.NewStream(conn)
		jsonConn := jsonrpc2.NewConn(stream)
		jsonConn.Go(ctx, lsp.GetHandler(jsonConn, out, opts, createRunner))
	}
}
