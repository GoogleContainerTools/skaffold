/*
Copyright 2024 The Skaffold Authors

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

package tags

import (
	"reflect"
	"slices"
	"strings"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

// ApplyTemplates recursively traverses the provided interface{} value,
// expanding any string fields or elements that contain Go templates.
//
// Supported types for template expansion include:
//   - string
//   - *string
//   - []string
//   - []*string
//   - map[string]string
//   - map[string]*string
//
// The function uses the "skaffold" struct tag to identify fields that should be
// treated as templates. A field is considered a template if its "skaffold" tag
// contains the value "template".
//
// If an error occurs during template expansion, the function returns the error.
// Otherwise, it returns nil.
func ApplyTemplates(in interface{}) error {
	return applyTemplatesRecursive(reflect.ValueOf(in))
}

func applyTemplatesRecursive(v reflect.Value) error {
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			if isSupportedType(field) && containTemplateTag(v.Type().Field(i)) {
				if err := expandTemplate(field); err != nil {
					return err
				}
			} else if err := applyTemplatesRecursive(field); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if err := applyTemplatesRecursive(v.Index(i)); err != nil {
				return err
			}
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			value := v.MapIndex(key)
			if value.Kind() == reflect.Ptr {
				if err := applyTemplatesRecursive(value); err != nil {
					return err
				}
			} else {
				p := reflect.New(value.Type())
				p.Elem().Set(value)
				if err := applyTemplatesRecursive(p); err != nil {
					return err
				}
				v.SetMapIndex(key, p.Elem())
			}
		}
	case reflect.Interface, reflect.Ptr:
		if err := applyTemplatesRecursive(v.Elem()); err != nil {
			return err
		}
	}
	return nil
}

func isSupportedType(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return true
	case reflect.Ptr:
		return isSupportedType(v.Elem())
	case reflect.Slice, reflect.Array:
		if v.Len() == 0 {
			return false
		}
		k := v.Type().Elem().Kind()
		return k == reflect.String || (k == reflect.Ptr && v.Index(0).Elem().Kind() == reflect.String)
	case reflect.Map:
		if v.Len() == 0 {
			return false
		}
		iter := v.MapRange()
		iter.Next()
		mv := iter.Value()
		return reflect.Indirect(mv).Kind() == reflect.String
	default:
		return false
	}
}

func expandTemplate(v reflect.Value) error {
	switch v.Kind() {
	case reflect.String:
		updated, err := util.ExpandEnvTemplate(v.String(), nil)
		if err != nil {
			return err
		}
		// we want to keep the original template if expanding fails, otherwise update the value with expanded result.
		if !strings.Contains(updated, "<no value>") {
			v.SetString(updated)
		}
	case reflect.Ptr:
		return expandTemplate(v.Elem())
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if err := expandTemplate(v.Index(i)); err != nil {
				return err
			}
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			vv := v.MapIndex(key)
			if vv.Kind() == reflect.Ptr {
				if err := expandTemplate(vv); err != nil {
					return err
				}
			} else if vv.Kind() == reflect.String {
				updated, err := util.ExpandEnvTemplate(vv.String(), nil)
				if err != nil {
					return err
				}
				// we want to keep the original template if expanding fails, otherwise update the value with expanded result.
				if !strings.Contains(updated, "<no value>") {
					v.SetMapIndex(key, reflect.ValueOf(updated))
				}
			}
		}
	}
	return nil
}
func containTemplateTag(sf reflect.StructField) bool {
	v, ok := sf.Tag.Lookup("skaffold")
	if !ok {
		return ok
	}
	split := strings.Split(v, ",")
	return slices.Contains(split, "template")
}
