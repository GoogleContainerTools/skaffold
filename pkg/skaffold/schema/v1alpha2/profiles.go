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
	"reflect"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ApplyProfiles returns configuration modified by the application
// of a list of profiles.
func (c *SkaffoldConfig) ApplyProfiles(profiles []string) error {
	byName := profilesByName(c.Profiles)
	for _, name := range profiles {
		profile, present := byName[name]
		if !present {
			return fmt.Errorf("couldn't find profile %s", name)
		}

		applyProfile(c, profile)
	}
	if err := c.setDefaultValues(); err != nil {
		return errors.Wrap(err, "applying default values")
	}

	return nil
}

func applyProfile(config *SkaffoldConfig, profile Profile) {
	logrus.Infof("applying profile: %s", profile.Name)

	// this intentionally removes the Profiles field from the returned config
	*config = SkaffoldConfig{
		APIVersion: config.APIVersion,
		Kind:       config.Kind,
		Build: BuildConfig{
			Artifacts: overlayProfileField(config.Build.Artifacts, profile.Build.Artifacts).([]*Artifact),
			TagPolicy: overlayProfileField(config.Build.TagPolicy, profile.Build.TagPolicy).(TagPolicy),
			BuildType: overlayProfileField(config.Build.BuildType, profile.Build.BuildType).(BuildType),
		},
		Deploy: DeployConfig{
			DeployType: overlayProfileField(config.Deploy.DeployType, profile.Deploy.DeployType).(DeployType),
		},
	}
}

func profilesByName(profiles []Profile) map[string]Profile {
	byName := make(map[string]Profile)
	for _, profile := range profiles {
		byName[profile.Name] = profile
	}
	return byName
}

func overlayProfileField(config interface{}, profile interface{}) interface{} {
	v := reflect.ValueOf(profile) // the profile itself
	t := reflect.TypeOf(profile)  // the type of the profile, used for getting struct field types
	logrus.Debugf("overlaying profile on config for field %s", t.Name())
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			fieldType := t.Field(i)              // the field type (e.g. 'LocalBuild' for BuildConfig)
			fieldValue := v.Field(i).Interface() // the value of the field itself
			if fieldValue != nil && !reflect.ValueOf(fieldValue).IsNil() {
				ret := reflect.New(t)                                                   // New(t) returns a Value representing pointer to new zero value for type t
				ret.Elem().FieldByName(fieldType.Name).Set(reflect.ValueOf(fieldValue)) // set the value
				return reflect.Indirect(ret).Interface()                                // since ret is a pointer, dereference it
			}
		}
		// if we're here, we didn't find any values set in the profile config. just return the original.
		logrus.Infof("no values found in profile for field %s, using original config values", t.Name())
		return config
	case reflect.Slice:
		// either return the values provided in the profile, or the original values if none were provided.
		if v.Len() == 0 {
			return config
		}
		return v.Interface()
	default:
		logrus.Warnf("unknown field type in profile overlay: %s. falling back to original config values", v.Kind())
		return config
	}
}
