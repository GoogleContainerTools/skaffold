/*
Copyright 2019 The Knative Authors

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

package apis

import (
	"context"
	"reflect"
	"strings"
)

const (
	deprecatedPrefix = "Deprecated"
)

// CheckDeprecated checks whether the provided named deprecated fields
// are set in a context where deprecation is disallowed.
// This is a shallow check.
func CheckDeprecated(ctx context.Context, obj interface{}) *FieldError {
	return CheckDeprecatedUpdate(ctx, obj, nil)
}

// CheckDeprecated checks whether the provided named deprecated fields
// are set in a context where deprecation is disallowed.
// This is a json shallow check. We will recursively check inlined structs.
func CheckDeprecatedUpdate(ctx context.Context, obj interface{}, original interface{}) *FieldError {
	if IsDeprecatedAllowed(ctx) {
		return nil
	}

	var errs *FieldError
	objFields, objInlined := getPrefixedNamedFieldValues(deprecatedPrefix, obj)

	if nonZero(reflect.ValueOf(original)) {
		originalFields, originalInlined := getPrefixedNamedFieldValues(deprecatedPrefix, original)

		// We only have to walk obj Fields because the assumption is that obj
		// and original are of the same type.
		for name, value := range objFields {
			if nonZero(value) {
				if differ(originalFields[name], value) {
					// Not allowed to update the value.
					errs = errs.Also(ErrDisallowedUpdateDeprecatedFields(name))
				}
			}
		}
		// Look for deprecated inlined updates.
		if len(objInlined) > 0 {
			for name, value := range objInlined {
				errs = errs.Also(CheckDeprecatedUpdate(ctx, value, originalInlined[name]))
			}
		}
	} else {
		for name, value := range objFields {
			if nonZero(value) {
				// Not allowed to set the value.
				errs = errs.Also(ErrDisallowedFields(name))
			}
		}
		// Look for deprecated inlined creates.
		if len(objInlined) > 0 {
			for _, value := range objInlined {
				errs = errs.Also(CheckDeprecated(ctx, value))
			}
		}
	}
	return errs
}

func getPrefixedNamedFieldValues(prefix string, obj interface{}) (map[string]reflect.Value, map[string]interface{}) {
	fields := make(map[string]reflect.Value, 0)
	inlined := make(map[string]interface{}, 0)

	objValue := reflect.Indirect(reflect.ValueOf(obj))

	// If res is not valid or a struct, don't even try to use it.
	if !objValue.IsValid() || objValue.Kind() != reflect.Struct {
		return fields, inlined
	}

	for i := 0; i < objValue.NumField(); i++ {
		tf := objValue.Type().Field(i)
		if v := objValue.Field(i); v.IsValid() {
			jTag := tf.Tag.Get("json")
			if strings.HasPrefix(tf.Name, prefix) {
				name := strings.Split(jTag, ",")[0]
				if name == "" {
					// Default to field name in go struct if no json name.
					name = tf.Name
				}
				fields[name] = v
			} else if jTag == ",inline" {
				inlined[tf.Name] = getInterface(v)
			}
		}
	}
	return fields, inlined
}

// getInterface returns the interface value of the reflected object.
func getInterface(a reflect.Value) interface{} {
	switch a.Kind() {
	case reflect.Ptr:
		if a.IsNil() {
			return nil
		}
		return a.Elem().Interface()

	case reflect.Map, reflect.Slice, reflect.Array:
		return a.Elem().Interface()

	// This is a nil interface{} type.
	case reflect.Invalid:
		return nil

	default:
		return a.Interface()
	}
}

// nonZero returns true if a is nil or reflect.Zero.
func nonZero(a reflect.Value) bool {
	switch a.Kind() {
	case reflect.Ptr:
		if a.IsNil() {
			return false
		}
		return nonZero(a.Elem())

	case reflect.Map, reflect.Slice, reflect.Array:
		if a.IsNil() {
			return false
		}
		return true

	// This is a nil interface{} type.
	case reflect.Invalid:
		return false

	default:
		if reflect.DeepEqual(a.Interface(), reflect.Zero(a.Type()).Interface()) {
			return false
		}
		return true
	}
}

// differ returns true if a != b
func differ(a, b reflect.Value) bool {
	if a.Kind() != b.Kind() {
		return true
	}

	switch a.Kind() {
	case reflect.Ptr:
		if a.IsNil() || b.IsNil() {
			return a.IsNil() != b.IsNil()
		}
		return differ(a.Elem(), b.Elem())

	default:
		if reflect.DeepEqual(a.Interface(), b.Interface()) {
			return false
		}
		return true
	}
}
