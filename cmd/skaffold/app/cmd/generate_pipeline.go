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
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

var (
	configFiles []string
)

func NewCmdGeneratePipeline() *cobra.Command {
	return NewCmd("generate-pipeline").
		Hidden().
		WithDescription("[ALPHA] Generate tekton pipeline from skaffold.yaml").
		WithCommonFlags().
		WithFlags(func(f *pflag.FlagSet) {
			f.StringSliceVar(&configFiles, "config-files", nil, "Select additional files whose artifacts to use when generating pipeline.")
		}).
		NoArgs(doGeneratePipeline)
}

func doGeneratePipeline(ctx context.Context, out io.Writer) error {
	return withRunner(ctx, func(r runner.Runner, config *latest.SkaffoldConfig) error {
		if err := r.GeneratePipeline(ctx, out, config, configFiles, "pipeline.yaml"); err != nil {
			return fmt.Errorf("generating : %w", err)
		}
		color.Default.Fprintln(out, "Pipeline config written to pipeline.yaml!")
		return nil
	})
}
