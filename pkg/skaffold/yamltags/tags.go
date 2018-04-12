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

package yamltags

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// ProcessStruct validates and processes the provided pointer to a struct.
func ProcessStruct(s interface{}) error {
	parentStruct := reflect.ValueOf(s).Elem()
	t := parentStruct.Type()

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
		// Recurse down the struct
		if val.Kind() == reflect.Struct {
			if err := ProcessStruct(val.Addr().Interface()); err != nil {
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
			yt = &RequiredTag{}
		case "default":
			yt = &DefaultTag{}
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
}

func (rt *RequiredTag) Load(s []string) error {
	return nil
}

func (rt *RequiredTag) Process(val reflect.Value) error {
	if isZeroValue(val) {
		return errors.New("required value not set")
	}
	return nil
}

type DefaultTag struct {
	dv string
}

func (dt *DefaultTag) Load(s []string) error {
	if len(s) != 2 {
		return fmt.Errorf("invalid default tag: %v, expected key=value", s)
	}
	dt.dv = s[1]
	return nil
}

func (dt *DefaultTag) Process(val reflect.Value) error {
	if !isZeroValue(val) {
		return nil
	}

	switch val.Kind() {
	case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(dt.dv, 0, 0)
		if err != nil {
			return err
		}
		val.SetInt(i)
	case reflect.String:
		val.SetString(dt.dv)
	}
	return nil
}

// A program can have many structs, that each have many oneOfSets
// each oneOfSet is a map of a set name to the list of fields that belong to that set
// only one field in that list can have a non-zero value.

var allOneOfs map[string]map[string][]string

func getOneOfSetsForStruct(structName string) map[string][]string {
	_, ok := allOneOfs[structName]
	if !ok {
		allOneOfs[structName] = map[string][]string{}
	}
	return allOneOfs[structName]
}

type OneOfTag struct {
	Field     reflect.StructField
	Parent    reflect.Value
	oneOfSets map[string][]string
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
	oot.oneOfSets[oot.setName] = append(oot.oneOfSets[oot.setName], oot.Field.Name)
	return nil
}

func (oot *OneOfTag) Process(val reflect.Value) error {
	if isZeroValue(val) {
		return nil
	}

	// This must exist because process is always called after Load.
	oneOfSet := oot.oneOfSets[oot.setName]
	for _, otherField := range oneOfSet {
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
	zv := reflect.Zero(val.Type()).Interface()
	return reflect.DeepEqual(zv, val.Interface())
}

func init() {
	allOneOfs = make(map[string]map[string][]string)
}
