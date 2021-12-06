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
	v2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util/stringslice"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

var testK8sManifest = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: leeroy-web
  labels:
    app: leeroy-web
    app.kubernetes.io/managed-by: helm
spec:
  replicas: 1
  selector:
    matchLabels:
      app: leeroy-web
  template:
    metadata:
      labels:
        app: leeroy-web
    spec:
      containers:
        - name: leeroy-web
          image: leeroy-web
          ports:
            - containerPort: 8080
`

var invalidK8sManifest = `apiVersion: {{} `

func TestGetK8sManifestsLintResults(t *testing.T) {
	ruleIDToK8sManifestRule := map[RuleID]*Rule{}
	for i := range k8sManifestLintRules {
		ruleIDToK8sManifestRule[k8sManifestLintRules[i].RuleID] = &k8sManifestLintRules[i]
	}
	tests := []struct {
		shouldErr              bool
		k8sManifestIsNil       bool
		description            string
		k8sManifestText        string
		err                    error
		profiles               []string
		modules                []string
		rules                  []RuleID
		moduleAndSkaffoldYamls map[string]string
		expected               map[string]*[]Result
	}{
		{
			description:            "verify K8sManifestManagedByLabelInUse rule works as intended",
			rules:                  []RuleID{K8sManifestManagedByLabelInUse},
			moduleAndSkaffoldYamls: map[string]string{"cfg0": testSkaffoldYaml},
			modules:                []string{"cfg0"},
			k8sManifestText:        testK8sManifest,
			expected: map[string]*[]Result{
				"cfg0": {
					{
						Rule:   ruleIDToK8sManifestRule[K8sManifestManagedByLabelInUse],
						Line:   7,
						Column: 5,
						Explanation: `Found usage of label 'app.kubernetes.io/managed-by'.  skaffold overwrites the 'app.kubernetes.io/managed-by' ` +
							`field to 'app.kubernetes.io/managed-by: skaffold'. and as such is recommended to remove this label`,
					},
				},
			},
		},
		{
			rules:                  []RuleID{K8sManifestManagedByLabelInUse},
			description:            "invalid k8sManifest file",
			k8sManifestText:        invalidK8sManifest,
			moduleAndSkaffoldYamls: map[string]string{"cfg0": testSkaffoldYaml},
			shouldErr:              true,
		},
		{
			rules:                  []RuleID{},
			description:            "no k8sManifest file for skaffold.yaml",
			k8sManifestIsNil:       true,
			moduleAndSkaffoldYamls: map[string]string{"cfg0": testSkaffoldYaml},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			testRules := []Rule{}
			for _, ruleID := range test.rules {
				testRules = append(testRules, *(ruleIDToK8sManifestRule[ruleID]))
			}
			t.Override(k8sManifestRules, testRules)
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
				mp := filepath.Join(tmpdir, "deployment.yaml")
				err = ioutil.WriteFile(mp, []byte(test.k8sManifestText), 0644)
				if err != nil {
					t.Fatalf("error creating deployment.yaml %s: %v", mp, err)
				}
				if test.k8sManifestIsNil {
					configSet = append(configSet, &parser.SkaffoldConfigEntry{SkaffoldConfig: &v2.SkaffoldConfig{
						Metadata: v2.Metadata{Name: module},
						Pipeline: v2.Pipeline{},
					},
					})
				} else {
					configSet = append(configSet, &parser.SkaffoldConfigEntry{SkaffoldConfig: &v2.SkaffoldConfig{
						Metadata: v2.Metadata{Name: module},
						Pipeline: v2.Pipeline{Deploy: v2.DeployConfig{DeployType: v2.DeployType{KubectlDeploy: &v2.KubectlDeploy{Manifests: []string{mp}}}}},
					},
					})
				}

				// test overwrites file paths for expected K8sManifestRules as they are made dynamically
				results := test.expected[module]
				if results == nil {
					continue
				}
				for i := range *results {
					(*results)[i].AbsFilePath = mp
					(*results)[i].RelFilePath = mp
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
			results, err := GetK8sManifestsLintResults(context.Background(), Options{
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
