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

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/inspect"
)

type buildEnvList struct {
	BuildEnvs []buildEnvEntry `json:"build_envs"`
}

type buildEnvEntry struct {
	Type   string `json:"type"`
	Path   string `json:"path"`
	Module string `json:"module,omitempty"`
}

func PrintBuildEnvsList(ctx context.Context, out io.Writer, opts inspect.Options) error {
	formatter := inspect.OutputFormatter(out, opts.OutFormat)
	cfgs, err := inspect.GetConfigSet(ctx, config.SkaffoldOptions{
		ConfigurationFile:   opts.Filename,
		RepoCacheDir:        opts.RepoCacheDir,
		Profiles:            opts.Profiles,
		PropagateProfiles:   opts.PropagateProfiles,
		ConfigurationFilter: opts.Modules,
	})
	if err != nil {
		formatter.WriteErr(err)
		return err
	}

	l := &buildEnvList{BuildEnvs: []buildEnvEntry{}}
	for _, c := range cfgs {
		buildEnv := inspect.GetBuildEnv(&c.Build.BuildType)
		l.BuildEnvs = append(l.BuildEnvs, buildEnvEntry{Type: string(buildEnv), Path: c.SourceFile, Module: c.Metadata.Name})
	}
	return formatter.Write(l)
}
