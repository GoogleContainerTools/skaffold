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
	"strings"

	"github.com/blang/semver"
	yamlpatch "github.com/krishicks/yaml-patch"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/apiversion"
	cfg "github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta4"
	skutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yamltags"
)

// ApplyProfiles returns configuration modified by the application
// of a list of profiles.
func ApplyProfiles(c interface{}, opts cfg.SkaffoldOptions) error {
	ver, err := apiversion.Parse(c.(util.VersionedConfig).GetVersion())
	if err != nil {
		return err
	}
	pVal := reflect.ValueOf(c).Elem().FieldByName("Profiles")
	if !pVal.IsValid() {
		return nil
	}
	byName := profilesByName(pVal)

	profiles, contextSpecificProfiles, err := activatedProfiles(pVal, ver, opts)
	if err != nil {
		return fmt.Errorf("finding auto-activated profiles: %w", err)
	}

	for _, name := range profiles {
		profile, present := byName[name]
		if !present {
			return fmt.Errorf("couldn't find profile %s", name)
		}

		if err := applyProfile(c, ver, profile); err != nil {
			return fmt.Errorf("applying profile %q: %w", name, err)
		}
	}

	return checkKubeContextConsistency(contextSpecificProfiles, opts.KubeContext, reflect.ValueOf(c).Elem().FieldByName("Deploy").FieldByName("KubeContext").String())
}

func checkKubeContextConsistency(contextSpecificProfiles []string, cliContext, effectiveContext string) error {
	// cli flag takes precedence
	if cliContext != "" {
		return nil
	}

	kubeConfig, err := kubectx.CurrentConfig()
	if err != nil {
		return fmt.Errorf("getting current cluster context: %w", err)
	}
	currentContext := kubeConfig.CurrentContext

	// nothing to do
	if effectiveContext == "" || effectiveContext == currentContext || len(contextSpecificProfiles) == 0 {
		return nil
	}

	return fmt.Errorf("profiles %q were activated by kube-context %q, but the effective kube-context is %q -- please revise your `profiles.activation` and `deploy.kubeContext` configurations", contextSpecificProfiles, currentContext, effectiveContext)
}

// activatedProfiles returns the activated profiles and activated profiles which are kube-context specific.
// The latter matters for error reporting when the effective kube-context changes.
func activatedProfiles(profiles reflect.Value, version semver.Version, opts cfg.SkaffoldOptions) ([]string, []string, error) {
	activated, contextSpecificProfiles, err := checkActivations(profiles, version, opts)
	if err != nil {
		return nil, nil, err
	}

	for _, profile := range opts.Profiles {
		if strings.HasPrefix(profile, "-") {
			activated = removeValue(activated, strings.TrimPrefix(profile, "-"))
		} else {
			activated = append(activated, profile)
		}
	}

	return activated, contextSpecificProfiles, nil
}

// checkActivations converts each profile `Activation` to a known version and applies the corresponding `checkActivation` function.
func checkActivations(profiles reflect.Value, version semver.Version, opts cfg.SkaffoldOptions) ([]string, []string, error) {
	if !opts.ProfileAutoActivation {
		return nil, nil, nil
	}

	var activatedProfiles []string
	var contextSpecificProfiles []string
	v1b4, _ := apiversion.Parse(v1beta4.Version)
	for i := 0; i < profiles.Len(); i++ {
		activations := profiles.Index(i).FieldByName("Activation")
		profileName := profiles.Index(i).FieldByName("Name").String()
		if activations.IsValid() {
			for j := 0; j < activations.Len(); j++ {
				activation := activations.Index(j).Addr().Interface()
				var isActive, isContextActivated bool
				var err error
				switch {
				// Custom activation not supported before v1beta4
				case version.LT(v1b4):
					isActive, isContextActivated = true, false
				// when modifying the `Activation` struct add a case condition here corresponding to the new version and a corresponding `checkActivation` function.
				default:
					var versionedActivation *v1beta4.Activation
					skutil.CloneThroughJSON(activation, &versionedActivation)
					isActive, isContextActivated, err = checkActivation(versionedActivation, opts)
				}
				if err != nil {
					return nil, nil, fmt.Errorf("checking profile activation: %w", err)
				}
				if isActive {
					activatedProfiles = append(activatedProfiles, profileName)
				}
				if isContextActivated {
					contextSpecificProfiles = append(contextSpecificProfiles, profileName)
				}
			}
		}
	}
	return activatedProfiles, contextSpecificProfiles, nil
}

// checkActivation validates profile activation for the `v1beta4` version of `Activation` struct.
func checkActivation(a *v1beta4.Activation, opts cfg.SkaffoldOptions) (bool, bool, error) {
	command := isCommand(a.Command, opts)
	env, err := isEnv(a.Env)
	if err != nil {
		return false, false, err
	}

	kubeContext, err := isKubeContext(a.KubeContext, opts)
	if err != nil {
		return false, false, err
	}
	return command && env && kubeContext, command && env && kubeContext && a.KubeContext != "", nil
}

