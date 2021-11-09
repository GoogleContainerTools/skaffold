/*
Copyright 2021 The Skaffold Authors

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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/inspect"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

func ModifyGcbBuildEnv(ctx context.Context, out io.Writer, opts inspect.Options) error {
	formatter := inspect.OutputFormatter(out, opts.OutFormat)
	cfgs, err := inspect.GetConfigSet(ctx, config.SkaffoldOptions{ConfigurationFile: opts.Filename, ConfigurationFilter: opts.Modules, SkipConfigDefaults: true, MakePathsAbsolute: util.BoolPtr(false)})
	if err != nil {
		formatter.WriteErr(err)
		return err
	}
	if opts.Profile == "" {
		// empty profile flag implies that only modify the default pipelines in the target `skaffold.yaml`
		cfgs = cfgs.SelectRootConfigs()
		for _, cfg := range cfgs {
			if cfg.Build.GoogleCloudBuild == nil {
				if opts.Strict {
					formatter.WriteErr(inspect.BuildEnvNotFound(inspect.BuildEnvs.GoogleCloudBuild, cfg.SourceFile, ""))
					return err
				}
				continue
			}
			cfg.Build.GoogleCloudBuild = constructGcbDefinition(cfg.Build.GoogleCloudBuild, opts.BuildEnvOptions)
			cfg.Build.LocalBuild = nil
			cfg.Build.Cluster = nil
		}
	} else {
		profileFound := false
		for _, cfg := range cfgs {
			index := -1
			for i := range cfg.Profiles {
				if cfg.Profiles[i].Name == opts.Profile {
					index = i
					break
				}
			}
			if index < 0 {
				continue
			}
			profileFound = true
			if cfg.Profiles[index].Build.GoogleCloudBuild == nil {
				if opts.Strict {
					formatter.WriteErr(inspect.BuildEnvNotFound(inspect.BuildEnvs.GoogleCloudBuild, cfg.SourceFile, opts.Profile))
					return err
				}
				continue
			}
			cfg.Profiles[index].Build.GoogleCloudBuild = constructGcbDefinition(cfg.Profiles[index].Build.GoogleCloudBuild, opts.BuildEnvOptions)
			cfg.Profiles[index].Build.LocalBuild = nil
			cfg.Profiles[index].Build.Cluster = nil
		}
		if !profileFound {
			formatter.WriteErr(inspect.ProfileNotFound(opts.Profile))
			return err
		}
	}
	return inspect.MarshalConfigSet(cfgs)
}
