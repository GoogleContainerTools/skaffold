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
	"reflect"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/inspect"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

func AddClusterBuildEnv(ctx context.Context, out io.Writer, opts inspect.Options) error {
	formatter := inspect.OutputFormatter(out, opts.OutFormat)
	cfgs, err := inspect.GetConfigSet(ctx, config.SkaffoldOptions{
		ConfigurationFile:   opts.Filename,
		RepoCacheDir:        opts.RepoCacheDir,
		ConfigurationFilter: opts.Modules,
		SkipConfigDefaults:  true,
		MakePathsAbsolute:   util.BoolPtr(false),
	})
	if err != nil {
		return formatter.WriteErr(err)
	}
	if opts.Profile == "" {
		// empty profile flag implies that the new build env needs to be added to the default pipeline.
		// for these cases, don't add the new env definition to any configs imported as dependencies.
		cfgs = cfgs.SelectRootConfigs()
		for _, cfg := range cfgs {
			if cfg.Build.Cluster != nil && !reflect.DeepEqual(cfg.Build.Cluster, &latestV1.ClusterDetails{}) {
				return formatter.WriteErr(inspect.BuildEnvAlreadyExists(inspect.BuildEnvs.Cluster, cfg.SourceFile, ""))
			}
			cfg.Build.GoogleCloudBuild = nil
			cfg.Build.LocalBuild = nil
			cfg.Build.Cluster = constructClusterDefinition(cfg.Build.Cluster, opts.BuildEnvOptions)
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
				cfg.Profiles = append(cfg.Profiles, latestV1.Profile{Name: opts.Profile})
			}
			if cfg.Profiles[index].Build.Cluster != nil && !reflect.DeepEqual(cfg.Profiles[index].Build.Cluster, &latestV1.ClusterDetails{}) {
				return formatter.WriteErr(inspect.BuildEnvAlreadyExists(inspect.BuildEnvs.Cluster, cfg.SourceFile, opts.Profile))
			}
			cfg.Profiles[index].Build.GoogleCloudBuild = nil
			cfg.Profiles[index].Build.LocalBuild = nil
			cfg.Profiles[index].Build.Cluster = constructClusterDefinition(cfg.Build.Cluster, opts.BuildEnvOptions)

			addProfileActivationStanza(cfg, opts.Profile)
		}
	}
	return inspect.MarshalConfigSet(cfgs)
}

func constructClusterDefinition(existing *latestV1.ClusterDetails, opts inspect.BuildEnvOptions) *latestV1.ClusterDetails {
	var b latestV1.ClusterDetails
	if existing != nil {
		b = *existing
	}
	if opts.PullSecretPath != "" {
		b.PullSecretPath = opts.PullSecretPath
	}
	if opts.PullSecretName != "" {
		b.PullSecretName = opts.PullSecretName
	}
	if opts.PullSecretMountPath != "" {
		b.PullSecretMountPath = opts.PullSecretMountPath
	}
	if opts.Namespace != "" {
		b.Namespace = opts.Namespace
	}
	if opts.DockerConfigPath != "" {
		if b.DockerConfig == nil {
			b.DockerConfig = &latestV1.DockerConfig{}
		}
		b.DockerConfig.Path = opts.DockerConfigPath
	}
	if opts.DockerConfigSecretName != "" {
		if b.DockerConfig == nil {
			b.DockerConfig = &latestV1.DockerConfig{}
		}
		b.DockerConfig.SecretName = opts.DockerConfigSecretName
	}
	if opts.ServiceAccount != "" {
		b.ServiceAccountName = opts.ServiceAccount
	}
	if opts.RunAsUser >= 0 {
		b.RunAsUser = &opts.RunAsUser
	}
	if opts.RandomPullSecret {
		b.RandomPullSecret = opts.RandomPullSecret
	}
	if opts.RandomDockerConfigSecret {
		b.RandomDockerConfigSecret = opts.RandomDockerConfigSecret
	}
	if opts.Concurrency >= 0 {
		b.Concurrency = opts.Concurrency
	}
	if opts.Timeout != "" {
		b.Timeout = opts.Timeout
	}
	return &b
}
