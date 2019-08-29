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

package v1beta13

import (
	next "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	pkgutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// Upgrade upgrades a configuration to the next version.
// Config changes from v1beta13 to v1beta14
// 1. Additions:
// single jib builder for local and gcb
// 2. Removals:
// jibMaven builder
// jibGradle builder
// 3. No updates
func (config *SkaffoldConfig) Upgrade() (util.VersionedConfig, error) {
	var newConfig next.SkaffoldConfig

	pkgutil.CloneThroughJSON(config, &newConfig)
	if err := util.UpgradePipelines(config, &newConfig, upgradeOnePipeline); err != nil {
		return nil, err
	}
	newConfig.APIVersion = next.Version

	return &newConfig, nil
}

func upgradeOnePipeline(oldPipeline, newPipeline interface{}) error {
	oldBuild := &oldPipeline.(*Pipeline).Build
	newBuild := &newPipeline.(*next.Pipeline).Build

	for i, a := range oldBuild.Artifacts {
		switch {
		case a.JibMavenArtifact != nil:
			newBuild.Artifacts[i].JibArtifact = &next.JibArtifact{
				Project: a.JibMavenArtifact.Module,
				Profile: a.JibMavenArtifact.Profile,
				Flags:   a.JibMavenArtifact.Flags,
			}
		case a.JibGradleArtifact != nil:
			newBuild.Artifacts[i].JibArtifact = &next.JibArtifact{
				Project: a.JibGradleArtifact.Project,
				Flags:   a.JibGradleArtifact.Flags,
			}
		}
	}
	return nil
}
