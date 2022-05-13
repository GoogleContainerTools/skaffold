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

package parser

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/git"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	schemaUtil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/parser/configlocations"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

const (
	template = `
apiVersion: %s
kind: Config
metadata:
  name: %s
%s
build:
  artifacts:
  - image: image%s
profiles:
- name: pf0
  build:
    artifacts:
    - image: pf0image%s
- name: pf1
  build:
    artifacts:
    - image: pf1image%s
`
)

func TestGetAllConfigs(t *testing.T) {
	for _, test := range tcs {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir()
			for i, d := range test.documents {
				var cfgs []string
				for j, c := range d.configs {
					id := fmt.Sprintf("%d%d", i, j)
					s := fmt.Sprintf(template, latest.Version, c.name, c.requiresStanza, id, id, id)
					cfgs = append(cfgs, s)
				}
				tmpDir.Write(d.path, strings.Join(cfgs, "\n---\n"))
			}
			tmpDir.Chdir()
			var expected []schemaUtil.VersionedConfig
			if test.expected != nil {
				wd, _ := util.RealWorkDir()
				expected = test.expected(wd)
			}
			t.Override(&git.SyncRepo, func(ctx context.Context, g latest.GitInfo, _ config.SkaffoldOptions) (string, error) {
				return g.Repo, nil
			})
			cfgs, err := GetAllConfigs(context.Background(), config.SkaffoldOptions{
				Command:             "dev",
				ConfigurationFile:   test.documents[0].path,
				ConfigurationFilter: test.configFilter,
				Profiles:            test.profiles,
				PropagateProfiles:   test.applyProfilesRecursively,
				MakePathsAbsolute:   test.makePathsAbsolute,
			})
			if test.errCode == proto.StatusCode_OK {
				t.CheckDeepEqual(expected, cfgs)
			} else {
				var e sErrors.Error
				if errors.As(err, &e) {
					t.CheckDeepEqual(test.errCode, e.StatusCode())
				} else {
					t.Fail()
				}
			}
		})
	}
}

var testSkaffoldYaml = `apiVersion: skaffold/v3alpha1
kind: Config
build:
  artifacts:
    - image: app-0
deploy:
  kubectl:
    manifests:
      - manifests-0
profiles:
  - name: profile-0
    build:
      artifacts:
        - image: app-0-profile
          context: app-0-profile
    deploy:
      kubectl:
        manifests:
          - manifests-0-profile
    patches:
      - op: replace
        path: /build/artifacts/0
        value:
          image: app-0-patch
      - op: add
        path: /deploy/kubectl/manifests/1
        value: 'manifests-1'
  - name: profile-1
    build:
      artifacts:
        - image: app-1-profile
          context: app-1-profile
    deploy:
      kubectl:
        manifests:
          - manifests-1-profile
`

