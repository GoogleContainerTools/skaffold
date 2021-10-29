/*
Copyright 2021 The Skaffold Authors

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

package lint

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/parser"
	v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util/stringslice"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

var testSkaffoldYaml = `apiVersion: skaffold/v2beta21
kind: Config
build:
  artifacts:
    - image: leeroy-app
      context: leeroy-app
      requires:
        - image: base
          alias: BASE
deploy:
  kubectl:
    manifests:
      - leeroy-app/kubernetes/*
`

var testManifest = `apiVersion: v1
kind: Service
metadata:
  name: leeroy-app
  labels:
    app: leeroy-app
spec:
  clusterIP: None
  ports:
    - port: 50051
      name: leeroy-app
  selector:
    app: leeroy-app
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: leeroy-app
  labels:
    app: leeroy-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: leeroy-app
  template:
    metadata:
      labels:
        app: leeroy-app
    spec:
      containers:
      - name: leeroy-app
        image: leeroy-app
        ports:
        - containerPort: 50051
          name: http
`

var invalidSkaffoldYaml = `invalid{\{\Yaml}`

func TestGetSkaffoldYamlsLintResults(t *testing.T) {
	ruleIDToskaffoldYamlRule := map[RuleID]*Rule{}
	for i := range skaffoldYamlLintRules {
		ruleIDToskaffoldYamlRule[skaffoldYamlLintRules[i].RuleID] = &skaffoldYamlLintRules[i]
	}
	tests := []struct {
		description            string
		rules                  []RuleID
		moduleAndSkaffoldYamls map[string]string
		profiles               []string
		modules                []string
		shouldErr              bool
		err                    error
		expected               map[string]*[]Result
	}{
		{
			description:            "apply 1 skaffold lint rule for 2 skaffold yaml files",
			rules:                  []RuleID{SkaffoldYamlAPIVersionOutOfDate},
			moduleAndSkaffoldYamls: map[string]string{"cfg0": testSkaffoldYaml, "cfg1": testSkaffoldYaml},
			expected: map[string]*[]Result{
				"cfg0": {
					{
						Rule:        ruleIDToskaffoldYamlRule[SkaffoldYamlAPIVersionOutOfDate],
						Line:        1,
						Column:      13,
						Explanation: ruleIDToskaffoldYamlRule[SkaffoldYamlAPIVersionOutOfDate].ExplanationTemplate,
					},
				},
				"cfg1": {
					{
						Rule:        ruleIDToskaffoldYamlRule[SkaffoldYamlAPIVersionOutOfDate],
						Line:        1,
						Column:      13,
						Explanation: ruleIDToskaffoldYamlRule[SkaffoldYamlAPIVersionOutOfDate].ExplanationTemplate,
					},
				},
			},
		},
		{
			description:            "get all skaffold yaml lint rules for one module",
			rules:                  []RuleID{SkaffoldYamlAPIVersionOutOfDate},
			moduleAndSkaffoldYamls: map[string]string{"cfg0": testSkaffoldYaml, "cfg1": testSkaffoldYaml},
			modules:                []string{"cfg0"},
			expected: map[string]*[]Result{
				"cfg0": {
					{
						Rule:        ruleIDToskaffoldYamlRule[SkaffoldYamlAPIVersionOutOfDate],
						Line:        1,
						Column:      13,
						Explanation: ruleIDToskaffoldYamlRule[SkaffoldYamlAPIVersionOutOfDate].ExplanationTemplate,
					},
				},
			},
		},
		{
			description:            "verify SkaffoldYamlUseStaticPort rule works as intended",
			rules:                  []RuleID{SkaffoldYamlUseStaticPort},
			moduleAndSkaffoldYamls: map[string]string{"cfg0": testSkaffoldYaml},
			modules:                []string{"cfg0"},
			expected: map[string]*[]Result{
				"cfg0": {
					{
						Rule:   ruleIDToskaffoldYamlRule[SkaffoldYamlUseStaticPort],
						Line:   14,
						Column: 1,
						Explanation: "It is a skaffold best practice to specify a static port (vs skaffold dynamically choosing one) for port forwarding " +
							"container based resources skaffold deploys.  This is helpful because with this the local ports are predictable across dev sessions which " +
							" makes testing/debugging easier. It is recommended to add the following stanza at the end of your skaffold.yaml for each shown deployed resource:\n" +
							`portForward:
- resourceType: deployment
  resourceName: leeroy-app
  port: 50051
  localPort: 32581`,
					},
				},
			},
		},
		{
			rules:                  []RuleID{SkaffoldYamlAPIVersionOutOfDate},
			description:            "invalid skaffold yaml file",
			moduleAndSkaffoldYamls: map[string]string{"cfg0": invalidSkaffoldYaml},
			shouldErr:              true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			testRules := []Rule{}
			for _, ruleID := range test.rules {
				testRules = append(testRules, *(ruleIDToskaffoldYamlRule[ruleID]))
			}
			t.Override(skaffoldYamlRules, testRules)
			t.Override(&realWorkDir, func() (string, error) {
				return "", nil
			})
			tmpdir := t.TempDir()
			configSet := parser.SkaffoldConfigSet{}
			// iteration done to enforce result order
			for i := 0; i < len(test.moduleAndSkaffoldYamls); i++ {
				module := fmt.Sprintf("cfg%d", i)
				skaffoldyamlText := test.moduleAndSkaffoldYamls[module]
				fp := filepath.Join(tmpdir, fmt.Sprintf("%s.yaml", module))
				err := ioutil.WriteFile(fp, []byte(skaffoldyamlText), 0644)
				if err != nil {
					t.Fatalf("error creating skaffold.yaml file with name %s: %v", fp, err)
				}
				mp := filepath.Join(tmpdir, fmt.Sprintf("%s.yaml", "deployment"))
				err = ioutil.WriteFile(mp, []byte(testManifest), 0644)
				if err != nil {
					t.Fatalf("error creating deployment.yaml file with name %s: %v", fp, err)
				}
				configSet = append(configSet, &parser.SkaffoldConfigEntry{SkaffoldConfig: &v1.SkaffoldConfig{
					Metadata: v1.Metadata{Name: module},
					Pipeline: v1.Pipeline{Deploy: v1.DeployConfig{DeployType: v1.DeployType{KubectlDeploy: &v1.KubectlDeploy{Manifests: []string{
						mp,
					}}}}},
				},
					SourceFile: fp,
				})
				// test overwrites file paths for expected SkaffoldYamlRules as they are made dynamically
				results := test.expected[module]
				if results == nil {
					continue
				}
				for i := range *results {
					(*results)[i].AbsFilePath = configSet[len(configSet)-1].SourceFile
					(*results)[i].RelFilePath = configSet[len(configSet)-1].SourceFile
				}
			}
			t.Override(&getConfigSet, func(_ context.Context, opts config.SkaffoldOptions) (parser.SkaffoldConfigSet, error) {
				// mock profile activation
				var set parser.SkaffoldConfigSet
				for _, c := range configSet {
					if len(opts.ConfigurationFilter) > 0 && !stringslice.Contains(opts.ConfigurationFilter, c.Metadata.Name) {
						continue
					}
					for _, pName := range opts.Profiles {
						for _, profile := range c.Profiles {
							if profile.Name != pName {
								continue
							}
							c.Test = profile.Test
						}
					}
					set = append(set, c)
				}
				return set, test.err
			})
			results, err := GetSkaffoldYamlsLintResults(context.Background(), Options{
				OutFormat: "json", Modules: test.modules, Profiles: test.profiles})
			t.CheckError(test.shouldErr, err)
			if !test.shouldErr {
				expectedResults := &[]Result{}
				// this is done to enforce result order
				for i := 0; i < len(test.expected); i++ {
					*expectedResults = append(*expectedResults, *test.expected[fmt.Sprintf("cfg%d", i)]...)
					(*expectedResults)[0].Rule.ExplanationPopulator = nil
					(*expectedResults)[0].Rule.LintConditions = nil
				}
				if results == nil {
					t.CheckDeepEqual(expectedResults, results)
					return
				}
				for i := 0; i < len(*results); i++ {
					(*results)[i].Rule.ExplanationPopulator = nil
					(*results)[i].Rule.LintConditions = nil
				}
				t.CheckDeepEqual(expectedResults, results)
			}
		})
	}
}
