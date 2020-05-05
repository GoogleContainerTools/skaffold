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

package v1beta8

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	next "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta9"
	pkgutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// Upgrade upgrades a configuration to the next version.
// Config changes from v1beta8 to v1beta9
// 1. Additions:
//    gitTagger/variant
// 2. Removed all schemas associated with builder plugins
// 3. No updates
func (c *SkaffoldConfig) Upgrade() (util.VersionedConfig, error) {
	var newConfig next.SkaffoldConfig
	pkgutil.CloneThroughJSON(c, &newConfig)
	newConfig.APIVersion = next.Version

	err := util.UpgradePipelines(c, &newConfig, upgradeOnePipeline)
	return &newConfig, err
}

func upgradeOnePipeline(oldPipeline, newPipeline interface{}) error {
	oldBuild := &oldPipeline.(*Pipeline).Build
	newBuild := &newPipeline.(*next.Pipeline).Build

	for i, a := range oldBuild.Artifacts {
		if a.BuilderPlugin == nil {
			continue
		}
		if a.BuilderPlugin.Name == "bazel" {
			var ba *next.BazelArtifact
			pkgutil.CloneThroughYAML(a.BuilderPlugin.Properties, &ba)

			newBuild.Artifacts[i].BazelArtifact = ba
		}
		if a.BuilderPlugin.Name == "docker" {
			var da *next.DockerArtifact
			pkgutil.CloneThroughYAML(a.BuilderPlugin.Properties, &da)

			newBuild.Artifacts[i].DockerArtifact = da
		}
	}

	if c := oldBuild.ExecutionEnvironment; c != nil {
		if c.Name == "googleCloudBuild" {
			var gcb *next.GoogleCloudBuild
			pkgutil.CloneThroughYAML(c.Properties, &gcb)

			newBuild.GoogleCloudBuild = gcb
		}
		if c.Name == "local" {
			var local *next.LocalBuild
			pkgutil.CloneThroughYAML(c.Properties, &local)

			newBuild.LocalBuild = local
		}
	}
	return nil
}
