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

	"github.com/GoogleContainerTools/skaffold/testutil"
	yaml "gopkg.in/yaml.v2"
)

const overridesYaml string = `global:
  enabled: true
  localstack: {}
`

func TestHelmOverridesMarshalling(t *testing.T) {
	h := &HelmOverrides{}
	err := yaml.Unmarshal([]byte(overridesYaml), h)
	testutil.CheckError(t, false, err)

	asJSON, err := json.Marshal(h)
	testutil.CheckError(t, false, err)

	err = json.Unmarshal(asJSON, h)
	testutil.CheckError(t, false, err)

	actual, err := yaml.Marshal(h)
	testutil.CheckErrorAndDeepEqual(t, false, err, overridesYaml, string(actual))
}

func TestHelmOverridesWhenEmbedded(t *testing.T) {
	h := HelmOverrides{}
	err := yaml.Unmarshal([]byte(overridesYaml), &h)
	testutil.CheckError(t, false, err)

	out, err := yaml.Marshal(struct {
		Overrides HelmOverrides `yaml:"overrides,omitempty"`
	}{h})

	testutil.CheckErrorAndDeepEqual(t, false, err, `overrides:
  global:
    enabled: true
    localstack: {}
`, string(out))
}
