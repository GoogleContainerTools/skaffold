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

package schema

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/defaults"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/validation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/walk"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

const (
	samplesRoot = "../../../docs/content/en/samples"
)

var (
	ignoredSamples = []string{"structureTest.yaml", "build.sh", "globalConfig.yaml"}
)

// Test that every example can be parsed and produces a valid
// Skaffold configuration.
func TestParseExamples(t *testing.T) {
	parseConfigFiles(t, "../../../examples")
	parseConfigFiles(t, "../../../integration/testdata/regressions")
}

// Samples are skaffold.yaml fragments that are used
// in the documentation.
func TestParseSamples(t *testing.T) {
	paths, err := walk.From(samplesRoot).WhenIsFile().CollectPaths()
	if err != nil {
		t.Fatalf("unable to list samples in %q", samplesRoot)
	}

	if len(paths) == 0 {
		t.Fatalf("did not find sample files in %q", samplesRoot)
	}

	for _, path := range paths {
		name := filepath.Base(path)
		if util.StrSliceContains(ignoredSamples, name) {
			continue
		}

		testutil.Run(t, name, func(t *testutil.T) {
			buf, err := ioutil.ReadFile(path)
			t.CheckNoError(err)

			checkSkaffoldConfig(t, addHeader(buf))
		})
	}
}

func checkSkaffoldConfig(t *testutil.T, yaml []byte) {
	root := t.NewTempDir()
	configFile := root.Path("skaffold.yaml")
	root.Write("skaffold.yaml", string(yaml))
	// create workspace directories referenced in these samples
	for _, d := range []string{"app", "node", "python", "leeroy-web", "leeroy-app", "backend", "base-service", "world-service"} {
		root.Mkdir(d)
	}
	root.Chdir()

	cfg, err := ParseConfigAndUpgrade(configFile, latest.Version)
	t.CheckNoError(err)

	err = defaults.Set(cfg.(*latest.SkaffoldConfig))
	t.CheckNoError(err)

	err = validation.Process(cfg.(*latest.SkaffoldConfig))
	t.CheckNoError(err)
}

func parseConfigFiles(t *testing.T, root string) {
	paths, err := walk.From(root).WhenHasName("skaffold.yaml").CollectPaths()
	if err != nil {
		t.Fatalf("unable to list skaffold configuration files in %q", root)
	}

	if len(paths) == 0 {
		t.Fatalf("did not find skaffold configuration files in %q", root)
	}

	for _, path := range paths {
		name := filepath.Base(filepath.Dir(path))

		testutil.Run(t, name, func(t *testutil.T) {
			buf, err := ioutil.ReadFile(path)
			t.CheckNoError(err)

			checkSkaffoldConfig(t, buf)
		})
	}
}

func addHeader(buf []byte) []byte {
	if bytes.HasPrefix(buf, []byte("apiVersion:")) {
		return buf
	}
	return []byte(fmt.Sprintf("apiVersion: %s\nkind: Config\n%s", latest.Version, buf))
}
