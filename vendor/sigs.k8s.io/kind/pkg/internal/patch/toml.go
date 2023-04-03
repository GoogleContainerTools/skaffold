/*
Copyright 2019 The Kubernetes Authors.

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

package patch

import (
	"bytes"
	"encoding/json"

	burntoml "github.com/BurntSushi/toml"
	jsonpatch "github.com/evanphx/json-patch/v5"
	toml "github.com/pelletier/go-toml"
	yaml "gopkg.in/yaml.v3"

	"sigs.k8s.io/kind/pkg/errors"
)

// TOML patches toPatch with the patches (should be TOML merge patches) and patches6902 (should be JSON 6902 patches)
func TOML(toPatch string, patches []string, patches6902 []string) (string, error) {
	// convert to JSON for patching
	j, err := tomlToJSON([]byte(toPatch))
	if err != nil {
		return "", err
	}
	// apply merge patches
	for _, patch := range patches {
		pj, err := tomlToJSON([]byte(patch))
		if err != nil {
			return "", err
		}
		patched, err := jsonpatch.MergePatch(j, pj)
		if err != nil {
			return "", errors.WithStack(err)
		}
		j = patched
	}
	// apply JSON 6902 patches
	for _, patch6902 := range patches6902 {
		patch, err := jsonpatch.DecodePatch([]byte(patch6902))
		if err != nil {
			return "", errors.WithStack(err)
		}
		patched, err := patch.Apply(j)
		if err != nil {
			return "", errors.WithStack(err)
		}
		j = patched
	}
	// convert result back to TOML
	return jsonToTOMLString(j)
}

// tomlToJSON converts arbitrary TOML to JSON
func tomlToJSON(t []byte) ([]byte, error) {
	// we use github.com.pelletier/go-toml here to unmarshal arbitrary TOML to JSON
	tree, err := toml.LoadBytes(t)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	b, err := json.Marshal(tree.ToMap())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return b, nil
}

// jsonToTOMLString converts arbitrary JSON to TOML
func jsonToTOMLString(j []byte) (string, error) {
	var unstruct interface{}
	// We are using yaml.Unmarshal here (instead of json.Unmarshal) because the
	// Go JSON library doesn't try to pick the right number type (int, float,
	// etc.) when unmarshalling to interface{}, it just picks float64
	// universally. go-yaml does go through the effort of picking the right
	// number type, so we can preserve number type throughout this process.
	if err := yaml.Unmarshal(j, &unstruct); err != nil {
		return "", errors.WithStack(err)
	}
	// we use github.com/BurntSushi/toml here because github.com.pelletier/go-toml
	// can only marshal structs AND BurntSushi/toml is what contained uses
	// and has more canonically formatted output (we initially plan to use
	// this package for patching containerd config)
	var buff bytes.Buffer
	if err := burntoml.NewEncoder(&buff).Encode(unstruct); err != nil {
		return "", errors.WithStack(err)
	}
	return buff.String(), nil
}
