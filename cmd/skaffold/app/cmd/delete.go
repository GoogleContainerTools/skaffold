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

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd/helper"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// NewCmdDelete describes the CLI command to delete deployed resources.
func NewCmdDelete(out io.Writer) *cobra.Command {
	cmd := helper.NoArgCommand(out, "delete", "Delete the deployed resources", delete)
	AddRunCommonFlags(cmd)
	return cmd
}

func delete(out io.Writer) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	catchCtrlC(cancel)

	runner, _, err := newRunner(opts)
	if err != nil {
		return errors.Wrap(err, "creating runner")
	}
	defer runner.RPCServerShutdown()

	return runner.Deployer.Cleanup(ctx, out)
}
