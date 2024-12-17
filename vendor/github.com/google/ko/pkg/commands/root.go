// Copyright 2021 ko Build Authors All Rights Reserved.
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

package commands

import (
	"os"

	cranecmd "github.com/google/go-containerregistry/cmd/crane/cmd"
	"github.com/google/go-containerregistry/pkg/logs"
	"github.com/spf13/cobra"
	"go.uber.org/automaxprocs/maxprocs"
)

var Root = New()

func New() *cobra.Command {
	var verbose bool
	root := &cobra.Command{
		Use:               "ko",
		Short:             "Rapidly iterate with Go, Containers, and Kubernetes.",
		SilenceUsage:      true, // Don't show usage on errors
		DisableAutoGenTag: true,
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			if verbose {
				logs.Warn.SetOutput(os.Stderr)
				logs.Debug.SetOutput(os.Stderr)
			}
			logs.Progress.SetOutput(os.Stderr)

			maxprocs.Set(maxprocs.Logger(logs.Debug.Printf))
		},
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Help()
		},
	}
	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable debug logs")

	AddKubeCommands(root)

	// Also add the auth group from crane to facilitate logging into a
	// registry.
	authCmd := cranecmd.NewCmdAuth(nil, "ko", "auth")
	// That was a mistake, but just set it to Hidden so we don't break people.
	authCmd.Hidden = true
	root.AddCommand(authCmd)

	// Just add a `ko login` command:
	root.AddCommand(cranecmd.NewCmdAuthLogin("ko"))
	return root
}
