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
	"context"
	"io"

	"github.com/spf13/cobra"

	debugging "github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
)

// for tests
var doDebug = runDebug

// NewCmdDebug describes the CLI command to run a pipeline in debug mode.
// Unlike `dev`, `debug` defaults `auto-build` and `auto-deploy` to `false`.
func NewCmdDebug() *cobra.Command {
	return NewCmd("debug").
		WithDescription("[beta] Run a pipeline in debug mode").
		WithLongDescription("Similar to `dev`, but configures the pipeline for debugging.").
		WithCommonFlags().
		NoArgs(func(ctx context.Context, out io.Writer) error {
			return doDebug(ctx, out)
		})
}

func runDebug(ctx context.Context, out io.Writer) error {
	opts.PortForward.ForwardPods = true
	deploy.AddManifestTransform(debugging.ApplyDebuggingTransforms)

	return doDev(ctx, out)
}
