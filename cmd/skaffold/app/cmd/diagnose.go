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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/diagnose"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
)

var (
	yamlOnly bool
)

// NewCmdDiagnose describes the CLI command to diagnose skaffold.
func NewCmdDiagnose() *cobra.Command {
	return NewCmd("diagnose").
		WithDescription("Run a diagnostic on Skaffold").
		WithExample("Search for configuration issues and print the effective configuration", "diagnose").
		WithExample("Print the effective skaffold.yaml configuration for given profile", "diagnose --yaml-only --profile PROFILE").
		WithCommonFlags().
		WithFlags(func(f *pflag.FlagSet) {
			f.BoolVar(&yamlOnly, "yaml-only", false, "Only prints the effective skaffold.yaml configuration")
		}).
		NoArgs(doDiagnose)
}

func doDiagnose(ctx context.Context, out io.Writer) error {
	runCtx, config, err := runContext(opts)
	if err != nil {
		return err
	}

	if !yamlOnly {
		fmt.Fprintln(out, "Skaffold version:", version.Get().GitCommit)
		fmt.Fprintln(out, "Configuration version:", config.APIVersion)
		fmt.Fprintln(out, "Number of artifacts:", len(config.Build.Artifacts))

		if err := diagnose.CheckArtifacts(ctx, runCtx, out); err != nil {
			return fmt.Errorf("running diagnostic on artifacts: %w", err)
		}

		color.Blue.Fprintln(out, "\nConfiguration")
	}

	buf, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshalling configuration: %w", err)
	}
	out.Write(buf)

	return nil
}
