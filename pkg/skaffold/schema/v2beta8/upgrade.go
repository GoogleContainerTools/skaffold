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

package v2beta8

import (
	next "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	pkgutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// Upgrade upgrades a configuration to the next version.
// Config changes from v2beta8 to v2beta9
// 1. No additions:
// 2. No removals
// 3. Updates:
//    - sync.auto becomes boolean
//    - localBuild.UseBuildkit bool becomes *bool
//    - localBuild.UseDockerCLI bool becomes *bool
func (c *SkaffoldConfig) Upgrade() (util.VersionedConfig, error) {
	var newConfig next.SkaffoldConfig
	pkgutil.CloneThroughJSON(c, &newConfig)
	newConfig.APIVersion = next.Version

	for i := 0; i < len(newConfig.Profiles); i++ {
		if newConfig.Profiles[i].Build.LocalBuild == nil {
			newConfig.Profiles[i].Build.LocalBuild = nil
		} else {
			if !c.Profiles[i].Build.BuildType.LocalBuild.UseBuildkit {
				newConfig.Profiles[i].Build.BuildType.LocalBuild.UseBuildkit = nil
			}
			if !c.Profiles[i].Build.BuildType.LocalBuild.UseDockerCLI {
				newConfig.Profiles[i].Build.BuildType.LocalBuild.UseDockerCLI = nil
			}
		}
	}

	err := util.UpgradePipelines(c, &newConfig, upgradeOnePipeline)
	return &newConfig, err
}

func upgradeOnePipeline(_, _ interface{}) error {
	return nil
}

func (a *Auto) MarshalJSON() ([]byte, error) {
	// The presence of an Auto{} means auto-sync is enabled.
	if a != nil {
		return []byte(`true`), nil
	}
	return nil, nil
}
