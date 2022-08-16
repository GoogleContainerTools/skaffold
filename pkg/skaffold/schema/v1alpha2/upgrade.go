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

package v1alpha2

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	next "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha3"
	pkgutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// Upgrade upgrades a configuration to the next version.
// Config changes from v1alpha2 to v1alpha3:
// 1. No additions
// 2. No removal
// 3. Updates
//  - KanikoBuildContext instead of GCSBucket
//  - HelmRelease.valuesFilePath -> valuesFiles in yaml
func (c *SkaffoldConfig) Upgrade() (util.VersionedConfig, error) {
	// convert Deploy (should be the same)
	var newDeploy next.DeployConfig
	pkgutil.CloneThroughJSON(c.Deploy, &newDeploy)

	// if the helm deploy config was set, then convert ValueFilePath to ValuesFiles
	if oldHelmDeploy := c.Deploy.DeployType.HelmDeploy; oldHelmDeploy != nil {
		for i, oldHelmRelease := range oldHelmDeploy.Releases {
			if oldHelmRelease.ValuesFilePath != "" {
				newDeploy.DeployType.HelmDeploy.Releases[i].ValuesFiles = []string{oldHelmRelease.ValuesFilePath}
			}
		}
	}

	// convert Profiles (should be the same)
	var newProfiles []next.Profile
	if c.Profiles != nil {
		pkgutil.CloneThroughJSON(c.Profiles, &newProfiles)
	}

	// if the helm deploy config was set for a profile, then convert ValueFilePath to ValuesFiles
	for p, oldProfile := range c.Profiles {
		if oldProfileHelmDeploy := oldProfile.Deploy.DeployType.HelmDeploy; oldProfileHelmDeploy != nil {
			for i, oldProfileHelmRelease := range oldProfileHelmDeploy.Releases {
				if oldProfileHelmRelease.ValuesFilePath != "" {
					newProfiles[p].Deploy.DeployType.HelmDeploy.Releases[i].ValuesFiles = []string{oldProfileHelmRelease.ValuesFilePath}
				}
			}
		}
	}

	// convert Build (different only for kaniko)
	oldKanikoBuilder := c.Build.KanikoBuild
	c.Build.KanikoBuild = nil

	// copy over old build config to new build config
	var newBuild next.BuildConfig
	pkgutil.CloneThroughJSON(c.Build, &newBuild)

	// if the kaniko build was set, then convert it
	if oldKanikoBuilder != nil {
		newBuild.BuildType.KanikoBuild = &next.KanikoBuild{
			BuildContext: next.KanikoBuildContext{
				GCSBucket: oldKanikoBuilder.GCSBucket,
			},
			Namespace:      oldKanikoBuilder.Namespace,
			PullSecret:     oldKanikoBuilder.PullSecret,
			PullSecretName: oldKanikoBuilder.PullSecretName,
			Timeout:        oldKanikoBuilder.Timeout,
		}
	}

	return &next.SkaffoldConfig{
		APIVersion: next.Version,
		Kind:       c.Kind,
		Deploy:     newDeploy,
		Build:      newBuild,
		Profiles:   newProfiles,
	}, nil
}