func TestConfigLocationsParse(t *testing.T) {
	tests := []struct {
		description      string
		skaffoldYamlText string
		profiles         []string
		missingNodeCount int
		expected         [][]kyaml.Filter
	}{
		{
			description:      "find all expected yaml nodes for input skaffold.yaml file",
			skaffoldYamlText: testSkaffoldYaml,
			profiles:         []string{},
			expected: [][]kyaml.Filter{
				{kyaml.Lookup("apiVersion")},
				{kyaml.Lookup("kind")},
				{kyaml.Lookup("build")},
				{kyaml.Lookup("build", "artifacts")},
				{kyaml.Lookup("deploy")},
				{kyaml.Lookup("deploy", "kubectl")},
				{kyaml.Lookup("deploy", "kubectl", "manifests")},
			},
		},
		{
			description:      "verify profile nodes not in yaml nodes when there is no profile",
			skaffoldYamlText: testSkaffoldYaml,
			profiles:         []string{},
			expected: [][]kyaml.Filter{
				{kyaml.Lookup("apiVersion")},
				{kyaml.Lookup("profiles"), kyaml.MatchElementList([]string{"name"}, []string{"profile-0"}), kyaml.Lookup("build"), kyaml.Lookup("artifacts")},
			},
			missingNodeCount: 1,
		},
		{
			description:      "find all expected yaml nodes for input skaffold.yaml file and input profile",
			skaffoldYamlText: testSkaffoldYaml,
			profiles:         []string{"profile-0"},
			expected: [][]kyaml.Filter{
				{kyaml.Lookup("apiVersion")},
				{kyaml.Lookup("kind")},
				{kyaml.Lookup("build")},
				{kyaml.Lookup("profiles"), kyaml.MatchElementList([]string{"name"}, []string{"profile-0"}), kyaml.Lookup("build"), kyaml.Lookup("artifacts")},
				{kyaml.Lookup("profiles"), kyaml.MatchElementList([]string{"name"}, []string{"profile-0"}),
					kyaml.Lookup("patches"), kyaml.GetElementByIndex(0), kyaml.Lookup("value"), kyaml.Lookup("image")},
				{kyaml.Lookup("profiles"), kyaml.MatchElementList([]string{"name"}, []string{"profile-0"}),
					kyaml.Lookup("patches"), kyaml.GetElementByIndex(1), kyaml.Lookup("value")},
				{kyaml.Lookup("deploy")},
				{kyaml.Lookup("profiles"), kyaml.MatchElementList([]string{"name"}, []string{"profile-0"}), kyaml.Lookup("deploy"), kyaml.Lookup("kubectl")},
				{kyaml.Lookup("profiles"), kyaml.MatchElementList([]string{"name"}, []string{"profile-0"}), kyaml.Lookup("deploy"), kyaml.Lookup("kubectl"), kyaml.Lookup("manifests")},
			},
		},
		{
			description:      "verify default nodes not in yaml nodes when there is an active profile overwriting the default node",
			skaffoldYamlText: testSkaffoldYaml,
			profiles:         []string{},
			expected: [][]kyaml.Filter{
				{kyaml.Lookup("apiVersion")},
				{kyaml.Lookup("build", "artifacts")},
			},
			missingNodeCount: 1,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			missingNodeCount := 0

			fp := t.TempFile("skaffoldyaml-", []byte(test.skaffoldYamlText))
			cfgs, err := GetConfigSet(context.TODO(), config.SkaffoldOptions{ConfigurationFile: fp, Profiles: test.profiles})
			if err != nil {
				t.Fatalf(err.Error())
			}
			root, err := kyaml.Parse(test.skaffoldYamlText)
			if err != nil {
				t.Fatalf(err.Error())
			}
			var seen bool
			for _, filters := range test.expected {
				seen = false
				expectedNode := root
				var err error
				for _, filter := range filters {
					expectedNode, err = expectedNode.Pipe(filter)
					if err != nil {
						t.Fatalf(err.Error())
					}
				}
				if expectedNode == nil {
					t.Errorf("test query led to nil node, should not be the case for kyaml filters: %v", filters)
				}
				for _, yamlInfos := range cfgs[0].YAMLInfos.GetYamlInfosCopy() {
					for _, v := range yamlInfos {
						if reflect.DeepEqual(expectedNode, v.RNode) {
							seen = true
						}
					}
				}
				if seen != true && test.missingNodeCount == 0 {
					str, _ := expectedNode.String()
					t.Errorf("unable to find expected yaml node text: %q in the generated yaml node map: %v", str, cfgs[0].YAMLInfos.GetYamlInfosCopy())
				}
				if seen != true && test.missingNodeCount > 0 {
					missingNodeCount++
					if missingNodeCount > test.missingNodeCount {
						t.Errorf("expected %d missing nodes in test, found %d missing nodes", test.missingNodeCount, missingNodeCount)
					}
				}
			}
		})
	}
}

func TestConfigLocationsLocate(t *testing.T) {
	tests := []struct {
		description      string
		skaffoldYamlText string
		profiles         []string
		expected         []configlocations.Location
	}{
		{
			description:      "verify location for SkaffoldConfig.Build.Artifacts[0] is as expected",
			skaffoldYamlText: testSkaffoldYaml,
			profiles:         []string{},
			expected: []configlocations.Location{
				{
					StartLine:   5,
					StartColumn: 14,
					EndLine:     6,
					EndColumn:   0,
				},
			},
		},
		{
			description:      "verify location for SkaffoldConfig.Build.Artifacts[0] is as expected with active profile with a patch",
			skaffoldYamlText: testSkaffoldYaml,
			profiles:         []string{"profile-0"},
			expected: []configlocations.Location{
				{
					StartLine:   24,
					StartColumn: 18,
					EndLine:     25,
					EndColumn:   0,
				},
			},
		},
		{
			description:      "verify location for SkaffoldConfig.Build.Artifacts[0] is as expected with active profile with no patch",
			skaffoldYamlText: testSkaffoldYaml,
			profiles:         []string{"profile-1"},
			expected: []configlocations.Location{
				{
					StartLine:   31,
					StartColumn: 18,
					EndLine:     32,
					EndColumn:   0,
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			fp := t.TempFile("skaffoldyaml-", []byte(test.skaffoldYamlText))
			cfgs, err := GetConfigSet(context.TODO(), config.SkaffoldOptions{ConfigurationFile: fp, Profiles: test.profiles})
			if err != nil {
				t.Fatalf(err.Error())
			}
			artifact0Location := cfgs.Locate(cfgs[0].SkaffoldConfig.Build.Artifacts[0])
			artifact0Location.SourceFile = ""
			t.CheckDeepEqual(&test.expected[0], artifact0Location)
		})
	}
}
