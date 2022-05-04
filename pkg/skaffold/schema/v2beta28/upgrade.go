/*
Copyright 2022 The Skaffold Authors

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

package v2beta28

import (
	next "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	pkgutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
)

// Upgrade upgrades a configuration to the next version.
// v2beta28 is the last config version for skaffold v1, and future version will
// follow the naming scheme of v3*
func (c *SkaffoldConfig) Upgrade() (util.VersionedConfig, error) {
	var newConfig next.SkaffoldConfig
	pkgutil.CloneThroughJSON(c, &newConfig)
	newConfig.APIVersion = next.Version

	err := util.UpgradePipelines(c, &newConfig, upgradeOnePipeline)
	return &newConfig, err
}

func upgradeOnePipeline(oldPipeline, newPipeline interface{}) error {
	old := oldPipeline.(*Pipeline)
	new := newPipeline.(*next.Pipeline)
}
