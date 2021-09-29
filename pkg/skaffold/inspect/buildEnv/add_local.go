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
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

func AddLocalBuildEnv(ctx context.Context, out io.Writer, opts inspect.Options) error {
	formatter := inspect.OutputFormatter(out, opts.OutFormat)
	cfgs, err := inspect.GetConfigSet(ctx, config.SkaffoldOptions{ConfigurationFile: opts.Filename, ConfigurationFilter: opts.Modules, SkipConfigDefaults: true, MakePathsAbsolute: util.BoolPtr(false)})
	if err != nil {
		return formatter.WriteErr(err)
	}
	if opts.Profile == "" {
		// empty profile flag implies that the new build env needs to be added to the default pipeline.
		// for these cases, don't add the new env definition to any configs imported as dependencies.
		cfgs = cfgs.SelectRootConfigs()
		for _, cfg := range cfgs {
			if cfg.Build.LocalBuild != nil && (*cfg.Build.LocalBuild != latestV2.LocalBuild{}) {
				return formatter.WriteErr(inspect.BuildEnvAlreadyExists(inspect.BuildEnvs.Local, cfg.SourceFile, ""))
			}
			cfg.Build.LocalBuild = constructLocalDefinition(cfg.Build.LocalBuild, opts.BuildEnvOptions)
			cfg.Build.GoogleCloudBuild = nil
			cfg.Build.Cluster = nil
		}
	} else {
		for _, cfg := range cfgs {
			index := -1
			for i := range cfg.Profiles {
				if cfg.Profiles[i].Name == opts.Profile {
					index = i
					break
				}
			}
			if index < 0 {
				index = len(cfg.Profiles)
				cfg.Profiles = append(cfg.Profiles, latestV2.Profile{Name: opts.Profile})
			}
			if cfg.Profiles[index].Build.LocalBuild != nil && (*cfg.Profiles[index].Build.LocalBuild != latestV2.LocalBuild{}) {
				return formatter.WriteErr(inspect.BuildEnvAlreadyExists(inspect.BuildEnvs.Local, cfg.SourceFile, opts.Profile))
			}
			cfg.Profiles[index].Build.LocalBuild = constructLocalDefinition(cfg.Profiles[index].Build.LocalBuild, opts.BuildEnvOptions)
			cfg.Profiles[index].Build.GoogleCloudBuild = nil
			cfg.Profiles[index].Build.Cluster = nil
		}
	}
	return inspect.MarshalConfigSet(cfgs)
}

func constructLocalDefinition(existing *latestV2.LocalBuild, opts inspect.BuildEnvOptions) *latestV2.LocalBuild {
	var b latestV2.LocalBuild
	if existing != nil {
		b = *existing
	}
	if opts.Concurrency >= 0 {
		b.Concurrency = util.IntPtr(opts.Concurrency)
	}
	if opts.Push != nil {
		b.Push = opts.Push
	}
	if opts.TryImportMissing != nil {
		b.TryImportMissing = *opts.TryImportMissing
	}
	if opts.UseDockerCLI != nil {
		b.UseDockerCLI = *opts.UseDockerCLI
	}
	if opts.UseBuildkit != nil {
		b.UseBuildkit = *opts.UseBuildkit
	}
	return &b
}
