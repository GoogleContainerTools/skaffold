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

	"github.com/pkg/errors"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

func SetKubeContext(out io.Writer, args []string) error {
	if err := setKubeContext(args[0]); err != nil {
		return err
	}
	fmt.Fprintf(out, "skaffold config %s now uses %s as default context\n", configMetadataName, args[0])
	return nil
}

func setKubeContext(kubeContext string) error {
	metadataName, err := resolveConfigName()
	if err != nil {
		return err
	}

	cfg, err := config.ReadConfigFile(configFile)
	if err != nil {
		return errors.Wrapf(err, "reading config")
	}

	if cfg.SkaffoldConfigs == nil {
		cfg.SkaffoldConfigs = make(map[string]string)
	}

	if kubeContext == "" {
		delete(cfg.SkaffoldConfigs, metadataName)
	} else {
		cfg.SkaffoldConfigs[metadataName] = kubeContext
	}

	return errors.Wrap(writeFullConfig(cfg), "writing config")
}

func resolveConfigName() (string, error) {
	if skaffoldYamlFile != "" && configMetadataName != "" {
		return "", fmt.Errorf("options `--skaffold-config` and `--filename` cannot be given at the same time")
	}

	if configMetadataName != "" {
		return configMetadataName, nil
	}

	if skaffoldYamlFile == "" {
		skaffoldYamlFile = "skaffold.yaml"
	}

	parsed, err := schema.ParseConfig(skaffoldYamlFile, true)
	if err != nil {
		return "", errors.Wrapf(err, "parsing pipeline config")
	}
	config := parsed.(*latest.SkaffoldConfig)

	if config.Metadata.Name == "" {
		return "", fmt.Errorf("metadata.name in %q is unset", skaffoldYamlFile)
	}

	return config.Metadata.Name, nil
}
