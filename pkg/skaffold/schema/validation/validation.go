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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/sirupsen/logrus"
)

// ValidateSchema checks if the Skaffold pipeline is valid and returns all encountered errors as a concatenated string
func ValidateSchema(config *latest.SkaffoldPipeline) error {
	errs := validateOneOf(config)
	if errs == nil {
		return nil
	}
	var messages []string
	for _, err := range errs {
		messages = append(messages, err.Error())
	}
	return fmt.Errorf(strings.Join(messages, " | "))
}

// validateOneOf recursively visits all fields in the config and collects errors for oneOf conflicts
func validateOneOf(config interface{}) []error {
	v := reflect.ValueOf(config) // the config itself
	t := reflect.TypeOf(config)  // the type of the config, used for getting struct field types
	logrus.Debugf("validating oneOf on %s", t.Name())

	switch v.Kind() {
	case reflect.Struct:
		var errs []error

		// fields marked with oneOf should only be set once
		if t.NumField() > 1 && util.IsOneOfField(t.Field(0)) {
			var given []string
			for i := 0; i < t.NumField(); i++ {
				zero := reflect.Zero(v.Field(i).Type())
				if util.IsOneOfField(t.Field(i)) && v.Field(i).Interface() != zero.Interface() {
					given = append(given, yamlName(t.Field(i)))
				}
			}
			if len(given) > 1 {
				err := fmt.Errorf("only one of %s may be set", given)
				errs = append(errs, err)
			}
		}

		// also check all fields of the current struct
		for i := 0; i < t.NumField(); i++ {
			if fieldErrs := validateOneOf(v.Field(i).Interface()); fieldErrs != nil {
				errs = append(errs, fieldErrs...)
			}
		}

		return errs

	case reflect.Slice:
		// for slices check each element
		if v.Len() == 0 {
			return nil
		}
		var errs []error
		for i := 0; i < v.Len(); i++ {
			if elemErrs := validateOneOf(v.Index(i).Interface()); elemErrs != nil {
				errs = append(errs, elemErrs...)
			}
		}
		return errs

	case reflect.Ptr:
		// for pointers check the referenced value
		if v.IsNil() {
			return nil
		}
		return validateOneOf(v.Elem().Interface())

	default:
		// other values are fine
		return nil
	}
}

// yamlName retrieves the field name in the yaml
func yamlName(field reflect.StructField) string {
	return strings.Split(field.Tag.Get("yaml"), ",")[0]
}
