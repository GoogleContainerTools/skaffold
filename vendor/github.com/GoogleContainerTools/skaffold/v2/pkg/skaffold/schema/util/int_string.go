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

package util

import (
	"encoding/json"
	"fmt"
	"strconv"

	"gopkg.in/yaml.v3"
)

type IntOrString struct {
	Type   Type
	IntVal int
	StrVal string
}

type Type int

const (
	Int    Type = iota // The IntOrString holds an int.
	String             // The IntOrString holds a string.
)

// FromInt creates an IntOrString object with an int32 value.
func FromInt(val int) IntOrString {
	return IntOrString{Type: Int, IntVal: val}
}

// FromString creates an IntOrString object with a string value.
func FromString(val string) IntOrString {
	return IntOrString{Type: String, StrVal: val}
}

// String returns the string value, or the Itoa of the int value.
func (t *IntOrString) String() string {
	if t.Type == String {
		return t.StrVal
	}
	return strconv.Itoa(t.IntVal)
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (t *IntOrString) UnmarshalYAML(node *yaml.Node) error {
	val := node.Value
	i, err := strconv.Atoi(val)
	if err != nil {
		*t = FromString(val)
	} else {
		*t = FromInt(i)
	}
	return nil
}

// MarshalYAML implements the yaml.Marshaler interface.
func (t IntOrString) MarshalYAML() (interface{}, error) {
	switch t.Type {
	case Int:
		return t.IntVal, nil
	case String:
		return t.StrVal, nil
	default:
		return nil, fmt.Errorf("impossible IntOrString.Type")
	}
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (t *IntOrString) UnmarshalJSON(value []byte) error {
	if value[0] == '"' {
		t.Type = String
		return json.Unmarshal(value, &t.StrVal)
	}
	t.Type = Int
	return json.Unmarshal(value, &t.IntVal)
}

// MarshalJSON implements the json.Marshaler interface.
func (t IntOrString) MarshalJSON() ([]byte, error) {
	switch t.Type {
	case Int:
		return json.Marshal(t.IntVal)
	case String:
		return json.Marshal(t.StrVal)
	default:
		return []byte{}, fmt.Errorf("impossible IntOrString.Type")
	}
}
