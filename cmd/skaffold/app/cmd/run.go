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

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/tips"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
)

// NewCmdRun describes the CLI command to run a pipeline.
func NewCmdRun() *cobra.Command {
	return NewCmd("run").
		WithDescription("Run a pipeline").
		WithLongDescription("Run a pipeline: build and test artifacts, tag them, update Kubernetes manifests and deploy to a cluster.").
		WithExample("Build, test, deploy and tail the logs", "run --tail").
		WithExample("Run with a given profile", "run -p <profile>").
		WithCommonFlags().
		WithHouseKeepingMessages().
		NoArgs(doRun)
}

func doRun(ctx context.Context, out io.Writer) error {
	return withRunner(ctx, out, func(r runner.Runner, configs []util.VersionedConfig) error {
		bRes, err := r.Build(ctx, out, targetArtifacts(opts, configs))
		if err != nil {
			return fmt.Errorf("failed to build: %w", err)
		}

		if !opts.SkipTests {
			err = r.Test(ctx, out, bRes)
			if err != nil {
				return fmt.Errorf("failed to test: %w", err)
			}
		}

		err = r.DeployAndLog(ctx, out, bRes)
		if err != nil {
			return fmt.Errorf("failed to deploy: %w", err)
		}

		tips.PrintForRun(out, opts)

		return nil
	})
}
