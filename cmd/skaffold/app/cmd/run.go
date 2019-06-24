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

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/tips"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// NewCmdRun describes the CLI command to run a pipeline.
func NewCmdRun(out io.Writer) *cobra.Command {
	return NewCmd(out, "run").
		WithDescription("Runs a pipeline file").
		WithCommonFlags().
		WithFlags(func(f *pflag.FlagSet) {
			f.StringVarP(&opts.CustomTag, "tag", "t", "", "The optional custom tag to use for images which overrides the current Tagger configuration")
		}).
		NoArgs(cancelWithCtrlC(context.Background(), doRun))
}

func doRun(ctx context.Context, out io.Writer) error {
	return withRunner(ctx, func(r runner.Runner, config *latest.SkaffoldConfig) error {
		bRes, err := r.BuildAndTest(ctx, out, config.Build.Artifacts)
		if err != nil {
			return errors.Wrap(err, "failed to build")
		}

		err = r.DeployAndLog(ctx, out, bRes)
		if err == nil {
			tips.PrintForRun(out, opts)
		}

		return err
	})
}
