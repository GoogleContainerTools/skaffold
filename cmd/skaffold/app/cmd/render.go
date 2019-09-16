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
	"io/ioutil"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	showBuild        bool
	renderOutputPath string
)

// NewCmdRender describes the CLI command to build artifacts render kubernetes manifests.
func NewCmdRender() *cobra.Command {
	return NewCmd("render").
		WithDescription("Perform all image builds, and output rendered kubernetes manifests").
		WithCommonFlags().
		WithFlags(func(f *pflag.FlagSet) {
			f.BoolVar(&showBuild, "loud", false, "Show the build logs and output")
			f.StringVar(&renderOutputPath, "output", "", "file to write rendered manifests to")
		}).
		NoArgs(cancelWithCtrlC(context.Background(), doRender))
}

func doRender(ctx context.Context, out io.Writer) error {
	buildOut := ioutil.Discard
	if showBuild {
		buildOut = out
	}

	return withRunner(ctx, func(r runner.Runner, config *latest.SkaffoldConfig) error {
		bRes, err := r.BuildAndTest(ctx, buildOut, targetArtifacts(opts, config))

		if err != nil {
			return errors.Wrap(err, "executing build")
		}

		if err := r.Render(ctx, out, bRes, renderOutputPath); err != nil {
			return errors.Wrap(err, "rendering manifests")
		}
		return nil
	})
}
