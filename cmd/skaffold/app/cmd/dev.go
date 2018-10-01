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
			return dev(out)
		},
	}
	AddRunDevFlags(cmd)
	AddDevFlags(cmd)
	cmd.Flags().BoolVar(&opts.TailDev, "tail", true, "Stream logs from deployed objects")
	return cmd
}

func dev(out io.Writer) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	catchCtrlC(cancel)

	if opts.Cleanup {
		defer func() {
			if err := delete(out); err != nil {
				logrus.Warnln("cleanup:", err)
			}
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

			if _, err := r.Dev(ctx, out, config.Build.Artifacts); err != nil {
				if errors.Cause(err) != runner.ErrorConfigurationChanged {
					return err
				}
			}
		}
	}
}
