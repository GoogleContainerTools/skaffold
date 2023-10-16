// Copyright 2023 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/google/go-containerregistry/pkg/registry"
)

func newCmdRegistry() *cobra.Command {
	cmd := &cobra.Command{
		Use: "registry",
	}
	cmd.AddCommand(newCmdServe())
	return cmd
}

func newCmdServe() *cobra.Command {
	var disk bool
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve an in-memory registry implementation",
		Long: `This sub-command serves an in-memory registry implementation on an automatically chosen port (or $PORT)

The command blocks while the server accepts pushes and pulls.

Contents are only stored in memory, and when the process exits, pushed data is lost.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			port := os.Getenv("PORT")
			if port == "" {
				port = "0"
			}
			listener, err := net.Listen("tcp", ":"+port)
			if err != nil {
				log.Fatalln(err)
			}
			porti := listener.Addr().(*net.TCPAddr).Port
			port = fmt.Sprintf("%d", porti)

			bh := registry.NewInMemoryBlobHandler()
			if disk {
				tmp := os.TempDir()
				log.Printf("storing blobs in %s", tmp)
				bh = registry.NewDiskBlobHandler(tmp)
			}

			s := &http.Server{
				ReadHeaderTimeout: 5 * time.Second, // prevent slowloris, quiet linter
				Handler:           registry.New(registry.WithBlobHandler(bh)),
			}
			log.Printf("serving on port %s", port)

			errCh := make(chan error)
			go func() { errCh <- s.Serve(listener) }()

			<-ctx.Done()
			log.Println("shutting down...")
			if err := s.Shutdown(ctx); err != nil {
				return err
			}

			if err := <-errCh; !errors.Is(err, http.ErrServerClosed) {
				return err
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&disk, "blobs-to-disk", false, "Store blobs on disk")
	cmd.Flags().MarkHidden("blobs-to-disk")
	return cmd
}
