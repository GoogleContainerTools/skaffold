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
	"context"
	"fmt"
	"os"
	"path"
	"reflect"
	"strings"
	"unicode"

	yamlpatch "github.com/krishicks/yaml-patch"

	cfg "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	kubectx "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/parser/configlocations"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/util"
	skutil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util/stringslice"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/yaml"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/yamltags"
)

// ApplyProfiles modifies the input skaffold configuration by the application
// of a list of profiles, and returns the list of applied profiles.
func ApplyProfiles(c *latest.SkaffoldConfig, fieldsOverrodeByProfile map[string]configlocations.YAMLOverrideInfo, opts cfg.SkaffoldOptions, namedProfiles []string) ([]string, map[string]configlocations.YAMLOverrideInfo, error) {
	byName := profilesByName(c.Profiles)

	profiles, contextSpecificProfiles, err := activatedProfiles(c.Profiles, opts, namedProfiles)
	if err != nil {
		return nil, nil, fmt.Errorf("finding auto-activated profiles: %w", err)
	}
	for _, name := range profiles {
		profile, present := byName[name]
		if !present {
			return nil, nil, fmt.Errorf("couldn't find profile %s", name)
		}

		if err := applyProfile(c, fieldsOverrodeByProfile, profile); err != nil {
			return nil, nil, fmt.Errorf("applying profile %q: %w", name, err)
		}
	}

	// remove profiles section for run modes where profiles are already merged into the main pipeline
	switch opts.Mode() {
	case cfg.RunModes.Build, cfg.RunModes.Dev, cfg.RunModes.Deploy, cfg.RunModes.Debug, cfg.RunModes.Render, cfg.RunModes.Run, cfg.RunModes.Diagnose, cfg.RunModes.Delete:
		c.Profiles = nil
	}
	return profiles, fieldsOverrodeByProfile, checkKubeContextConsistency(contextSpecificProfiles, opts.KubeContext, c.Deploy.KubeContext)
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
func activatedProfiles(profiles []latest.Profile, opts cfg.SkaffoldOptions, namedProfiles []string) ([]string, []string, error) {
	var activated []string
	var contextSpecificProfiles []string

	if opts.ProfileAutoActivation {
		// Auto-activated profiles
		for _, profile := range profiles {
			isActivated, isCtxSpecific, err := isProfileActivated(profile, opts)
			if err != nil {
				return nil, nil, err
			}

			if isActivated {
				if isCtxSpecific {
					contextSpecificProfiles = append(contextSpecificProfiles, profile.Name)
				}
				activated = append(activated, profile.Name)
			}
		}
	}

	var allProfileNames []string
	for _, p := range profiles {
		allProfileNames = append(allProfileNames, p.Name)
	}

	for _, profile := range namedProfiles {
		if strings.HasPrefix(profile, "-") {
			activated = removeValue(activated, strings.TrimPrefix(profile, "-"))
		} else if stringslice.Contains(allProfileNames, profile) && !stringslice.Contains(activated, profile) {
			activated = append(activated, profile)
		}
	}

	return activated, contextSpecificProfiles, nil
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

func isProfileActivated(profile latest.Profile, opts cfg.SkaffoldOptions) (bool, bool, error) {
	if profile.RequiresAllActivations {
		return isAllActivationsTriggered(profile, opts)
	}
	return isAnyActivationTriggered(profile, opts)
}

// isAnyActivationTriggered returns true when the profile has at least one of its activations triggered.
// When the profile is activated, the second returned value indicates whether the profile is context-specific.
func isAnyActivationTriggered(profile latest.Profile, opts cfg.SkaffoldOptions) (bool, bool, error) {
	for _, cond := range profile.Activation {
		activated, err := isActivationTriggered(cond, opts)
		if err != nil {
			return false, false, err
		}
		if activated {
			isContextSpecific := cond.KubeContext != ""
			return true, isContextSpecific, nil
		}
	}
	return false, false, nil
}

// isAllActivationsTriggered returns true when the profile has all of its activations triggered.
// When the profile is activated, the second returned value indicates whether the profile is context-specific.
func isAllActivationsTriggered(profile latest.Profile, opts cfg.SkaffoldOptions) (bool, bool, error) {
	// no activation conditions means no auto-activation
	if len(profile.Activation) == 0 {
		return false, false, nil
	}

	isContextSpecific := false
	for _, cond := range profile.Activation {
		activated, err := isActivationTriggered(cond, opts)
		if err != nil {
			return false, false, err
		}
		if !activated {
			return false, false, nil
		}
		isContextSpecific = isContextSpecific || cond.KubeContext != ""
	}

	return true, isContextSpecific, nil
}

func isActivationTriggered(cond latest.Activation, opts cfg.SkaffoldOptions) (bool, error) {
	command := isCommand(cond.Command, opts)

	env, err := isEnv(cond.Env)
	if err != nil {
		return false, err
	}

	kubeContext, err := isKubeContext(cond.KubeContext, opts)
	if err != nil {
		return false, err
	}
	return command && env && kubeContext, nil
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

func applyProfile(config *latest.SkaffoldConfig, fieldsOverrodeByProfile map[string]configlocations.YAMLOverrideInfo, profile latest.Profile) error {
	log.Entry(context.TODO()).Infof("applying profile: %s", profile.Name)

	// Apply profile, field by field
	mergedV := reflect.Indirect(reflect.ValueOf(&config.Pipeline))
	configV := reflect.ValueOf(config.Pipeline)
	profileV := reflect.ValueOf(profile.Pipeline)

	profileT := profileV.Type()

	for i := 0; i < profileT.NumField(); i++ {
		name := profileT.Field(i).Name
		merged := overlayProfileField(profile.Name, name, yamltags.YamlName(profileT.Field(i)), []string{}, fieldsOverrodeByProfile, configV.FieldByName(name).Interface(), profileV.FieldByName(name).Interface())
		mergedV.FieldByName(name).Set(reflect.ValueOf(merged))
	}

	if len(profile.Patches) == 0 {
		return nil
	}

	// Apply profile patches
	buf, err := yaml.Marshal(*config)
	if err != nil {
		return err
	}

	for i, patch := range profile.Patches {
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

		updated, valid := tryPatch(patch, buf)
		buf = updated
		if !valid {
			return fmt.Errorf("invalid path: %s", patch.Path)
		}

		// TODO(aaron-prindle) we can ignore - op:'remove' - patch profiles as there is no corresponding schema object for them (it is removed already)
		yamlOverrideInfo := configlocations.YAMLOverrideInfo{
			ProfileName:    profile.Name,
			PatchOperation: string(patch.Op),
			PatchIndex:     i,
		}
		if patch.Op == "copy" {
			yamlOverrideInfo.PatchCopyFrom = patch.From.String()
		}
		if patch.Op != "remove" { // TODO(aaron-prindle) yamlpatch lib doesn't export op types, should copy paste the types elsewhere and refer to that here
			fieldsOverrodeByProfile[string(patch.Path)] = yamlOverrideInfo
		}
	}

	if err != nil {
		return err
	}

	*config = latest.SkaffoldConfig{}
	return yaml.Unmarshal(buf, config)
}

// tryPatch is here to verify patches one by one before we
// apply them because yamlpatch.Patch is known to panic when a path
// is not valid.
func tryPatch(patch yamlpatch.Operation, buf []byte) (patched []byte, valid bool) {
	defer func() {
		if errPanic := recover(); errPanic != nil {
			valid = false
		}
	}()

	updated, err := yamlpatch.Patch([]yamlpatch.Operation{patch}).Apply(buf)
	return updated, err == nil
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
func overlayOneOfField(profileName string, fieldPath []string, fieldsOverrodeByProfile map[string]configlocations.YAMLOverrideInfo, config interface{}, profile interface{}) interface{} {
	v := reflect.ValueOf(profile) // the profile itself
	t := reflect.TypeOf(profile)  // the type of the profile, used for getting struct field types
	for i := 0; i < v.NumField(); i++ {
		fieldType := t.Field(i)              // the field type (e.g. 'LocalBuild' for BuildConfig)
		fieldValue := v.Field(i).Interface() // the value of the field itself

		if fieldValue != nil && !reflect.ValueOf(fieldValue).IsNil() {
			yamltags.YamlName(fieldType)
			fieldPath = append(fieldPath, yamltags.YamlName(fieldType))
			ret := reflect.New(t)                                                   // New(t) returns a Value representing pointer to new zero value for type t
			ret.Elem().FieldByName(fieldType.Name).Set(reflect.ValueOf(fieldValue)) // set the value
			fieldsOverrodeByProfile["/"+path.Join(fieldPath...)] = configlocations.YAMLOverrideInfo{
				ProfileName: profileName,
				PatchIndex:  -1,
			}
			return reflect.Indirect(ret).Interface() // since ret is a pointer, dereference it
		}
	}
	// if we're here, we didn't find any values set in the profile config. just return the original.
	log.Entry(context.TODO()).Infof("no values found in profile for field %s, using original config values", t.Name())
	return config
}

func overlayStructField(profileName string, fieldPath []string, fieldsOverrodeByProfile map[string]configlocations.YAMLOverrideInfo, config interface{}, profile interface{}) interface{} {
	// we already know the top level fields for whatever struct we have are themselves structs
	// (and not one-of values), so we need to recursively overlay them
	configValue := reflect.ValueOf(config)
	profileValue := reflect.ValueOf(profile)
	t := reflect.TypeOf(profile)
	finalConfig := reflect.New(t)

	for i := 0; i < profileValue.NumField(); i++ {
		fieldType := t.Field(i)
		yamlFieldName := yamltags.YamlName(fieldType)
		var first rune
		for _, c := range yamlFieldName {
			first = c
			break
		}
		if !unicode.IsLower(first) {
			yamlFieldName = "-"
		}
		overlay := overlayProfileField(profileName, yamltags.YamlName(fieldType), yamlFieldName, fieldPath, fieldsOverrodeByProfile, configValue.Field(i).Interface(), profileValue.Field(i).Interface())
		finalConfig.Elem().FieldByName(fieldType.Name).Set(reflect.ValueOf(overlay))
	}
	return reflect.Indirect(finalConfig).Interface() // since finalConfig is a pointer, dereference it
}

// I could either get struct names in a flat manner and just not care about name collisions for now
// OR I could make a tree of fieldNames and then check the tree when doing the profile yaml node stuff
func overlayProfileField(profileName, fieldName string, yamlFieldName string, fieldPath []string, fieldsOverrodeByProfile map[string]configlocations.YAMLOverrideInfo, config interface{}, profile interface{}) interface{} {
	v := reflect.ValueOf(profile) // the profile itself
	t := reflect.TypeOf(profile)  // the type of the profile, used for getting struct field types
	log.Entry(context.TODO()).Debugf("overlaying profile on config for field %s", fieldName)

	if yamlFieldName != "-" {
		fieldPath = append(fieldPath, yamlFieldName)
	}

	switch v.Kind() {
	case reflect.Struct:
		// check the first field of the struct for a oneOf yamltag.
		if util.IsOneOfField(t.Field(0)) {
			return overlayOneOfField(profileName, fieldPath, fieldsOverrodeByProfile, config, profile)
		}
		return overlayStructField(profileName, fieldPath, fieldsOverrodeByProfile, config, profile)
	case reflect.Slice:
		// either return the values provided in the profile, or the original values if none were provided.
		if v.Len() == 0 {
			return config
		}
		fieldsOverrodeByProfile["/"+path.Join(fieldPath...)] = configlocations.YAMLOverrideInfo{
			ProfileName: profileName,
			PatchIndex:  -1,
		}
		return v.Interface()
	case reflect.Ptr:
		// either return the values provided in the profile, or the original values if none were provided.
		if v.IsNil() {
			return config
		}
		fieldsOverrodeByProfile["/"+path.Join(fieldPath...)] = configlocations.YAMLOverrideInfo{
			ProfileName: profileName,
			PatchIndex:  -1,
		}
		return v.Interface()
	case reflect.Int:
		if v.Interface() == reflect.Zero(v.Type()).Interface() {
			return config
		}
		fieldsOverrodeByProfile["/"+path.Join(fieldPath...)] = configlocations.YAMLOverrideInfo{
			ProfileName: profileName,
			PatchIndex:  -1,
		}
		return v.Interface()
	case reflect.Bool:
		if v.Interface() == reflect.Zero(v.Type()).Interface() {
			return config
		}
		fieldsOverrodeByProfile["/"+path.Join(fieldPath...)] = configlocations.YAMLOverrideInfo{
			ProfileName: profileName,
			PatchIndex:  -1,
		}
		return v.Interface()
	case reflect.String:
		if reflect.DeepEqual("", v.Interface()) {
			return config
		}
		fieldsOverrodeByProfile["/"+path.Join(fieldPath...)] = configlocations.YAMLOverrideInfo{
			ProfileName: profileName,
			PatchIndex:  -1,
		}
		return v.Interface()
	default:
		log.Entry(context.TODO()).Fatalf("Type mismatch in profile overlay for field '%s' with type %s; falling back to original config values", fieldName, v.Kind())
		return config
	}
}
