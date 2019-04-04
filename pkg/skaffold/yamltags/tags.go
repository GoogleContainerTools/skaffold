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

package yamltags

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/sirupsen/logrus"
)

type FieldSet map[string]struct{}

// ValidateStruct validates and processes the provided pointer to a struct.
func ValidateStruct(s interface{}) error {
	parentStruct := reflect.Indirect(reflect.ValueOf(s))
	t := parentStruct.Type()
	logrus.Debugf("validating yamltags of struct %s", t.Name())

	// Loop through the fields on the struct, looking for tags.
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		val := parentStruct.Field(i)
		field := parentStruct.Type().Field(i)
		if tags, ok := f.Tag.Lookup("yamltags"); ok {
			if err := ProcessTags(tags, val, parentStruct, field); err != nil {
				return err
			}
		}
	}
	return nil
}

func ProcessTags(yamltags string, val reflect.Value, parentStruct reflect.Value, field reflect.StructField) error {
	tags := strings.Split(yamltags, ",")
	for _, tag := range tags {
		tagParts := strings.Split(tag, "=")
		var yt YamlTag
		switch tagParts[0] {
		case "required":
			yt = &RequiredTag{
				Field: field,
			}
		case "oneOf":
			yt = &OneOfTag{
				Field:  field,
				Parent: parentStruct,
			}
		}
		if err := yt.Load(tagParts); err != nil {
			return err
		}
		if err := yt.Process(val); err != nil {
			return err
		}
	}
	return nil
}

type YamlTag interface {
	Load([]string) error
	Process(reflect.Value) error
}

type RequiredTag struct {
	Field reflect.StructField
}

func (rt *RequiredTag) Load(s []string) error {
	return nil
}

func (rt *RequiredTag) Process(val reflect.Value) error {
	if isZeroValue(val) {
		if tags, ok := rt.Field.Tag.Lookup("yaml"); ok {
			return fmt.Errorf("required value not set: %s", strings.Split(tags, ",")[0])
		}
		return fmt.Errorf("required value not set: %s", rt.Field.Name)
	}
	return nil
}

// A program can have many structs, that each have many oneOfSets.
// Each oneOfSet is a map of a oneOf-set name to the set of fields that belong to that oneOf-set
// Only one field in that set may have a non-zero value.

var allOneOfs map[string]map[string]FieldSet

func getOneOfSetsForStruct(structName string) map[string]FieldSet {
	_, ok := allOneOfs[structName]
	if !ok {
		allOneOfs[structName] = map[string]FieldSet{}
	}
	return allOneOfs[structName]
}

type OneOfTag struct {
	Field     reflect.StructField
	Parent    reflect.Value
	oneOfSets map[string]FieldSet
	setName   string
}

func (oot *OneOfTag) Load(s []string) error {
	if len(s) != 2 {
		return fmt.Errorf("invalid default struct tag: %v, expected key=value", s)
	}
	oot.setName = s[1]

	// Fetch the right oneOfSet for the struct.
	structName := oot.Parent.Type().Name()
	oot.oneOfSets = getOneOfSetsForStruct(structName)

	// Add this field to the oneOfSet
	if _, ok := oot.oneOfSets[oot.setName]; !ok {
		oot.oneOfSets[oot.setName] = FieldSet{}
	}
	oot.oneOfSets[oot.setName][oot.Field.Name] = struct{}{}
	return nil
}

func (oot *OneOfTag) Process(val reflect.Value) error {
	if isZeroValue(val) {
		return nil
	}

	// This must exist because process is always called after Load.
	oneOfSet := oot.oneOfSets[oot.setName]
	for otherField := range oneOfSet {
		if otherField == oot.Field.Name {
			continue
		}
		field := oot.Parent.FieldByName(otherField)
		if !isZeroValue(field) {
			return fmt.Errorf("only one element in set %s can be set. got %s and %s", oot.setName, otherField, oot.Field.Name)
		}
	}
	return nil
}

func isZeroValue(val reflect.Value) bool {
	if val.Kind() == reflect.Invalid {
		return true
	}
	zv := reflect.Zero(val.Type()).Interface()
	return reflect.DeepEqual(zv, val.Interface())
}

func init() {
	allOneOfs = make(map[string]map[string]FieldSet)
}
