// Copyright 2019 Google LLC All Rights Reserved.
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
	"context"
	"fmt"
	"io"
	"path"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/spf13/cobra"
)

// NewCmdCatalog creates a new cobra.Command for the catalog subcommand.
func NewCmdCatalog(options *[]crane.Option, _ ...string) *cobra.Command {
	var fullRef bool
	cmd := &cobra.Command{
		Use:   "catalog REGISTRY",
		Short: "List the repos in a registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o := crane.GetOptions(*options...)

			return catalog(cmd.Context(), cmd.OutOrStdout(), args[0], fullRef, o)
		},
	}
	cmd.Flags().BoolVar(&fullRef, "full-ref", false, "(Optional) if true, print the full image reference")

	return cmd
}

func catalog(ctx context.Context, w io.Writer, src string, fullRef bool, o crane.Options) error {
	reg, err := name.NewRegistry(src, o.Name...)
	if err != nil {
		return fmt.Errorf("parsing reg %q: %w", src, err)
	}

	puller, err := remote.NewPuller(o.Remote...)
	if err != nil {
		return err
	}

	catalogger, err := puller.Catalogger(ctx, reg)
	if err != nil {
		return fmt.Errorf("reading tags for %s: %w", reg, err)
	}

	for catalogger.HasNext() {
		repos, err := catalogger.Next(ctx)
		if err != nil {
			return err
		}
		for _, repo := range repos.Repos {
			if fullRef {
				fmt.Fprintln(w, path.Join(src, repo))
			} else {
				fmt.Fprintln(w, repo)
			}
		}
	}
	return nil
}
