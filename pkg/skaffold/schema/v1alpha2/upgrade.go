/*
Copyright 2018 The Skaffold Authors

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
	"encoding/json"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	next "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha3"
	"github.com/pkg/errors"
)

// Upgrade upgrades a configuration to the next version.
func (config *SkaffoldConfig) Upgrade() (util.VersionedConfig, error) {
	// convert Deploy (should be the same)
	var newDeploy next.DeployConfig
	if err := convert(config.Deploy, &newDeploy); err != nil {
		return nil, errors.Wrap(err, "converting deploy config")
	}
	// if the helm deploy config was set, then convert ValueFilePath to ValuesFiles
	if oldHelmDeploy := config.Deploy.DeployType.HelmDeploy; oldHelmDeploy != nil {
		for i, oldHelmRelease := range oldHelmDeploy.Releases {
			if oldHelmRelease.ValuesFilePath != "" {
				newDeploy.DeployType.HelmDeploy.Releases[i].ValuesFiles = []string{oldHelmRelease.ValuesFilePath}
			}
		}
	}

	// convert Profiles (should be the same)
	var newProfiles []next.Profile
	if config.Profiles != nil {
		if err := convert(config.Profiles, &newProfiles); err != nil {
			return nil, errors.Wrap(err, "converting new profile")
		}
	}

	// if the helm deploy config was set for a profile, then convert ValueFilePath to ValuesFiles
	for p, oldProfile := range config.Profiles {
		if oldProfileHelmDeploy := oldProfile.Deploy.DeployType.HelmDeploy; oldProfileHelmDeploy != nil {
			for i, oldProfileHelmRelease := range oldProfileHelmDeploy.Releases {
				if oldProfileHelmRelease.ValuesFilePath != "" {
					newProfiles[p].Deploy.DeployType.HelmDeploy.Releases[i].ValuesFiles = []string{oldProfileHelmRelease.ValuesFilePath}
				}
			}
		}
	}

	// convert Build (different only for kaniko)
	oldKanikoBuilder := config.Build.KanikoBuild
	config.Build.KanikoBuild = nil

	// copy over old build config to new build config
	var newBuild next.BuildConfig
	if err := convert(config.Build, &newBuild); err != nil {
		return nil, errors.Wrap(err, "converting new build")
	}
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
		Kind:       config.Kind,
		Deploy:     newDeploy,
		Build:      newBuild,
		Profiles:   newProfiles,
	}, nil
}

func convert(old interface{}, new interface{}) error {
	o, err := json.Marshal(old)
	if err != nil {
		return errors.Wrap(err, "marshalling old")
	}
	if err := json.Unmarshal(o, &new); err != nil {
		return errors.Wrap(err, "unmarshalling new")
	}
	return nil
}
