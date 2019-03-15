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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debugging"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/spf13/cobra"
)

// NewCmdDebug describes the CLI command to run a pipeline in debug mode.
func NewCmdDebug(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug",
		Short: "Runs a pipeline file in debug mode",
		Long:  "Similar to `dev`, but configures the pipeline for debugging.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return debug(out)
		},
	}
	AddRunDevFlags(cmd)
	AddDevDebugFlags(cmd)
	return cmd
}

func debug(out io.Writer) error {
	// HACK: disable watcher to prevent redeploying changed containers during debugging
	// TODO: avoid redeploys of debuggable artifacts, but still enable file-sync
	if len(opts.TargetImages) == 0 {
		opts.TargetImages = []string{"none"}
	}

	deploy.AddManifestTransform(debugging.ApplyDebuggingTransforms)

	return dev(out, opts.ExperimentalGUI)
}
