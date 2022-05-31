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

package validation

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/parser"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/defaults"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util/stringslice"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/walk"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

const (
	samplesRoot = "../../../../docs/content/en/samples"
)

var (
	//nolint:golint,unused
	ignoredSamples = []string{"structureTest.yaml", "build.sh", "globalConfig.yaml", "Dockerfile.app", "Dockerfile.base"}
	//nolint:golint,unused
	ignoredExamples = []string{"docker-deploy", "react-reload-docker"}
)

// Test that every example can be parsed and produces a valid
// Skaffold configuration.
func TestParseExamples(t *testing.T) {
	// TODO: add examples for v2
	// t.SkipNow()()

	parseConfigFiles(t, "../../../../examples")
	parseConfigFiles(t, "../../../../integration/examples")
	parseConfigFiles(t, "../../../../integration/testdata/regressions")
}

// Samples are skaffold.yaml fragments that are used
// in the documentation.
func TestParseSamples(t *testing.T) {
	// TODO: add sample for v2
	// t.SkipNow()()

	paths, err := walk.From(samplesRoot).WhenIsFile().CollectPaths()
	if err != nil {
		t.Fatalf("unable to list samples in %q", samplesRoot)
	}

	if len(paths) == 0 {
		t.Fatalf("did not find sample files in %q", samplesRoot)
	}

	for _, path := range paths {
		name := filepath.Base(path)
		if stringslice.Contains(ignoredSamples, name) {
			continue
		}

		testutil.Run(t, name, func(t *testutil.T) {
			t.Logf("Checking %s...", path)
			buf, err := ioutil.ReadFile(path)
			t.CheckNoError(err)

			checkSkaffoldConfig(t, addHeader(buf))
		})
	}
}

func checkSkaffoldConfig(t *testutil.T, yaml []byte) {
	configFile := t.TempFile("skaffold.yaml", yaml)
	parsed, err := schema.ParseConfigAndUpgrade(configFile)
	t.CheckNoError(err)
	var cfgs parser.SkaffoldConfigSet
	for _, p := range parsed {
		cfg := &parser.SkaffoldConfigEntry{SkaffoldConfig: p.(*latest.SkaffoldConfig)}
		err = defaults.Set(cfg.SkaffoldConfig)
		defaults.SetDefaultDeployer(cfg.SkaffoldConfig)
		t.CheckNoError(err)
		cfgs = append(cfgs, cfg)
	}
	err = Process(cfgs, Options{CheckDeploySource: false})
	t.CheckNoError(err)
}

//nolint:golint,unused
func parseConfigFiles(t *testing.T, root string) {
	groupedPaths, err := walk.From(root).WhenHasName("skaffold.yaml").CollectPathsGrouped(1)
	if err != nil {
		t.Fatalf("unable to list skaffold configuration files in %q", root)
	}

	if len(groupedPaths) == 0 {
		t.Fatalf("did not find skaffold configuration files in %q", root)
	}
	for base, paths := range groupedPaths {
		name := filepath.Base(base)
		if stringslice.Contains(ignoredExamples, name) {
			continue
		}
		testutil.Run(t, name, func(t *testutil.T) {
			var data []string
			for _, path := range paths {
				buf, err := ioutil.ReadFile(path)
				t.CheckNoError(err)
				data = append(data, string(buf))
			}
			checkSkaffoldConfig(t, []byte(strings.Join(data, "\n---\n")))
		})
	}
}

func addHeader(buf []byte) []byte {
	if bytes.HasPrefix(buf, []byte("apiVersion:")) {
		return buf
	}
	return []byte(fmt.Sprintf("apiVersion: %s\nkind: Config\n%s", latest.Version, buf))
}
