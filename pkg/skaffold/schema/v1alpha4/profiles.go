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

package v1alpha4

import (
	"fmt"
	"reflect"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
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
		Build:      overlayProfileField(config.Build, profile.Build).(BuildConfig),
		Deploy:     overlayProfileField(config.Deploy, profile.Deploy).(DeployConfig),
	}
}

func profilesByName(profiles []Profile) map[string]Profile {
	byName := make(map[string]Profile)
	for _, profile := range profiles {
		byName[profile.Name] = profile
	}
	return byName
}

// if we find a oneOf tag, the fields in this struct are themselves pointers to structs,
// but should be treated as values. the first non-nil one we find is what we should use.
func overlayOneOfField(config interface{}, profile interface{}) interface{} {
	v := reflect.ValueOf(profile) // the profile itself
	t := reflect.TypeOf(profile)  // the type of the profile, used for getting struct field types
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
}

func overlayStructField(config interface{}, profile interface{}) interface{} {
	// we already know the top level fields for whatever struct we have are themselves structs
	// (and not one-of values), so we need to recursively overlay them
	configValue := reflect.ValueOf(config)
	profileValue := reflect.ValueOf(profile)
	t := reflect.TypeOf(profile)
	finalConfig := reflect.New(t)

	for i := 0; i < profileValue.NumField(); i++ {
		fieldType := t.Field(i)
		overlay := overlayProfileField(configValue.Field(i).Interface(), profileValue.Field(i).Interface())
		finalConfig.Elem().FieldByName(fieldType.Name).Set(reflect.ValueOf(overlay))
	}
	return reflect.Indirect(finalConfig).Interface() // since finalConfig is a pointer, dereference it
}

func overlayProfileField(config interface{}, profile interface{}) interface{} {
	v := reflect.ValueOf(profile) // the profile itself
	t := reflect.TypeOf(profile)  // the type of the profile, used for getting struct field types
	logrus.Debugf("overlaying profile on config for field %s", t.Name())
	switch v.Kind() {
	case reflect.Struct:
		// check the first field of the struct for a oneOf yamltag.
		if util.IsOneOf(t.Field(0)) {
			return overlayOneOfField(config, profile)
		}
		return overlayStructField(config, profile)
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
