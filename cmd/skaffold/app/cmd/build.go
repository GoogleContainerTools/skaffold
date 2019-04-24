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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	quietFlag bool
	// buildFormatFlag = flags.NewTemplateFlag("{{json .}}", flags.BuildOutput{})
	template        = "{{ range .Builds}}{{if not .}}{{else}}{{.Result}}{{end}}{{end}}"
	buildFormatFlag = flags.NewTemplateFlag(template, BuildOutput{})
)

// For testing
var (
	createRunnerAndBuildFunc = createRunnerAndBuild
)

// NewCmdBuild describes the CLI command to build artifacts.
func NewCmdBuild(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Builds the artifacts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBuild(out)
		},
	}
	AddRunDevFlags(cmd)
	cmd.Flags().StringSliceVarP(&opts.TargetImages, "build-image", "b", nil, "Choose which artifacts to build. Artifacts with image names that contain the expression will be built only. Default is to build sources for all artifacts")
	cmd.Flags().BoolVarP(&quietFlag, "quiet", "q", false, "Suppress the build output and print image built on success. See --output to format output.")
	cmd.Flags().VarP(buildFormatFlag, "output", "o", "Used in conjuction with --quiet flag. "+buildFormatFlag.Usage())
	return cmd
}

// BuildOutput is the output of `skaffold build`.
type BuildOutput struct {
	Builds []build.Result
}

func runBuild(out io.Writer) error {
	start := time.Now()
	defer func() {
		if !quietFlag {
			color.Default.Fprintln(out, "Complete in", time.Since(start))
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	catchCtrlC(cancel)

	buildOut := out
	if quietFlag {
		buildOut = ioutil.Discard
	}

	bRes, err := createRunnerAndBuildFunc(ctx, buildOut)

	if err != nil {
		return err
	}

	if quietFlag {
		cmdOut := flags.BuildOutput{Builds: bRes}
		if err := buildFormatFlag.Template().Execute(out, cmdOut); err != nil {
			return errors.Wrap(err, "executing template")
		}
	}

	return nil
}

func createRunnerAndBuild(ctx context.Context, buildOut io.Writer) ([]build.Result, error) {
	runner, config, err := newRunner(opts)
	if err != nil {
		return nil, errors.Wrap(err, "creating runner")
	}
	defer runner.RPCServerShutdown()
	var targetArtifacts []*latest.Artifact
	for _, artifact := range config.Build.Artifacts {
		if runner.IsTargetImage(artifact) {
			targetArtifacts = append(targetArtifacts, artifact)
		}
	}
	return runner.BuildAndTest(ctx, buildOut, targetArtifacts)
	// if err != nil {
	// return nil, err
	// }
	// results := make([]build.Artifact, len(bRes))
	// for i, r := range bRes {
	// 	results[i] = *r.Result
	// }
	// return results, nil
}
