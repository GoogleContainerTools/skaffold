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

package schema

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	cfg "github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

// ApplyProfiles returns configuration modified by the application
// of a list of profiles.
func ApplyProfiles(c *latest.SkaffoldPipeline, opts *cfg.SkaffoldOptions) error {
	byName := profilesByName(c.Profiles)

	profiles, err := activatedProfiles(c.Profiles, opts)
	if err != nil {
		return errors.Wrap(err, "finding auto-activated profiles")
	}

	for _, name := range profiles {
		profile, present := byName[name]
		if !present {
			return fmt.Errorf("couldn't find profile %s", name)
		}

		if err := applyProfile(c, profile); err != nil {
			return errors.Wrapf(err, "appying profile %s", name)
		}
	}

	return nil
}

func activatedProfiles(profiles []latest.Profile, opts *cfg.SkaffoldOptions) ([]string, error) {
	activated := opts.Profiles

	// Auto-activated profiles
	for _, profile := range profiles {
		for _, cond := range profile.Activation {
			command := isCommand(cond.Command, opts)

			env, err := isEnv(cond.Env)
			if err != nil {
				return nil, err
			}

			kubeContext, err := isKubeContext(cond.KubeContext)
			if err != nil {
				return nil, err
			}

			if command && env && kubeContext {
				activated = append(activated, profile.Name)
			}
		}
	}

	return activated, nil
}

func isEnv(env string) (bool, error) {
	if env == "" {
		return true, nil
	}

	keyValue := strings.SplitN(env, "=", 2)
	if len(keyValue) != 2 {
		return false, fmt.Errorf("invalid env variable format: %s, should be KEY=VALUE", env)
	}

	key := keyValue[0]
	value := keyValue[1]

	return satisfies(value, os.Getenv(key)), nil
}

func isCommand(command string, opts *cfg.SkaffoldOptions) bool {
	if command == "" {
		return true
	}

	return satisfies(command, opts.Command)
}

func isKubeContext(kubeContext string) (bool, error) {
	if kubeContext == "" {
		return true, nil
	}

	currentKubeContext, err := kubectx.CurrentContext()
	if err != nil {
		return false, errors.Wrap(err, "getting current cluster context")
	}

	return satisfies(kubeContext, currentKubeContext), nil
}

func satisfies(expected, actual string) bool {
	if strings.HasPrefix(expected, "!") {
		return actual != expected[1:]
	}
	return actual == expected
}

func applyProfile(config *latest.SkaffoldPipeline, profile latest.Profile) error {
	logrus.Infof("applying profile: %s", profile.Name)

	// this intentionally removes the Profiles field from the returned config
	*config = latest.SkaffoldPipeline{
		APIVersion: config.APIVersion,
		Kind:       config.Kind,
		Build:      overlayProfileField(config.Build, profile.Build).(latest.BuildConfig),
		Deploy:     overlayProfileField(config.Deploy, profile.Deploy).(latest.DeployConfig),
		Test:       overlayProfileField(config.Test, profile.Test).(latest.TestConfig),
	}

	if len(profile.Patches) == 0 {
		return nil
	}

	// Default patch operation to `replace`
	for i, p := range profile.Patches {
		if p.Op == "" {
			p.Op = "replace"
			profile.Patches[i] = p
		}
	}

	// Apply profile patches
	buf, err := yaml.Marshal(*config)
	if err != nil {
		return err
	}

	buf, err = profile.Patches.Apply(buf)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(buf, config)
}

func profilesByName(profiles []latest.Profile) map[string]latest.Profile {
	byName := make(map[string]latest.Profile)
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
		if isOneOf(t.Field(0)) {
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

func isOneOf(field reflect.StructField) bool {
	for _, tag := range strings.Split(field.Tag.Get("yamltags"), ",") {
		tagParts := strings.Split(tag, "=")

		if tagParts[0] == "oneOf" {
			return true
		}
	}
	return false
}
