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
	"io"
	"io/ioutil"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
)

func NewCmdFix(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fix",
		Short: "Converts old Skaffold config to newest schema version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFix(out, opts.ConfigurationFile, overwrite)
		},
		Args: cobra.NoArgs,
	}
	cmd.Flags().StringVarP(&opts.ConfigurationFile, "filename", "f", "skaffold.yaml", "Filename or URL to the pipeline file")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "Overwrite original config with fixed config")
	return cmd
}

func runFix(out io.Writer, configFile string, overwrite bool) error {
	cfg, err := schema.ParseConfig(configFile, false)
	if err != nil {
		return err
	}

	if cfg.GetVersion() == latest.Version {
		color.Default.Fprintln(out, "config is already latest version")
		return nil
	}

	cfg, err = schema.ParseConfig(configFile, true)
	if err != nil {
		return err
	}

	newCfg, err := yaml.Marshal(cfg)
	if err != nil {
		return errors.Wrap(err, "marshaling new config")
	}

	if overwrite {
		if err := ioutil.WriteFile(configFile, newCfg, 0644); err != nil {
			return errors.Wrap(err, "writing config file")
		}
		color.Default.Fprintf(out, "New config at version %s generated and written to %s\n", cfg.GetVersion(), opts.ConfigurationFile)
	} else {
		out.Write(newCfg)
	}

	return nil
}
