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

package schema

import (
	"fmt"
	"os"
	"reflect"
	re "regexp"
	"strings"

	cfg "github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yamltags"
	yamlpatch "github.com/krishicks/yaml-patch"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

// ApplyProfiles returns configuration modified by the application
// of a list of profiles.
func ApplyProfiles(c *latest.SkaffoldConfig, opts cfg.SkaffoldOptions) error {
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
			return errors.Wrapf(err, "applying profile %s", name)
		}
	}

	return nil
}

func activatedProfiles(profiles []latest.Profile, opts cfg.SkaffoldOptions) ([]string, error) {
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

func isCommand(command string, opts cfg.SkaffoldOptions) bool {
	if command == "" {
		return true
	}

	return satisfies(command, opts.Command)
}

func isKubeContext(kubeContext string) (bool, error) {
	if kubeContext == "" {
		return true, nil
	}

	currentKubeConfig, err := kubectx.CurrentConfig()
	if err != nil {
		return false, errors.Wrap(err, "getting current cluster context")
	}

	return satisfies(kubeContext, currentKubeConfig.CurrentContext), nil
}

func satisfies(expected, actual string) bool {
	if strings.HasPrefix(expected, "!") {
		notExpected := expected[1:]

		return !matches(notExpected, actual)
	}

	return matches(expected, actual)
}

func matches(expected, actual string) bool {
	if actual == expected {
		return true
	}

	matcher, err := re.Compile(expected)
	if err != nil {
		logrus.Infof("profile activation criteria '%s' is not a valid regexp, falling back to string", expected)
		return false
	}

	return matcher.MatchString(actual)
}

func applyProfile(config *latest.SkaffoldConfig, profile latest.Profile) error {
	logrus.Infof("applying profile: %s", profile.Name)

	// Apply profile, field by field
	mergedV := reflect.Indirect(reflect.ValueOf(&config.Pipeline))
	configV := reflect.ValueOf(config.Pipeline)
	profileV := reflect.ValueOf(profile.Pipeline)

	profileT := profileV.Type()
	for i := 0; i < profileT.NumField(); i++ {
		name := profileT.Field(i).Name
		merged := overlayProfileField(name, configV.FieldByName(name).Interface(), profileV.FieldByName(name).Interface())
		mergedV.FieldByName(name).Set(reflect.ValueOf(merged))
	}

	// Remove the Profiles field from the returned config
	config.Profiles = nil

	if len(profile.Patches) == 0 {
		return nil
	}

	// Apply profile patches
	buf, err := yaml.Marshal(*config)
	if err != nil {
		return err
	}

	var patches []yamlpatch.Operation
	for _, patch := range profile.Patches {
		// Default patch operation to `replace`
		op := patch.Op
		if op == "" {
			op = "replace"
		}

		var value *yamlpatch.Node
		if v := patch.Value; v != nil {
			value = &v.Node
		}

		patch := yamlpatch.Operation{
			Op:    yamlpatch.Op(op),
			Path:  yamlpatch.OpPath(patch.Path),
			From:  yamlpatch.OpPath(patch.From),
			Value: value,
		}

		if !tryPatch(patch, buf) {
			return fmt.Errorf("invalid path: %s", patch.Path)
		}

		patches = append(patches, patch)
	}

	buf, err = yamlpatch.Patch(patches).Apply(buf)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(buf, config)
}

// tryPatch is here to verify patches one by one before we
// apply them because yamlpatch.Patch is known to panic when a path
// is not valid.
func tryPatch(patch yamlpatch.Operation, buf []byte) (valid bool) {
	defer func() {
		if errPanic := recover(); errPanic != nil {
			valid = false
		}
	}()

	_, err := yamlpatch.Patch([]yamlpatch.Operation{patch}).Apply(buf)
	return err == nil
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
		overlay := overlayProfileField(yamltags.YamlName(fieldType), configValue.Field(i).Interface(), profileValue.Field(i).Interface())
		finalConfig.Elem().FieldByName(fieldType.Name).Set(reflect.ValueOf(overlay))
	}
	return reflect.Indirect(finalConfig).Interface() // since finalConfig is a pointer, dereference it
}

func overlayProfileField(fieldName string, config interface{}, profile interface{}) interface{} {
	v := reflect.ValueOf(profile) // the profile itself
	t := reflect.TypeOf(profile)  // the type of the profile, used for getting struct field types
	logrus.Debugf("overlaying profile on config for field %s", fieldName)
	switch v.Kind() {
	case reflect.Struct:
		// check the first field of the struct for a oneOf yamltag.
		if util.IsOneOfField(t.Field(0)) {
			return overlayOneOfField(config, profile)
		}
		return overlayStructField(config, profile)
	case reflect.Slice:
		// either return the values provided in the profile, or the original values if none were provided.
		if v.Len() == 0 {
			return config
		}
		return v.Interface()
	case reflect.Ptr:
		// either return the values provided in the profile, or the original values if none were provided.
		if v.IsNil() {
			return config
		}
		return v.Interface()
	case reflect.Int:
		if v.Interface() == reflect.Zero(v.Type()).Interface() {
			return config
		}
		return v.Interface()
	default:
		logrus.Fatalf("Type mismatch in profile overlay for field '%s' with type %s; falling back to original config values", fieldName, v.Kind())
		return config
	}
}
