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
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestFromInt(t *testing.T) {
	i := FromInt(42)
	if i.Type != Int || i.IntVal != 42 {
		t.Errorf("Expected IntVal=42, got %+v", i)
	}
}

func TestFromString(t *testing.T) {
	i := FromString("test")
	if i.Type != String || i.StrVal != "test" {
		t.Errorf("Expected StrVal=\"test\", got %+v", i)
	}
}

func TestString(t *testing.T) {
	cases := []struct {
		input  IntOrString
		result string
	}{
		{FromInt(8080), "8080"},
		{FromString("http"), "http"},
	}

	for _, c := range cases {
		testutil.CheckDeepEqual(t, c.result, c.input.String())
	}
}

type IntOrStringHolder struct {
	Val IntOrString `json:"val" yaml:"val"`
}

func TestUnmarshalJSON(t *testing.T) {
	cases := []struct {
		input  string
		result IntOrString
	}{
		{"{\"val\": 8080}", FromInt(8080)},
		{"{\"val\": \"http\"}", FromString("http")},
	}

	for _, c := range cases {
		var result IntOrStringHolder
		if err := json.Unmarshal([]byte(c.input), &result); err != nil {
			t.Errorf("Failed to unmarshal input '%v': %v", c.input, err)
		}
		if result.Val != c.result {
			t.Errorf("Failed to unmarshal input '%v': expected %+v, got %+v", c.input, c.result, result)
		}
	}
}

func TestMarshalJSON(t *testing.T) {
	cases := []struct {
		input  IntOrString
		result string
	}{
		{FromInt(8080), "{\"val\":8080}"},
		{FromString("http"), "{\"val\":\"http\"}"},
	}

	for _, c := range cases {
		input := IntOrStringHolder{c.input}
		result, err := json.Marshal(&input)
		if err != nil {
			t.Errorf("Failed to marshal input '%v': %v", input, err)
		}
		if string(result) != c.result {
			t.Errorf("Failed to marshal input '%v': expected: %+v, got %q", input, c.result, string(result))
		}
	}
}

func TestUnmarshalYaml(t *testing.T) {
	cases := []struct {
		input  string
		result IntOrString
	}{
		{"{\"val\": 8080}", FromInt(8080)},
		{"{\"val\": \"http\"}", FromString("http")},
	}

	for _, c := range cases {
		var result IntOrStringHolder
		if err := yaml.Unmarshal([]byte(c.input), &result); err != nil {
			t.Errorf("Failed to unmarshal input '%v': %v", c.input, err)
		}
		if result.Val != c.result {
			t.Errorf("Failed to unmarshal input '%v': expected %+v, got %+v", c.input, c.result, result)
		}
	}
}

func TestMarshalYaml(t *testing.T) {
	cases := []struct {
		input  IntOrString
		result string
	}{
		{FromInt(8080), "val: 8080\n"},
		{FromString("http"), "val: http\n"},
	}

	for _, c := range cases {
		input := IntOrStringHolder{c.input}
		result, err := yaml.Marshal(&input)
		if err != nil {
			t.Errorf("Failed to marshal input '%v': %v", input, err)
		}
		if string(result) != c.result {
			t.Errorf("Failed to marshal input '%v': expected: %+v, got %q", input, c.result, string(result))
		}
	}
}