func removeValue(values []string, value string) []string {
	var updated []string

	for _, v := range values {
		if v != value {
			updated = append(updated, v)
		}
	}

	return updated
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

	envValue := os.Getenv(key)

	// Special case, since otherwise the regex substring check (`re.Compile("").MatchString(envValue)`)
	// would always match which is most probably not what the user wanted.
	if value == "" {
		return envValue == "", nil
	}

	return skutil.RegexEqual(value, envValue), nil
}

func isCommand(command string, opts cfg.SkaffoldOptions) bool {
	if command == "" {
		return true
	}

	return skutil.RegexEqual(command, opts.Command)
}

func isKubeContext(kubeContext string, opts cfg.SkaffoldOptions) (bool, error) {
	if kubeContext == "" {
		return true, nil
	}

	// cli flag takes precedence
	if opts.KubeContext != "" {
		return skutil.RegexEqual(kubeContext, opts.KubeContext), nil
	}

	currentKubeConfig, err := kubectx.CurrentConfig()
	if err != nil {
		return false, fmt.Errorf("getting current cluster context: %w", err)
	}

	return skutil.RegexEqual(kubeContext, currentKubeConfig.CurrentContext), nil
}
func applyProfile(config interface{}, version semver.Version, profile interface{}) error {
	c := reflect.Indirect(reflect.ValueOf(config))
	p := reflect.Indirect(reflect.ValueOf(profile))
	// Apply profile, field by field
	mergedV := c.FieldByName("Pipeline")
	configV := c.FieldByName("Pipeline")
	profileV := p.FieldByName("Pipeline")
	logrus.Infof("applying profile: %s", p.FieldByName("Name").Interface())

	profileT := profileV.Type()
	for i := 0; i < profileT.NumField(); i++ {
		name := profileT.Field(i).Name
		merged := overlayProfileField(name, configV.FieldByName(name).Interface(), profileV.FieldByName(name).Interface())
		mergedV.FieldByName(name).Set(reflect.ValueOf(merged))
	}

	// Remove the Profiles field from the returned config
	defer c.FieldByName("Profiles").Set(reflect.Zero(c.FieldByName("Profiles").Type()))

	if !p.FieldByName("Patches").IsValid() {
		return nil
	}
	profilePatches := p.FieldByName("Patches")

	// Apply profile patches
	buf, err := yaml.Marshal(c.Interface())
	if err != nil {
		return err
	}

	var patches []yamlpatch.Operation
	v1b4, _ := apiversion.Parse(v1beta4.Version)

	for i := 0; i < profilePatches.Len(); i++ {
		var patch *yamlpatch.Operation
		switch {
		// Profile patches not supported before v1beta4
		case version.LT(v1b4):
			return fmt.Errorf("profile patches not supported in v%v", v1b4)
		// when modifying the `JSONPatch` struct add a case condition here corresponding to the new version and a corresponding `createPatchOperation` function.
		default:
			patch, err = createPatchOperation(profilePatches.Index(i), buf)
		}

		if err != nil {
			return err
		}
		patches = append(patches, *patch)
	}

	buf, err = yamlpatch.Patch(patches).Apply(buf)
	if err != nil {
		return err
	}

	res := reflect.New(c.Type())
	if err = yaml.Unmarshal(buf, res.Interface()); err != nil {
		return err
	}
	c.Set(res.Elem())
	return nil
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

func createPatchOperation(patchProfile reflect.Value, data []byte) (*yamlpatch.Operation, error) {
	op := patchProfile.FieldByName("Op").String()
	path := patchProfile.FieldByName("Path").String()
	from := patchProfile.FieldByName("From").String()
	val := patchProfile.FieldByName("Value").Interface().(*util.YamlpatchNode)
	// Default patch operation to `replace`
	if op == "" {
		op = "replace"
	}

	var value *yamlpatch.Node
	if val != nil {
		value = &val.Node
	}

	patch := yamlpatch.Operation{
		Op:    yamlpatch.Op(op),
		Path:  yamlpatch.OpPath(path),
		From:  yamlpatch.OpPath(from),
		Value: value,
	}

	if !tryPatch(patch, data) {
		return nil, fmt.Errorf("invalid path: %s", patch.Path)
	}
	return &patch, nil
}

func profilesByName(profiles reflect.Value) map[string]interface{} {
	byName := make(map[string]interface{})
	for i := 0; i < profiles.Len(); i++ {
		profileName := profiles.Index(i).FieldByName("Name").String()
		profileVal := profiles.Index(i).Addr().Interface()
		byName[profileName] = profileVal
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
	case reflect.String:
		if reflect.DeepEqual("", v.Interface()) {
			return config
		}
		return v.Interface()
	default:
		logrus.Fatalf("Type mismatch in profile overlay for field '%s' with type %s; falling back to original config values", fieldName, v.Kind())
		return config
	}
}
