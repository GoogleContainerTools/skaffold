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
	"reflect"
	"strings"

	yamlpatch "github.com/krishicks/yaml-patch"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/yaml"
)

type VersionedConfig interface {
	GetVersion() string
	Upgrade() (VersionedConfig, error)
}

// HelmOverrides is a helper struct to aid with json serialization of map[string]interface{}
type HelmOverrides struct {
	Values map[string]interface{} `yaml:",inline"`
}

// FlatMap flattens deeply nested yaml into a map with corresponding dot separated keys
type FlatMap map[string]string

// MarshalJSON implements JSON marshalling by including the value as an inline yaml fragment.
func (h *HelmOverrides) MarshalJSON() ([]byte, error) {
	return marshalInlineYaml(h)
}

// UnmarshalYAML implements JSON unmarshalling by reading an inline yaml fragment.
func (h *HelmOverrides) UnmarshalJSON(text []byte) error {
	yml, err := unmarshalInlineYaml(text)
	if err != nil {
		return err
	}
	return yaml.Unmarshal([]byte(yml), h)
}

// YamlpatchNode wraps a `yamlpatch.Node` and makes it serializable to JSON.
// The yaml serialization needs to be implemented manually, because the node may be
// an arbitrary yaml fragment so that a field tag `yaml:",inline"` does not work here.
type YamlpatchNode struct {
	// node is an arbitrary yaml fragment
	Node yamlpatch.Node
}

// MarshalJSON implements JSON marshalling by including the value as an inline yaml fragment.
func (n *YamlpatchNode) MarshalJSON() ([]byte, error) {
	return marshalInlineYaml(n)
}

// UnmarshalYAML implements JSON unmarshalling by reading an inline yaml fragment.
func (n *YamlpatchNode) UnmarshalJSON(text []byte) error {
	yml, err := unmarshalInlineYaml(text)
	if err != nil {
		return err
	}
	return yaml.Unmarshal([]byte(yml), n)
}

// MarshalYAML implements yaml.Marshaler.
func (n *YamlpatchNode) MarshalYAML() (interface{}, error) {
	return n.Node.MarshalYAML()
}

// UnmarshalYAML implements yaml.Unmarshaler
func (n *YamlpatchNode) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return n.Node.UnmarshalYAML(unmarshal)
}

func (m *FlatMap) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var obj map[string]interface{}
	if err := unmarshal(&obj); err != nil {
		return err
	}
	result := make(map[string]string)
	buildFlatMap(obj, result, "")
	*m = result
	return nil
}

func buildFlatList(list []interface{}, result map[string]string, parentK string) {
	for idx, item := range list {
		currK := fmt.Sprintf("%v[%d]", parentK, idx)
		processItem(item, result, currK)
	}
}

func buildFlatMap(obj map[string]interface{}, result map[string]string, parentK string) {
	for key, item := range obj {
		currK := fmt.Sprintf("%v%v", parentK, key)
		processItem(item, result, currK)
	}
}

func processItem(item interface{}, result map[string]string, currK string) {
	switch v := item.(type) {
	case map[string]interface{}:
		buildFlatMap(v, result, fmt.Sprintf("%v.", currK))
	case []interface{}:
		buildFlatList(v, result, currK)
	default:
		result[currK] = fmt.Sprintf("%v", v)
	}
}

func marshalInlineYaml(in interface{}) ([]byte, error) {
	yaml, err := yaml.Marshal(in)
	if err != nil {
		return nil, err
	}
	return json.Marshal(string(yaml))
}

func unmarshalInlineYaml(text []byte) (string, error) {
	var in string
	err := json.Unmarshal(text, &in)
	return in, err
}

// IsOneOfField checks if a field is tagged with oneOf
func IsOneOfField(field reflect.StructField) bool {
	for _, tag := range strings.Split(field.Tag.Get("yamltags"), ",") {
		tagParts := strings.Split(tag, "=")

		if tagParts[0] == "oneOf" {
			return true
		}
	}
	return false
}
