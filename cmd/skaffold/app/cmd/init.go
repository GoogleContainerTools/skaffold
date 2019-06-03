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
	"github.com/spf13/pflag"
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
	return NewCmd(out, "init").
		WithDescription("Automatically generate Skaffold configuration for deploying an application").
		WithFlags(func(f *pflag.FlagSet) {
			f.StringVarP(&opts.ConfigurationFile, "filename", "f", "skaffold.yaml", "Filename or URL to the pipeline file")
			f.BoolVar(&skipBuild, "skip-build", false, "Skip generating build artifacts in Skaffold config")
			f.BoolVar(&force, "force", false, "Force the generation of the Skaffold config")
			f.StringVar(&composeFile, "compose-file", "", "Initialize from a docker-compose file")
			f.StringSliceVarP(&cliArtifacts, "artifact", "a", nil, "'='-delimited dockerfile/image pair to generate build artifact\n(example: --artifact=/web/Dockerfile.web=gcr.io/web-project/image)")
			f.BoolVar(&analyze, "analyze", false, "Print all discoverable Dockerfiles and images in JSON format to stdout")
		}).
		NoArgs(doInit)
}

func doInit(out io.Writer) error {
	return initializer.DoInit(out, initializer.Config{
		ComposeFile:  composeFile,
		CliArtifacts: cliArtifacts,
		SkipBuild:    skipBuild,
		Force:        force,
		Analyze:      analyze,
		Opts:         opts,
	})
}
