/*
Copyright 2018 The Skaffold Authors

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
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// NewCmdDev describes the CLI command to run a pipeline in development mode.
func NewCmdDev(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dev",
		Short: "Runs a pipeline file in development mode",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Command = "dev"
			return dev(out)
		},
	}
	AddRunDevFlags(cmd)
	cmd.Flags().BoolVar(&opts.TailDev, "tail", true, "Stream logs from deployed objects")
	cmd.Flags().StringVar(&opts.Trigger, "trigger", "polling", "How are changes detected? (polling, manual or notify)")
	cmd.Flags().BoolVar(&opts.Cleanup, "cleanup", true, "Delete deployments after dev mode is interrupted")
	cmd.Flags().StringArrayVarP(&opts.Watch, "watch-image", "w", nil, "Choose which artifacts to watch. Artifacts with image names that contain the expression will be watched only. Default is to watch sources for all artifacts")
	cmd.Flags().IntVarP(&opts.WatchPollInterval, "watch-poll-interval", "i", 1000, "Interval (in ms) between two checks for file changes")
	cmd.Flags().BoolVar(&opts.PortForward, "port-forward", true, "Port-forward exposed container ports within pods")
	cmd.Flags().StringArrayVarP(&opts.CustomLabels, "label", "l", nil, "Add custom labels to deployed objects. Set multiple times for multiple labels")
	return cmd
}

func dev(out io.Writer) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	catchCtrlC(cancel)

	cleanup := func() {}
	if opts.Cleanup {
		defer func() {
			cleanup()
		}()
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			r, config, err := newRunner(opts)
			if err != nil {
				return errors.Wrap(err, "creating runner")
			}

			err = r.Dev(ctx, out, config.Build.Artifacts)
			if r.HasDeployed() {
				cleanup = func() {
					if err := r.Cleanup(context.Background(), out); err != nil {
						logrus.Warnln("cleanup:", err)
					}
				}
			}
			if err != nil {
				if errors.Cause(err) != runner.ErrorConfigurationChanged {
					return err
				}
			}
		}
	}
}
