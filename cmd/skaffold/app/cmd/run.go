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

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd/commands"
	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/tips"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// NewCmdRun describes the CLI command to run a pipeline.
func NewCmdRun(out io.Writer) *cobra.Command {
	return commands.
		New(out).
		WithDescription("run", "Runs a pipeline file").
		WithFlags(func(f *pflag.FlagSet) {
			f.StringVarP(&opts.CustomTag, "tag", "t", "", "The optional custom tag to use for images which overrides the current Tagger configuration")
			AddRunDevFlags(f)
			AddRunDeployFlags(f)
		}).
		NoArgs(doRun)
}

func doRun(out io.Writer) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	catchCtrlC(cancel)

	runner, config, err := newRunner(opts)
	if err != nil {
		return errors.Wrap(err, "creating runner")
	}
	defer runner.RPCServerShutdown()

	err = runner.Run(ctx, out, config.Build.Artifacts)
	if err == nil {
		tips.PrintForRun(out, opts)
	}

	return err
}
