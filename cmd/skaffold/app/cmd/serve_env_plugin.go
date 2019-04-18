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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/plugins/environments"
	"github.com/spf13/cobra"
)

// NewServeBuilderPlugins describes the CLI command to build artifacts.
func NewServeEnvPlugins(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve-env-plugin",
		Short: "Serve env plugins",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Command = "serve-env-plugin"
			// start google cloud plugin by default
			return environments.Execute("googlecloudbuild")
		},
	}
	return cmd
}
