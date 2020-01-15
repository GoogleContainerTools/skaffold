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

package v2alpha1

import (
	next "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	pkgutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// Upgrade upgrades a configuration to the next version.
// Config changes from v2alpha1 to v2alpha2
// 1. Additions:
// 2. Removals:
//    kaniko.buildContext
// 3. No updates
func (c *SkaffoldConfig) Upgrade() (util.VersionedConfig, error) {
	var newConfig next.SkaffoldConfig

	pkgutil.CloneThroughJSON(c, &newConfig)
	if err := util.UpgradePipelines(c, &newConfig, upgradeOnePipeline); err != nil {
		return nil, err
	}
	newConfig.APIVersion = next.Version

	return &newConfig, nil
}

// Placeholder for upgrade logic
func upgradeOnePipeline(oldPipeline, newPipeline interface{}) error {
	oldBuild := &oldPipeline.(*Pipeline).Build
	newBuild := &newPipeline.(*next.Pipeline).Build

	// move: kaniko.BuildContext.LocalDir.InitImage
	//   to: kaniko.InitImage
	for i, newArtifact := range newBuild.Artifacts {
		oldArtifact := oldBuild.Artifacts[i]

		kaniko := oldArtifact.KanikoArtifact
		if kaniko == nil {
			continue
		}

		buildContext := kaniko.BuildContext
		if buildContext == nil {
			continue
		}

		if buildContext.LocalDir != nil {
			newArtifact.KanikoArtifact.InitImage = buildContext.LocalDir.InitImage
		}
	}

	return nil
}
