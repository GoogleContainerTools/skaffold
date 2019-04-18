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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/plugins/builders"
	"github.com/spf13/cobra"
)

// NewServeBuilderPlugins describes the CLI command to build artifacts.
func NewServeBuilderPlugins(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve-builder-plugins",
		Short: "Serves builder plugins",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Command = "serve-build-plugins"
			corePlugin, err := builders.GetCorePluginFromEnv()
			if err != nil {
				return err
			}
			if corePlugin != "" {
				return builders.Execute(corePlugin)
			}
			return nil
		},
	}
	return cmd
}
