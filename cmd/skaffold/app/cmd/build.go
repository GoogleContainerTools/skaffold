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
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/flags"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

var (
	quietFlag       bool
	buildFormatFlag = flags.NewTemplateFlag("{{json .}}", flags.BuildOutput{})
	buildOutputFlag string
)

// NewCmdBuild describes the CLI command to build artifacts.
func NewCmdBuild() *cobra.Command {
	return NewCmd("build").
		WithDescription("Build the artifacts").
		WithLongDescription("Build, test and tag the artifacts").
		WithExample("Build all the artifacts", "build").
		WithExample("Build artifacts with a profile activated", "build -p <profile>").
		WithExample("Build artifacts whose image name contains <db>", "build -b <db>").
		WithExample("Quietly build artifacts and output the image names as json", "build -q > build_result.json").
		WithExample("Build the artifacts and then deploy them", "build -q | skaffold deploy --build-artifacts -").
		WithExample("Print the final image names", "build -q --dry-run").
		WithCommonFlags().
		WithFlags(func(f *pflag.FlagSet) {
			f.StringSliceVarP(&opts.TargetImages, "build-image", "b", nil, "Choose which artifacts to build. Artifacts with image names that contain the expression will be built only. Default is to build sources for all artifacts")
			f.BoolVarP(&quietFlag, "quiet", "q", false, "Suppress the build output and print image built on success. See --output to format output.")
			f.VarP(buildFormatFlag, "output", "o", "Used in conjunction with --quiet flag. "+buildFormatFlag.Usage())
			f.StringVar(&buildOutputFlag, "file-output", "", "Filename to write build images to")
			f.BoolVar(&opts.DryRun, "dry-run", false, "Don't build images, just compute the tag for each artifact.")
		}).
		NoArgs(doBuild)
}

func doBuild(ctx context.Context, out io.Writer) error {
	buildOut := out
	if quietFlag {
		buildOut = ioutil.Discard
	}

	return withRunner(ctx, func(r runner.Runner, config *latest.SkaffoldConfig) error {
		bRes, err := r.BuildAndTest(ctx, buildOut, targetArtifacts(opts, config))

		if quietFlag || buildOutputFlag != "" {
			cmdOut := flags.BuildOutput{Builds: bRes}
			var buildOutput bytes.Buffer
			if err := buildFormatFlag.Template().Execute(&buildOutput, cmdOut); err != nil {
				return fmt.Errorf("executing template: %w", err)
			}

			if quietFlag {
				if _, err := out.Write(buildOutput.Bytes()); err != nil {
					return fmt.Errorf("writing build output: %w", err)
				}
			}

			if buildOutputFlag != "" {
				if err := ioutil.WriteFile(buildOutputFlag, buildOutput.Bytes(), 0644); err != nil {
					return fmt.Errorf("writing build output to file: %w", err)
				}
			}
		}

		return err
	})
}

func targetArtifacts(opts config.SkaffoldOptions, cfg *latest.SkaffoldConfig) []*latest.Artifact {
	var targetArtifacts []*latest.Artifact

	for _, artifact := range cfg.Build.Artifacts {
		if opts.IsTargetImage(artifact) {
			targetArtifacts = append(targetArtifacts, artifact)
		}
	}

	return targetArtifacts
}
