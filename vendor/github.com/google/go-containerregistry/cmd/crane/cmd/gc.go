// Copyright 2018 Google LLC All Rights Reserved.
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
	"fmt"
	"os"

	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/spf13/cobra"
)

func NewCmdLayout() *cobra.Command {
	cmd := &cobra.Command{
		Use: "layout",
	}
	cmd.AddCommand(newCmdGc())
	return cmd
}

// NewCmdGc creates a new cobra.Command for the pull subcommand.
func newCmdGc() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "gc OCI-LAYOUT",
		Short:  "Garbage collect unreferenced blobs in a local oci-layout",
		Args:   cobra.ExactArgs(1),
		Hidden: true, // TODO: promote to public once theres some milage
		RunE: func(_ *cobra.Command, args []string) error {
			path := args[0]

			p, err := layout.FromPath(path)

			if err != nil {
				return err
			}

			blobs, err := p.GarbageCollect()
			if err != nil {
				return err
			}

			for _, blob := range blobs {
				if err := p.RemoveBlob(blob); err != nil {
					return err
				}
				fmt.Fprintf(os.Stderr, "garbage collecting: %s\n", blob.String())
			}

			return nil
		},
	}

	return cmd
}
