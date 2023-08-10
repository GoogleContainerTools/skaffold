// Copyright 2020 Google LLC All Rights Reserved.
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
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/spf13/cobra"
)

// NewCmdOptimize creates a new cobra.Command for the optimize subcommand.
func NewCmdOptimize(options *[]crane.Option) *cobra.Command {
	var files []string

	cmd := &cobra.Command{
		Use:     "optimize SRC DST",
		Hidden:  true,
		Aliases: []string{"opt"},
		Short:   "Optimize a remote container image from src to dst",
		Args:    cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			src, dst := args[0], args[1]
			return crane.Optimize(src, dst, files, *options...)
		},
	}

	cmd.Flags().StringSliceVar(&files, "prioritize", nil,
		"The list of files to prioritize in the optimized image.")

	return cmd
}
