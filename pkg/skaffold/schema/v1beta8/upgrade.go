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
	"github.com/pkg/errors"

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
func (config *SkaffoldConfig) Upgrade() (util.VersionedConfig, error) {
	var newConfig next.SkaffoldConfig

	if err := pkgutil.CloneThroughJSON(config, &newConfig); err != nil {
		return nil, err
	}
	newConfig.APIVersion = next.Version

	if err := util.UpgradePipelines(config, &newConfig, upgradeOnePipeline); err != nil {
		return nil, err
	}

	return &newConfig, nil
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
			if err := pkgutil.CloneThroughYAML(a.BuilderPlugin.Properties, &ba); err != nil {
				return errors.Wrap(err, "converting bazel artifact")
			}
			newBuild.Artifacts[i].BazelArtifact = ba
		}
		if a.BuilderPlugin.Name == "docker" {
			var da *next.DockerArtifact
			if err := pkgutil.CloneThroughYAML(a.BuilderPlugin.Properties, &da); err != nil {
				return errors.Wrap(err, "converting docker artifact")
			}
			newBuild.Artifacts[i].DockerArtifact = da
		}
	}

	if c := oldBuild.ExecutionEnvironment; c != nil {
		if c.Name == "googleCloudBuild" {
			var gcb *next.GoogleCloudBuild
			if err := pkgutil.CloneThroughYAML(c.Properties, &gcb); err != nil {
				return errors.Wrap(err, "converting gcb artifact")
			}
			newBuild.GoogleCloudBuild = gcb
		}
		if c.Name == "local" {
			var local *next.LocalBuild
			if err := pkgutil.CloneThroughYAML(c.Properties, &local); err != nil {
				return errors.Wrap(err, "converting local artifact")
			}
			newBuild.LocalBuild = local
		}
	}
	return nil
}
