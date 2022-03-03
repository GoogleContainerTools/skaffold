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

package v1

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"

	pipeline "github.com/GoogleContainerTools/skaffold/pkg/skaffold/generate_pipeline"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/defaults"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
)

func (r *SkaffoldRunner) GeneratePipeline(ctx context.Context, out io.Writer, configs []util.VersionedConfig, configPaths []string, fileOut string) error {
	// Keep track of files, configs, and profiles. This will be used to know which files to write
	// profiles to and what flags to add to task commands
	var baseConfig []*pipeline.ConfigFile
	for _, config := range configs {
		cfgFile := &pipeline.ConfigFile{
			Path:    r.runCtx.ConfigurationFile(),
			Config:  config.(*latestV1.SkaffoldConfig),
			Profile: nil,
		}
		baseConfig = append(baseConfig, cfgFile)
	}

	configFiles, err := setupConfigFiles(configPaths)
	if err != nil {
		return fmt.Errorf("setting up ConfigFiles: %w", err)
	}
	configFiles = append(baseConfig, configFiles...)

	// Will run the profile setup multiple times and require user input for each specified config
	output.Default.Fprintln(out, "Running profile setup...")
	for _, configFile := range configFiles {
		if err := pipeline.CreateSkaffoldProfile(out, r.runCtx.GetKubeNamespace(), configFile); err != nil {
			return fmt.Errorf("setting up profile: %w", err)
		}
	}

	output.Default.Fprintln(out, "Generating Pipeline...")
	pipelineYaml, err := pipeline.Yaml(out, r.runCtx.GetKubeNamespace(), configFiles)
	if err != nil {
		return fmt.Errorf("generating pipeline yaml contents: %w", err)
	}

	// write all yaml pieces to output
	return ioutil.WriteFile(fileOut, pipelineYaml.Bytes(), 0755)
}

func setupConfigFiles(configPaths []string) ([]*pipeline.ConfigFile, error) {
	if configPaths == nil {
		return []*pipeline.ConfigFile{}, nil
	}

	// Read all given config files to read contents into SkaffoldConfig
	var configFiles []*pipeline.ConfigFile
	for _, path := range configPaths {
		parsedCfgs, err := schema.ParseConfigAndUpgrade(path)
		if err != nil {
			return nil, fmt.Errorf("parsing config %q: %w", path, err)
		}
		for _, parsedCfg := range parsedCfgs {
			config := parsedCfg.(*latestV1.SkaffoldConfig)

			if err := defaults.Set(config); err != nil {
				return nil, fmt.Errorf("setting default values for extra configs: %w", err)
			}
			defaults.SetDefaultDeployer(config)
			configFile := &pipeline.ConfigFile{
				Path:   path,
				Config: config,
			}
			configFiles = append(configFiles, configFile)
		}
	}

	return configFiles, nil
}
