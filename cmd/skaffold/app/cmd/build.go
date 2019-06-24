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
	"time"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/flags"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	quietFlag       bool
	buildFormatFlag = flags.NewTemplateFlag("{{json .}}", flags.BuildOutput{})
)

// NewCmdBuild describes the CLI command to build artifacts.
func NewCmdBuild(out io.Writer) *cobra.Command {
	return NewCmd(out, "build").
		WithDescription("Builds the artifacts").
		WithCommonFlags().
		WithFlags(func(f *pflag.FlagSet) {
			f.StringSliceVarP(&opts.TargetImages, "build-image", "b", nil, "Choose which artifacts to build. Artifacts with image names that contain the expression will be built only. Default is to build sources for all artifacts")
			f.BoolVarP(&quietFlag, "quiet", "q", false, "Suppress the build output and print image built on success. See --output to format output.")
			f.VarP(buildFormatFlag, "output", "o", "Used in conjunction with --quiet flag. "+buildFormatFlag.Usage())
		}).
		NoArgs(cancelWithCtrlC(context.Background(), doBuild))
}

func doBuild(ctx context.Context, out io.Writer) error {
	buildOut := out
	if quietFlag {
		buildOut = ioutil.Discard
	}

	start := time.Now()
	defer func() {
		color.Default.Fprintln(buildOut, "Complete in", time.Since(start))
	}()

	return withRunner(ctx, func(r runner.Runner, config *latest.SkaffoldConfig) error {
		bRes, err := r.BuildAndTest(ctx, buildOut, targetArtifacts(opts, config))

		if quietFlag {
			cmdOut := flags.BuildOutput{Builds: bRes}
			if err := buildFormatFlag.Template().Execute(out, cmdOut); err != nil {
				return errors.Wrap(err, "executing template")
			}
		}

		return err
	})
}

func targetArtifacts(opts *config.SkaffoldOptions, cfg *latest.SkaffoldConfig) []*latest.Artifact {
	var targetArtifacts []*latest.Artifact

	for _, artifact := range cfg.Build.Artifacts {
		if opts.IsTargetImage(artifact) {
			targetArtifacts = append(targetArtifacts, artifact)
		}
	}

	return targetArtifacts
}
