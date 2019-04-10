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

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/flags"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	buildOutputFile flags.BuildOutputFileFlag
)

// NewCmdDeploy describes the CLI command to deploy artifacts.
func NewCmdDeploy(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploys the artifacts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Command = "deploy"
			return runDeploy(out)
		},
	}
	AddRunDevFlags(cmd)
	AddRunDeployFlags(cmd)
	cmd.Flags().VarP(&buildOutputFile, "build-artifacts", "a", "`skaffold build -o {{.}}` output")
	return cmd
}

func runDeploy(out io.Writer) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	catchCtrlC(cancel)
	runner, _, err := newRunner(opts)
	if err != nil {
		return errors.Wrap(err, "creating runner")
	}
	defer runner.RPCServerShutdown()

	if err := runner.Deploy(ctx, out, buildOutputFile.BuildAritifacts()); err != nil {
		return err
	}

	runner.TailLogs(ctx, out, nil, buildOutputFile.BuildAritifacts())
	return nil
}
