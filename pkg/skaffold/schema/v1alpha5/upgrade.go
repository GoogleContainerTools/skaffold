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

package v1alpha5

import (
	"fmt"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	next "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta1"
	pkgutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// Upgrade upgrades a configuration to the next version.
// Config changes from v1alpha5 to v1beta1:
// 1. Additions:
//   - KanikoCache struct, KanikoBuild.Cache
//   - BazelArtifact.BuildArgs
// 2. Removals:
//   - AzureContainerBuilder
// 3. No updates
func (c *SkaffoldConfig) Upgrade() (util.VersionedConfig, error) {
	if c.Build.AzureContainerBuild != nil {
		return nil, fmt.Errorf("can't upgrade to %s, build.acr is not supported anymore, please remove it manually", next.Version)
	}

	for _, profile := range c.Profiles {
		if profile.Build.AzureContainerBuild != nil {
			return nil, fmt.Errorf("can't upgrade to %s, profiles.build.acr is not supported anymore, please remove it from the %s profile manually", next.Version, profile.Name)
		}
	}

	var newConfig next.SkaffoldConfig

	pkgutil.CloneThroughJSON(c, &newConfig)
	newConfig.APIVersion = next.Version

	return &newConfig, nil
}
