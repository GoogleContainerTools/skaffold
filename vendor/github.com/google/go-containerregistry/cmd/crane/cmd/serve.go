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

func NewCmdRegistry() *cobra.Command {
	cmd := &cobra.Command{
		Use: "registry",
	}
	cmd.AddCommand(newCmdServe())
	return cmd
}

func newCmdServe() *cobra.Command {
	var address, disk string
	var blobsToDisk bool
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve a registry implementation",
		Long: `This sub-command serves a registry implementation on an automatically chosen port (:0), $PORT or --address

The command blocks while the server accepts pushes and pulls.

Contents are can be stored in memory (when the process exits, pushed data is lost.), and disk (--disk).`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			port := os.Getenv("PORT")
			if port == "" {
				port = "0"
			}
			listenOn := ":" + port
			if address != "" {
				listenOn = address
			}

			listener, err := net.Listen("tcp", listenOn)
			if err != nil {
				log.Fatalln(err)
			}
			porti := listener.Addr().(*net.TCPAddr).Port
			port = fmt.Sprintf("%d", porti)

			bh := registry.NewInMemoryBlobHandler()

			diskp := disk
			if cmd.Flags().Changed("blobs-to-disk") {
				if disk != "" {
					return fmt.Errorf("--disk and --blobs-to-disk can't be used together")
				}
				diskp, err = os.MkdirTemp(os.TempDir(), "craneregistry*")
				if err != nil {
					return err
				}
			}

			if diskp != "" {
				log.Printf("storing blobs in %s", diskp)
				bh = registry.NewDiskBlobHandler(diskp)
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
	// TODO: remove --blobs-to-disk in a future release.
	cmd.Flags().BoolVarP(&blobsToDisk, "blobs-to-disk", "", false, "Store blobs on disk on tmpdir")
	cmd.Flags().MarkHidden("blobs-to-disk")
	cmd.Flags().MarkDeprecated("blobs-to-disk", "and will stop working in a future release. use --disk=$(mktemp -d) instead.")
	cmd.Flags().StringVarP(&disk, "disk", "", "", "Path to a directory where blobs will be stored")
	cmd.Flags().StringVar(&address, "address", "", "Address to listen on")

	return cmd
}
