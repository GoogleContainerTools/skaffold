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

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/yaml"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
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

func TestFlatMap_UnmarshalYAMLNested(t *testing.T) {
	y1 := `
val1: foo1
values:
  val2:
    - val4: bar1
      val5: foo2
    - val6: bar2
      val7: foo3
  val3:
    - val8: bar3
      val9: foo4
`
	y2 := `val1: foo1
values.val2[0].val4: bar1
values.val2[0].val5: foo2
values.val2[1].val6: bar2
values.val2[1].val7: foo3
values.val3[0].val8: bar3
values.val3[0].val9: foo4
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
  values.val2[0].val4: bar1
  values.val2[0].val5: foo2
  values.val2[1].val6: bar2
  values.val2[1].val7: foo3
  values.val3[0].val8: bar3
  values.val3[0].val9: foo4
`, string(out))
}

func TestFlatMap_UnmarshalYAMLNestedArrays(t *testing.T) {
	inputYaml := `
name: nameValue
configs:
  name: configName
  defaults:
    - prop1: bar1
      prop2: foo1
    - prop1: bar2
      prop2: foo2
  mixlist:
    - prop1: bar3
      prop2:
       - foo3
    - element2
  matrix:
    - - i1
      - i2
    - - j1
      - j2
`

	expected := `name: nameValue
configs.name: configName
configs.defaults[0].prop1: bar1
configs.defaults[0].prop2: foo1
configs.defaults[1].prop1: bar2
configs.defaults[1].prop2: foo2
configs.mixlist[0].prop1: bar3
configs.mixlist[0].prop2[0]: foo3
configs.mixlist[1]: element2
configs.matrix[0][0]: i1
configs.matrix[0][1]: i2
configs.matrix[1][0]: j1
configs.matrix[1][1]: j2
`
	inputFlatmap := &FlatMap{}
	expectedFlatmap := &FlatMap{}

	err := yaml.Unmarshal([]byte(inputYaml), &inputFlatmap)
	testutil.CheckError(t, false, err)

	err = yaml.Unmarshal([]byte(expected), &expectedFlatmap)
	testutil.CheckError(t, false, err)

	testutil.CheckDeepEqual(t, *inputFlatmap, *expectedFlatmap)

	out, err := yaml.Marshal(struct {
		M *FlatMap `yaml:"value,omitempty"`
	}{inputFlatmap})

	testutil.CheckErrorAndDeepEqual(t, false, err, `value:
  configs.defaults[0].prop1: bar1
  configs.defaults[0].prop2: foo1
  configs.defaults[1].prop1: bar2
  configs.defaults[1].prop2: foo2
  configs.matrix[0][0]: i1
  configs.matrix[0][1]: i2
  configs.matrix[1][0]: j1
  configs.matrix[1][1]: j2
  configs.mixlist[0].prop1: bar3
  configs.mixlist[0].prop2[0]: foo3
  configs.mixlist[1]: element2
  configs.name: configName
  name: nameValue
`, string(out))
}
