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

package v1alpha3

import (
	"encoding/json"

	next "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/pkg/errors"
)

// Upgrade upgrades a configuration to the next version.
func (config *SkaffoldConfig) Upgrade() (util.VersionedConfig, error) {
	// convert Deploy (should be the same)
	var newDeploy next.DeployConfig
	if err := convert(config.Deploy, &newDeploy); err != nil {
		return nil, errors.Wrap(err, "converting deploy config")
	}

	// convert Profiles (should be the same)
	var newProfiles []next.Profile
	if config.Profiles != nil {
		if err := convert(config.Profiles, &newProfiles); err != nil {
			return nil, errors.Wrap(err, "converting new profile")
		}
	}

	// convert Build (should be the same)
	var newBuild next.BuildConfig
	if err := convert(config.Build, &newBuild); err != nil {
		return nil, errors.Wrap(err, "converting new build")
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
