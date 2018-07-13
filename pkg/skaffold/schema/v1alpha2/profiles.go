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
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

// ApplyProfiles returns configuration modified by the application
// of a list of profiles.
func (c *SkaffoldConfig) ApplyProfiles(profiles []string) error {
	var err error

	byName := profilesByName(c.Profiles)
	for _, name := range profiles {
		profile, present := byName[name]
		if !present {
			return fmt.Errorf("couldn't find profile %s", name)
		}

		err = applyProfile(c, profile)
		if err != nil {
			return errors.Wrapf(err, "applying profile %s", name)
		}
	}

	c.Profiles = nil
	if err := c.setDefaultValues(); err != nil {
		return errors.Wrap(err, "applying default values")
	}

	return nil
}

func applyProfile(config *SkaffoldConfig, profile Profile) error {
	logrus.Infof("Applying profile: %s", profile.Name)

	buf, err := yaml.Marshal(profile)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(buf, config)
}

func profilesByName(profiles []Profile) map[string]Profile {
	byName := make(map[string]Profile)
	for _, profile := range profiles {
		byName[profile.Name] = profile
	}
	return byName
}
