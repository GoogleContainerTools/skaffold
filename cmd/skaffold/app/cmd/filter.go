/*
Copyright 2020 The Skaffold Authors

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
	"os"

	"github.com/spf13/cobra"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/flags"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	debugging "github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// for tests
var doFilter = runFilter

// NewCmdFilter describes the CLI command to filter and transform a set of Kubernetes manifests.
func NewCmdFilter() *cobra.Command {
	var debuggingFilters bool
	var renderFromBuildOutputFile flags.BuildOutputFileFlag

	return NewCmd("filter").
		Hidden(). // internal command
		WithDescription("[alpha] Filter and transform a set of Kubernetes manifests from stdin").
		WithLongDescription("Unlike `render`, this command does not build artifacts.").
		WithCommonFlags().
		WithFlags([]*Flag{
			{Value: &renderFromBuildOutputFile, Name: "build-artifacts", Shorthand: "a", Usage: "File containing build result from a previous 'skaffold build --file-output'"},
			{Value: &debuggingFilters, Name: "debugging", DefValue: false, Usage: `Apply debug transforms similar to "skaffold debug"`, IsEnum: true},
		}).
		NoArgs(func(ctx context.Context, out io.Writer) error {
			return doFilter(ctx, out, debuggingFilters, renderFromBuildOutputFile.BuildArtifacts())
		})
}

// runFilter loads the Kubernetes manifests from stdin and applies the debug transformations.
// Unlike `skaffold debug`, this filtering affects all images and not just the built artifacts.
func runFilter(ctx context.Context, out io.Writer, debuggingFilters bool, buildArtifacts []build.Artifact) error {
	return withRunner(ctx, func(r runner.Runner, configs []*latest.SkaffoldConfig) error {
		manifestList, err := manifest.Load(os.Stdin)
		if err != nil {
			return fmt.Errorf("loading manifests: %w", err)
		}
		if debuggingFilters {
			// TODO(bdealwis): refactor this code
			debugHelpersRegistry, err := config.GetDebugHelpersRegistry(opts.GlobalConfig)
			if err != nil {
				return fmt.Errorf("resolving debug helpers: %w", err)
			}
			insecureRegistries, err := getInsecureRegistries(opts, configs)
			if err != nil {
				return fmt.Errorf("retrieving insecure registries: %w", err)
			}

			manifestList, err = debugging.ApplyDebuggingTransforms(manifestList, buildArtifacts, manifest.Registries{
				DebugHelpersRegistry: debugHelpersRegistry,
				InsecureRegistries:   insecureRegistries,
			})
			if err != nil {
				return fmt.Errorf("transforming manifests: %w", err)
			}
		}
		out.Write([]byte(manifestList.String()))
		return nil
	})
}

func getInsecureRegistries(opts config.SkaffoldOptions, configs []*latest.SkaffoldConfig) (map[string]bool, error) {
	cfgRegistries, err := config.GetInsecureRegistries(opts.GlobalConfig)
	if err != nil {
		return nil, err
	}
	var regList []string

	regList = append(regList, opts.InsecureRegistries...)
	for _, cfg := range configs {
		regList = append(regList, cfg.Build.InsecureRegistries...)
	}
	regList = append(regList, cfgRegistries...)
	insecureRegistries := make(map[string]bool, len(regList))
	for _, r := range regList {
		insecureRegistries[r] = true
	}
	return insecureRegistries, nil
}
