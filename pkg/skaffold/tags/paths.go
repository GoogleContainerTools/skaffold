/*
Copyright 2020 The Skaffold Authors

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
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/sirupsen/logrus"
)

// MakeFilePathsAbsolute recursively sets all fields marked with the tag `filepath` to absolute paths
func MakeFilePathsAbsolute(s interface{}, base string) error {
	errs := makeFilePathsAbsolute(s, base)
	if len(errs) == 0 {
		return nil
	}
	var messages []string
	for _, err := range errs {
		messages = append(messages, err.Error())
	}
	return fmt.Errorf(strings.Join(messages, " | "))
}

func makeFilePathsAbsolute(config interface{}, base string) []error {
	if config == nil {
		return nil
	}
	parentStruct := reflect.Indirect(reflect.ValueOf(config))

	switch parentStruct.Kind() {
	case reflect.Struct:
		t := parentStruct.Type()
		var errs []error
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			v := parentStruct.Field(i)
			if !v.CanInterface() {
				return errs
			}
			if filepathTagExists(f) {
				switch v.Interface().(type) {
				case string:
					path := v.String()
					if path == "" || filepath.IsAbs(path) {
						continue
					}
					v.SetString(filepath.Join(base, path))
					logrus.Tracef("setting absolute path for config field %q", f.Name)
				case []string:
					for j := 0; j < v.Len(); j++ {
						elem := v.Index(j)
						path := elem.String()
						if path == "" || filepath.IsAbs(path) {
							continue
						}
						elem.SetString(filepath.Join(base, path))
						logrus.Tracef("setting absolute paths for config field %q index %d", f.Name, j)
					}
				case map[string]string:
					for _, key := range v.MapKeys() {
						path := v.MapIndex(key).String()
						if path == "" || filepath.IsAbs(path) {
							continue
						}
						v.SetMapIndex(key, reflect.ValueOf(filepath.Join(base, path)))
						logrus.Tracef("setting absolute paths for config field %q key %q", f.Name, key.String())
					}
				default:
					return []error{fmt.Errorf("yaml tag `filepath` needs struct field %q to be string or string slice", f.Name)}
				}
				continue
			}

			if v.Kind() != reflect.Ptr {
				v = v.Addr()
			}
			if elemErrs := makeFilePathsAbsolute(v.Interface(), base); elemErrs != nil {
				errs = append(errs, elemErrs...)
			}
		}
		return errs
	case reflect.Slice:
		var errs []error
		for i := 0; i < parentStruct.Len(); i++ {
			elem := parentStruct.Index(i)
			if elem.Kind() != reflect.Ptr {
				elem = elem.Addr()
			}
			if !elem.CanInterface() {
				continue
			}
			if elemErrs := makeFilePathsAbsolute(elem.Interface(), base); elemErrs != nil {
				errs = append(errs, elemErrs...)
			}
		}
		return errs
	default:
		return nil
	}
}

func filepathTagExists(f reflect.StructField) bool {
	t, ok := f.Tag.Lookup("skaffold")
	if !ok {
		return false
	}
	return t == "filepath"
}
