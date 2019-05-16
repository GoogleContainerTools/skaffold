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

package config

import (
	"fmt"
	"io"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd/commands"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	yaml "gopkg.in/yaml.v2"
)

func NewCmdList(out io.Writer) *cobra.Command {
	return commands.
		New(out).
		WithDescription("list", "List all values set in the global Skaffold config").
		WithFlags(func(f *pflag.FlagSet) {
			f.BoolVarP(&showAll, "all", "a", false, "Show values for all kubecontexts")
			AddConfigFlags(f)
		}).
		NoArgs(doList)
}

func doList(out io.Writer) error {
	var configYaml []byte
	if showAll {
		cfg, err := readConfig()
		if err != nil {
			return err
		}
		if cfg == nil || (cfg.Global == nil && len(cfg.ContextConfigs) == 0) { // empty config
			return nil
		}
		configYaml, err = yaml.Marshal(&cfg)
		if err != nil {
			return errors.Wrap(err, "marshaling config")
		}
	} else {
		config, err := GetConfigForKubectx()
		if err != nil {
			return err
		}
		if config == nil { // empty config
			return nil
		}
		configYaml, err = yaml.Marshal(&config)
		if err != nil {
			return errors.Wrap(err, "marshaling config")
		}
	}
	out.Write([]byte(fmt.Sprintf("skaffold config: %s\n", configFile)))
	out.Write(configYaml)
	return nil
}
