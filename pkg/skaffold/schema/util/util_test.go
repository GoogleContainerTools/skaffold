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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

const yamlFragment string = `global:
  enabled: true
  localstack: {}
`

func TestHelmOverridesMarshalling(t *testing.T) {
	h := &HelmOverrides{}
	err := yaml.Unmarshal([]byte(yamlFragment), h)
	testutil.CheckError(t, false, err)

	asJSON, err := json.Marshal(h)
	testutil.CheckError(t, false, err)

	err = json.Unmarshal(asJSON, h)
	testutil.CheckError(t, false, err)

	actual, err := yaml.Marshal(h)
	testutil.CheckErrorAndDeepEqual(t, false, err, yamlFragment, string(actual))
}

func TestHelmOverridesWhenEmbedded(t *testing.T) {
	h := HelmOverrides{}
	err := yaml.Unmarshal([]byte(yamlFragment), &h)
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

func TestYamlpatchNodeMarshalling(t *testing.T) {
	n := &YamlpatchNode{}
	err := yaml.Unmarshal([]byte(yamlFragment), n)
	testutil.CheckError(t, false, err)

	asJSON, err := json.Marshal(n)
	testutil.CheckError(t, false, err)

	err = json.Unmarshal(asJSON, n)
	testutil.CheckError(t, false, err)

	actual, err := yaml.Marshal(n)
	testutil.CheckErrorAndDeepEqual(t, false, err, yamlFragment, string(actual))
}

func TestYamlpatchNodeWhenEmbedded(t *testing.T) {
	n := &YamlpatchNode{}
	err := yaml.Unmarshal([]byte(yamlFragment), &n)
	testutil.CheckError(t, false, err)

	out, err := yaml.Marshal(struct {
		Node *YamlpatchNode `yaml:"value,omitempty"`
	}{n})

	testutil.CheckErrorAndDeepEqual(t, false, err, `value:
  global:
    enabled: true
    localstack: {}
`, string(out))
}

func TestFlatMap_UnmarshalYAML(t *testing.T) {
	y1 := `val1: foo1
val2: 
  val3: bar1
  val4: foo2
  val5:
    val6: bar2
`
	y2 := `val1: foo1
val2.val3: bar1
val2.val4: foo2
val2.val5.val6: bar2
`

	f1 := &FlatMap{}
	f2 := &FlatMap{}

	err := yaml.Unmarshal([]byte(y1), &f1)
	testutil.CheckError(t, false, err)

	err = yaml.Unmarshal([]byte(y2), &f2)
	testutil.CheckError(t, false, err)

	testutil.CheckDeepEqual(t, *f1, *f2)

	out, err := yaml.Marshal(struct {
		M *FlatMap `yaml:"value,omitempty"`
	}{f1})

	testutil.CheckErrorAndDeepEqual(t, false, err, `value:
  val1: foo1
  val2.val3: bar1
  val2.val4: foo2
  val2.val5.val6: bar2
`, string(out))
}
