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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// NewCmdDev describes the CLI command to run a pipeline in development mode.
func NewCmdDev(out io.Writer) *cobra.Command {
	cmdUse := "dev"
	return NewCmd(out, cmdUse).
		WithDescription("Runs a pipeline file in development mode").
		WithCommonFlags().
		WithFlags(func(f *pflag.FlagSet) {
			f.StringVar(&opts.Trigger, "trigger", "polling", "How are changes detected? (polling, manual or notify)")
			f.StringSliceVarP(&opts.TargetImages, "watch-image", "w", nil, "Choose which artifacts to watch. Artifacts with image names that contain the expression will be watched only. Default is to watch sources for all artifacts")
			f.IntVarP(&opts.WatchPollInterval, "watch-poll-interval", "i", 1000, "Interval (in ms) between two checks for file changes")
		}).
		NoArgs(cancelWithCtrlC(context.Background(), doDev))
}

func doDev(ctx context.Context, out io.Writer) error {
	opts.EnableRPC = true

	cleanup := func() {}
	if opts.Cleanup {
		defer func() {
			cleanup()
		}()
	}

	prune := func() {}
	if opts.Prune() {
		defer func() {
			prune()
		}()
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			err := withRunner(ctx, func(r runner.Runner, config *latest.SkaffoldConfig) error {
				err := r.Dev(ctx, out, config.Build.Artifacts)

				if r.HasDeployed() {
					cleanup = func() {
						if err := r.Cleanup(context.Background(), out); err != nil {
							logrus.Warnln("deployer cleanup:", err)
						}
					}
				}

				if r.HasBuilt() {
					prune = func() {
						if err := r.Prune(context.Background(), out); err != nil {
							logrus.Warnln("builder cleanup:", err)
						}
					}
				}

				return err
			})
			if err != nil {
				if errors.Cause(err) != runner.ErrorConfigurationChanged {
					return err
				}
				// Otherwise, the skaffold config has changed.
				// just recreate a new runner and restart a dev loop
			}
		}
	}
}
