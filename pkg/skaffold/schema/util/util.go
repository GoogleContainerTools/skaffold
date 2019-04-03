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
	"reflect"
	"strings"

	"gopkg.in/yaml.v2"
)

type VersionedConfig interface {
	GetVersion() string
	Upgrade() (VersionedConfig, error)
}

// HelmOverrides is a helper struct to aid with json serialization of map[string]interface{}
type HelmOverrides struct {
	Values map[string]interface{} `yaml:",inline"`
}

type inlineYaml struct {
	Yaml string
}

func (h *HelmOverrides) MarshalJSON() ([]byte, error) {
	yaml, err := yaml.Marshal(h)
	if err != nil {
		return nil, err
	}
	return json.Marshal(inlineYaml{string(yaml)})
}

func (h *HelmOverrides) UnmarshalJSON(text []byte) error {
	in := inlineYaml{}
	if err := json.Unmarshal(text, &in); err != nil {
		return err
	}
	return yaml.Unmarshal([]byte(in.Yaml), h)
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
