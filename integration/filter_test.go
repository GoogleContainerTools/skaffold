/*
Copyright 2020 The Skaffold Authors

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

package integration

import (
	"testing"

	yaml "gopkg.in/yaml.v3"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestFilterPassthrough(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)
	// `filter` currently expects to receive a digested yaml
	renderedOutput := skaffold.Render().InDir("examples/getting-started").RunOrFailOutput(t)

	testutil.Run(t, "no filters should just pass through", func(t *testutil.T) {
		transformedOutput := skaffold.Filter().InDir("examples/getting-started").WithStdin(renderedOutput).RunOrFailOutput(t.T)

		t.CheckDeepEqual(unmarshalYaml(renderedOutput), unmarshalYaml(transformedOutput))
	})

	testutil.Run(t, "--build-artifacts=file with no filters should just pass through", func(t *testutil.T) {
		buildFile := t.TempFile("build.txt", []byte(`{"builds":[{"imageName":"doesnotexist","tag":"doesnotexist:notag"}]}`))
		transformedOutput := skaffold.Filter("--build-artifacts=" + buildFile).InDir("examples/getting-started").WithStdin(renderedOutput).RunOrFailOutput(t.T)

		t.CheckDeepEqual(unmarshalYaml(renderedOutput), unmarshalYaml(transformedOutput))
	})
}

func unmarshalYaml(data []byte) interface{} {
	m := make(map[interface{}]interface{})
	if err := yaml.Unmarshal(data, &m); err != nil {
		return err
	}
	return m
}
