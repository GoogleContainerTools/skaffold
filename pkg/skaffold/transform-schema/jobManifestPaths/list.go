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

type jobManifestPathList struct {
	VerifyJobManifestPaths       []verifyJobManifestPathEntry        `json:"verifyJobManifestPaths"`
	CustomActionJobManifestPaths []customActionsJobManifestPathEntry `json:"customActionJobManifestPaths"`
}

// CustomJobManifestPath entries are handled by CustomJobManifestPath struct, there is no StructureJobManifestPath so verifyJobManifestPathEntry is required
type verifyJobManifestPathEntry struct {
	Name            string `json:"name"`
	JobManifestPath string `json:"jobManifestPath"`
}

type customActionsJobManifestPathEntry struct {
	Name            string `json:"name"`
	JobManifestPath string `json:"jobManifestPath"`
}

func PrintJobManifestPathsList(ctx context.Context, out io.Writer, opts inspect.Options) error {
	formatter := inspect.OutputFormatter(out, opts.OutFormat)
	cfgs, err := inspect.GetConfigSet(ctx, config.SkaffoldOptions{
		ConfigurationFile:   opts.Filename,
		ConfigurationFilter: opts.Modules,
		RepoCacheDir:        opts.RepoCacheDir,
		Profiles:            opts.Profiles,
		PropagateProfiles:   opts.PropagateProfiles,
	})
	if err != nil {
		formatter.WriteErr(err)
		return err
	}

	l := &jobManifestPathList{
		VerifyJobManifestPaths:       []verifyJobManifestPathEntry{},
		CustomActionJobManifestPaths: []customActionsJobManifestPathEntry{},
	}
	for _, c := range cfgs {
		for _, tc := range c.Verify {
			if tc.ExecutionMode.KubernetesClusterExecutionMode != nil && tc.ExecutionMode.KubernetesClusterExecutionMode.JobManifestPath != "" {
				l.VerifyJobManifestPaths = append(l.VerifyJobManifestPaths,
					verifyJobManifestPathEntry{
						Name:            tc.Name,
						JobManifestPath: tc.ExecutionMode.KubernetesClusterExecutionMode.JobManifestPath,
					})
			}
			// TODO(#8572) add similar logic for customAction schema fields when they are complete
		}
	}
	return formatter.Write(l)
}
