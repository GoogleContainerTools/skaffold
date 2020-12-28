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
	"io/ioutil"

	"github.com/spf13/cobra"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/validation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
)

var toVersion string

func NewCmdFix() *cobra.Command {
	return NewCmd("fix").
		WithDescription("Update old configuration to a newer schema version").
		WithExample("Update \"skaffold.yaml\" in the current folder to the latest version", "fix").
		WithExample("Update \"skaffold.yaml\" in the current folder to version \"skaffold/v1\"", "fix --version skaffold/v1").
		WithCommonFlags().
		WithFlags([]*Flag{
			{Value: &overwrite, Name: "overwrite", DefValue: false, Usage: "Overwrite original config with fixed config"},
			{Value: &toVersion, Name: "version", DefValue: latest.Version, Usage: "Target schema version to upgrade to"},
		}).
		NoArgs(doFix)
}

func doFix(_ context.Context, out io.Writer) error {
	return fix(out, opts.ConfigurationFile, toVersion, overwrite)
}

func fix(out io.Writer, configFile string, toVersion string, overwrite bool) error {
	parsedCfgs, err := schema.ParseConfig(configFile)
	if err != nil {
		return err
	}
	needsUpdate := false
	for _, cfg := range parsedCfgs {
		if cfg.GetVersion() != toVersion {
			needsUpdate = true
			break
		}
	}
	if !needsUpdate {
		color.Default.Fprintln(out, "config is already version", toVersion)
		return nil
	}

	versionedCfgs, err := schema.ParseConfigAndUpgrade(configFile, toVersion)
	if err != nil {
		return err
	}

	// TODO(dgageot): We should be able run validations on any schema version
	// but that's not the case. They can only run on the latest version for now.
	if toVersion == latest.Version {
		var cfgs []*latest.SkaffoldConfig
		for _, cfg := range versionedCfgs {
			cfgs = append(cfgs, cfg.(*latest.SkaffoldConfig))
		}
		if err := validation.Process(cfgs); err != nil {
			return fmt.Errorf("validating upgraded config: %w", err)
		}
	}
	newCfg, err := yaml.MarshalWithSeparator(versionedCfgs)
	if err != nil {
		return fmt.Errorf("marshaling new config: %w", err)
	}
	if overwrite {
		if err := ioutil.WriteFile(configFile, newCfg, 0644); err != nil {
			return fmt.Errorf("writing config file: %w", err)
		}
		color.Default.Fprintf(out, "New config at version %s generated and written to %s\n", toVersion, opts.ConfigurationFile)
	} else {
		out.Write(newCfg)
	}

	return nil
}
