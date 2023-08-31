/*
Copyright 2023 The Skaffold Authors

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

package inspect

import (
	"context"
	"io"
	"os"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/inspect"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/yaml"
)

func AddConfigDependencies(ctx context.Context, out io.Writer, opts inspect.Options, inputFile string) error {
	formatter := inspect.OutputFormatter(out, opts.OutFormat)

	yamlFile, err := os.Open(inputFile)
	if err != nil {
		formatter.WriteErr(err)
		return err
	}
	defer yamlFile.Close()
	fileBytes, err := io.ReadAll(yamlFile)
	if err != nil {
		formatter.WriteErr(err)
		return err
	}
	var cds []latest.ConfigDependency
	if err = yaml.UnmarshalStrict(fileBytes, &cds); err != nil {
		formatter.WriteErr(err)
		return err
	}

	cfgs, err := inspect.GetConfigSet(ctx, config.SkaffoldOptions{
		ConfigurationFile:   opts.Filename,
		RepoCacheDir:        opts.RepoCacheDir,
		ConfigurationFilter: opts.Modules,
		SkipConfigDefaults:  true,
		MakePathsAbsolute:   util.Ptr(false),
	})
	if err != nil {
		formatter.WriteErr(err)
		return err
	}
	for _, cfg := range cfgs {
		cfg.Dependencies = append(cfg.Dependencies, cds...)
	}
	return inspect.MarshalConfigSet(cfgs)
}
