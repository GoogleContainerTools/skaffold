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
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// Version can be set via:
// -ldflags="-X 'github.com/google/go-containerregistry/cmd/crane/cmd.Version=$TAG'"
var Version string

func init() {
	if Version == "" {
		i, ok := debug.ReadBuildInfo()
		if !ok {
			return
		}
		Version = i.Main.Version
	}
}

// NewCmdVersion creates a new cobra.Command for the version subcommand.
func NewCmdVersion() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Long: `The version string is completely dependent on how the binary was built, so you should not depend on the version format. It may change without notice.

This could be an arbitrary string, if specified via -ldflags.
This could also be the go module version, if built with go modules (often "(devel)").`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			if Version == "" {
				fmt.Fprintln(cmd.OutOrStdout(), "could not determine build information")
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), Version)
			}
		},
	}
}
