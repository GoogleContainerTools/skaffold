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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/parser"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

func AddGcbBuildEnv(ctx context.Context, out io.Writer, opts inspect.Options) error {
	formatter := inspect.OutputFormatter(out, opts.OutFormat)
	cfgs, err := inspect.GetConfigSet(ctx, config.SkaffoldOptions{
		ConfigurationFile:   opts.Filename,
		RepoCacheDir:        opts.RepoCacheDir,
		ConfigurationFilter: opts.Modules,
		SkipConfigDefaults:  true,
		MakePathsAbsolute:   util.BoolPtr(false)})
	if err != nil {
		return formatter.WriteErr(err)
	}
	if opts.Profile == "" {
		// empty profile flag implies that the new build env needs to be added to the default pipeline.
		// for these cases, don't add the new env definition to any configs imported as dependencies.
		cfgs = cfgs.SelectRootConfigs()
		for _, cfg := range cfgs {
			if cfg.Build.GoogleCloudBuild != nil && (*cfg.Build.GoogleCloudBuild != latestV2.GoogleCloudBuild{}) {
				return formatter.WriteErr(inspect.BuildEnvAlreadyExists(inspect.BuildEnvs.GoogleCloudBuild, cfg.SourceFile, ""))
			}
			cfg.Build.GoogleCloudBuild = constructGcbDefinition(cfg.Build.GoogleCloudBuild, opts.BuildEnvOptions)
			cfg.Build.LocalBuild = nil
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
			if cfg.Profiles[index].Build.GoogleCloudBuild != nil && (*cfg.Profiles[index].Build.GoogleCloudBuild != latestV2.GoogleCloudBuild{}) {
				return formatter.WriteErr(inspect.BuildEnvAlreadyExists(inspect.BuildEnvs.GoogleCloudBuild, cfg.SourceFile, opts.Profile))
			}
			cfg.Profiles[index].Build.GoogleCloudBuild = constructGcbDefinition(cfg.Profiles[index].Build.GoogleCloudBuild, opts.BuildEnvOptions)
			cfg.Profiles[index].Build.LocalBuild = nil
			cfg.Profiles[index].Build.Cluster = nil

			addProfileActivationStanza(cfg, opts.Profile)
		}
	}
	return inspect.MarshalConfigSet(cfgs)
}

func constructGcbDefinition(existing *latestV2.GoogleCloudBuild, opts inspect.BuildEnvOptions) *latestV2.GoogleCloudBuild {
	var b latestV2.GoogleCloudBuild
	if existing != nil {
		b = *existing
	}
	if opts.Concurrency >= 0 {
		b.Concurrency = opts.Concurrency
	}
	if opts.DiskSizeGb > 0 {
		b.DiskSizeGb = opts.DiskSizeGb
	}
	if opts.MachineType != "" {
		b.MachineType = opts.MachineType
	}
	if opts.ProjectID != "" {
		b.ProjectID = opts.ProjectID
	}
	if opts.Timeout != "" {
		b.Timeout = opts.Timeout
	}
	if opts.Logging != "" {
		b.Logging = opts.Logging
	}
	if opts.LogStreamingOption != "" {
		b.LogStreamingOption = opts.LogStreamingOption
	}
	if opts.WorkerPool != "" {
		b.WorkerPool = opts.WorkerPool
	}
	return &b
}

func addProfileActivationStanza(cfg *parser.SkaffoldConfigEntry, profileName string) {
	for i := range cfg.Dependencies {
		if cfg.Dependencies[i].GitRepo != nil {
			// setup profile activation stanza only for local config dependencies
			continue
		}
		for j := range cfg.Dependencies[i].ActiveProfiles {
			if cfg.Dependencies[i].ActiveProfiles[j].Name == profileName {
				if !util.StrSliceContains(cfg.Dependencies[i].ActiveProfiles[j].ActivatedBy, profileName) {
					cfg.Dependencies[i].ActiveProfiles[j].ActivatedBy = append(cfg.Dependencies[i].ActiveProfiles[j].ActivatedBy, profileName)
				}
				return
			}
		}
		cfg.Dependencies[i].ActiveProfiles = append(cfg.Dependencies[i].ActiveProfiles, latestV2.ProfileDependency{Name: profileName, ActivatedBy: []string{profileName}})
	}
}
