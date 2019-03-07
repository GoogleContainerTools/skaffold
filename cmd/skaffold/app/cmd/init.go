/*
Copyright 2019 The Skaffold Authors

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
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer"
	"github.com/spf13/cobra"
)

var (
	composeFile  string
	cliArtifacts []string
	skipBuild    bool
	force        bool
	analyze      bool
)

// NewCmdInit describes the CLI command to generate a Skaffold configuration.
func NewCmdInit(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Automatically generate Skaffold configuration for deploying an application",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := initializer.Config{
				ComposeFile:  composeFile,
				CliArtifacts: cliArtifacts,
				SkipBuild:    skipBuild,
				Force:        force,
				Analyze: analyze
				SkaffoldOpts: opts,
			}
			return initializer.DoInit(out, c)
		},
	}
	cmd.Flags().StringVarP(&opts.ConfigurationFile, "filename", "f", "skaffold.yaml", "Filename or URL to the pipeline file")
	cmd.Flags().BoolVar(&skipBuild, "skip-build", false, "Skip generating build artifacts in Skaffold config")
	cmd.Flags().BoolVar(&force, "force", false, "Force the generation of the Skaffold config")
	cmd.Flags().StringVar(&composeFile, "compose-file", "", "Initialize from a docker-compose file")
	cmd.Flags().StringArrayVarP(&cliArtifacts, "artifact", "a", nil, "'='-delimited dockerfile/image pair to generate build artifact\n(example: --artifact=/web/Dockerfile.web=gcr.io/web-project/image)")
	cmd.Flags().BoolVar(&analyze, "analyze", false, "Print all discoverable Dockerfiles and images in JSON format to stdout")
	return cmd
}
