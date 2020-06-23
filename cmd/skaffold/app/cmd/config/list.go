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
	"context"
	"fmt"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
)

func List(ctx context.Context, out io.Writer) error {
	var configYaml []byte
	if showAll {
		cfg, err := config.ReadConfigFile(configFile)
		if err != nil {
			return err
		}
		if cfg == nil || (cfg.Global == nil && len(cfg.ContextConfigs) == 0) { // empty config
			return nil
		}
		configYaml, err = yaml.Marshal(&cfg)
		if err != nil {
			return fmt.Errorf("marshaling config: %w", err)
		}
	} else {
		contextConfig, err := getConfigForKubectx()
		if err != nil {
			return err
		}
		if contextConfig == nil { // empty config
			return nil
		}
		configYaml, err = yaml.Marshal(&contextConfig)
		if err != nil {
			return fmt.Errorf("marshaling config: %w", err)
		}
	}

	fmt.Fprintf(out, "skaffold config: %s\n", configFile)
	out.Write(configYaml)

	return nil
}
