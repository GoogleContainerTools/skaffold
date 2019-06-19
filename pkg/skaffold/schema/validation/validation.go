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

package validation

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yamltags"
)

var (
	// for testing
	validateYamltags = yamltags.ValidateStruct
)

// Process checks if the Skaffold pipeline is valid and returns all encountered errors as a concatenated string
func Process(config *latest.SkaffoldConfig) error {
	errs := visitStructs(config, validateYamltags)
	errs = append(errs, validateDockerNetworkMode(config.Build.Artifacts)...)
	errs = append(errs, validateCustomDependencies(config.Build.Artifacts)...)
	errs = append(errs, validateSyncRules(config.Build.Artifacts)...)

	if len(errs) == 0 {
		return nil
	}

	var messages []string
	for _, err := range errs {
		messages = append(messages, err.Error())
	}
	return fmt.Errorf(strings.Join(messages, " | "))
}

// validateDockerNetworkMode makes sure that networkMode is one of `bridge`, `none`, or `host` if set.
func validateDockerNetworkMode(artifacts []*latest.Artifact) (errs []error) {
	for _, a := range artifacts {
		if a.DockerArtifact == nil || a.DockerArtifact.NetworkMode == "" {
			continue
		}
		mode := strings.ToLower(a.DockerArtifact.NetworkMode)
		if mode == "none" || mode == "bridge" || mode == "host" {
			continue
		}
		errs = append(errs, fmt.Errorf("artifact %s has invalid networkMode '%s'", a.ImageName, mode))
	}
	return
}

// validateCustomDependencies makes sure that dependencies.ignore is only used in conjunction with dependencies.paths
func validateCustomDependencies(artifacts []*latest.Artifact) (errs []error) {
	for _, a := range artifacts {
		if a.CustomArtifact == nil {
			continue
		}
		if a.CustomArtifact.Dependencies.Ignore == nil {
			continue
		}
		if a.CustomArtifact.Dependencies.Dockerfile != nil || a.CustomArtifact.Dependencies.Command != "" {
			errs = append(errs, fmt.Errorf("artifact %s has invalid dependencies; dependencies.ignore can only be used in conjunction with dependencies.paths", a.ImageName))
		}
	}
	return
}

// visitStructs recursively visits all fields in the config and collects errors found by the visitor
func visitStructs(s interface{}, visitor func(interface{}) error) []error {
	v := reflect.ValueOf(s)
	t := reflect.TypeOf(s)

	switch v.Kind() {
	case reflect.Struct:
		var errs []error
		if err := visitor(v.Interface()); err != nil {
			errs = append(errs, err)
		}

		// also check all fields of the current struct
		for i := 0; i < t.NumField(); i++ {
			if !v.Field(i).CanInterface() {
				continue
			}
			if fieldErrs := visitStructs(v.Field(i).Interface(), visitor); fieldErrs != nil {
				errs = append(errs, fieldErrs...)
			}
		}

		return errs

	case reflect.Slice:
		// for slices check each element
		var errs []error
		for i := 0; i < v.Len(); i++ {
			if elemErrs := visitStructs(v.Index(i).Interface(), visitor); elemErrs != nil {
				errs = append(errs, elemErrs...)
			}
		}
		return errs

	case reflect.Ptr:
		// for pointers check the referenced value
		if v.IsNil() {
			return nil
		}
		return visitStructs(v.Elem().Interface(), visitor)

	default:
		// other values are fine
		return nil
	}
}

// validateSyncRules checks that all manual sync rules have a valid strip prefix
func validateSyncRules(artifacts []*latest.Artifact) []error {
	var errs []error
	for _, a := range artifacts {
		if a.Sync != nil {
			for _, r := range a.Sync.Manual {
				if !strings.HasPrefix(r.Src, r.Strip) {
					err := fmt.Errorf("sync rule pattern '%s' does not have prefix '%s'", r.Src, r.Strip)
					errs = append(errs, err)
				}
			}
		}
	}
	return errs
}
