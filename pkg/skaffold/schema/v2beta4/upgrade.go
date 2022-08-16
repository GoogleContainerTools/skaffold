/*
Copyright 2020 The Skaffold Authors

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

package v2beta4

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	next "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v2beta5"
	pkgutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// Upgrade upgrades a configuration to the next version.
// Config changes from v2beta4 to v2beta5
// 1. Additions:
// 2. Removals:
// 3. Updates:
//    pullSecret renamed to pullSecretPath
//    Rename `buildpack` to `buildpacks`
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

	// rename: cluster.PullSecretPath cluster.PullSecretName
	if c := oldBuild.Cluster; c != nil {
		newBuild.Cluster.PullSecretPath = c.PullSecret
	}

	return nil
}
