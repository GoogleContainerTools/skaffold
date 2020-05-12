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
	"github.com/spf13/pflag"

	debugging "github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
)

// for tests
var doDebug = runDebug

// NewCmdDebug describes the CLI command to run a pipeline in debug mode.
// Unlike `dev`, `debug` defaults `auto-build` and `auto-deploy` to `false`.
func NewCmdDebug() *cobra.Command {
	// local copies to avoid aliasing references from `NewCmdDev` https://github.com/GoogleContainerTools/skaffold/issues/4129
	var trigger string
	var autoBuild, autoDeploy, autoSync bool
	var targetImages []string
	var watchPollInterval int

	return NewCmd("debug").
		WithDescription("[beta] Run a pipeline in debug mode").
		WithLongDescription("Similar to `dev`, but configures the pipeline for debugging.").
		WithCommonFlags().
		WithFlags(func(f *pflag.FlagSet) {
			f.StringVar(&trigger, "trigger", "notify", "How is change detection triggered? (polling, notify, or manual)")
			// disable auto-build as it may trigger a hot-reload; requires more testing
			f.BoolVar(&autoBuild, "auto-build", false, "When set to false, builds wait for API request instead of running automatically (default true)")
			f.MarkHidden("auto-build")
			// disable auto-deploy as it tears down containers and kills paused processes being debugged
			f.BoolVar(&autoDeploy, "auto-deploy", false, "When set to false, deploys wait for API request instead of running automatically (default false)")
			f.MarkHidden("auto-deploy")
			// disable auto-sync as it may trigger a hot-reload; requires more testing
			f.BoolVar(&autoSync, "auto-sync", false, "When set to false, syncs wait for API request instead of running automatically (default true)")
			f.MarkHidden("auto-sync")
			f.StringSliceVarP(&targetImages, "watch-image", "w", nil, "Choose which artifacts to watch. Artifacts with image names that contain the expression will be watched only. Default is to watch sources for all artifacts")
			f.IntVarP(&watchPollInterval, "watch-poll-interval", "i", 1000, "Interval (in ms) between two checks for file changes")
		}).
		NoArgs(func(ctx context.Context, out io.Writer) error {
			opts.Trigger = trigger
			opts.AutoBuild = autoBuild
			opts.AutoDeploy = autoDeploy
			opts.AutoSync = autoSync
			opts.TargetImages = targetImages
			opts.WatchPollInterval = watchPollInterval
			return doDebug(ctx, out)
		})
}

func runDebug(ctx context.Context, out io.Writer) error {
	opts.PortForward.ForwardPods = true
	deploy.AddManifestTransform(debugging.ApplyDebuggingTransforms)

	return doDev(ctx, out)
}
