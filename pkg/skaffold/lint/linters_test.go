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
	"testing"

	"go.lsp.dev/protocol"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

var helloWorldTextFile = ConfigFile{
	Text:    "Hello World!",
	AbsPath: "/abs/rel/path",
	RelPath: "rel/path",
}

var k8sYamlFile = ConfigFile{
	Text: `apiVersion: apps/v1
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
`,
	AbsPath: "/abs/rel/path",
	RelPath: "rel/path",
}

func TestRegExpLinter(t *testing.T) {
	tests := []struct {
		description string
		configFile  ConfigFile
		rules       *[]Rule
		profiles    []string
		module      []string
		shouldErr   bool
		expected    *[]Result
	}{
		{
			description: "Test regexp linter",
			configFile:  helloWorldTextFile,
			rules: &[]Rule{
				{
					RuleID:      DummyRuleIDForTesting,
					RuleType:    RegExpLintLintRule,
					Explanation: "test explanation",
					Severity:    protocol.DiagnosticSeverityError,
					Filter:      "World",
				},
			},
			expected: &[]Result{
				{
					Rule: &Rule{
						RuleID:      DummyRuleIDForTesting,
						RuleType:    RegExpLintLintRule,
						Explanation: "test explanation",
						Severity:    protocol.DiagnosticSeverityError,
						Filter:      "World",
					},
					AbsFilePath: "/abs/rel/path",
					RelFilePath: "rel/path",
					Line:        1,
					Column:      6,
				},
			},
		},
		{
			description: "regexp linter w/ an different type lint rule",
			configFile: ConfigFile{
				Text:    "Hello World!",
				AbsPath: "/abs/rel/path",
				RelPath: "rel/path",
			},
			rules: &[]Rule{
				{
					RuleID:   DummyRuleIDForTesting,
					RuleType: YamlFieldLintRule,
				},
			},
			expected: &[]Result{},
		},
		{
			description: "regexp linter w/ an incorrect Filter type",
			configFile:  helloWorldTextFile,
			rules: &[]Rule{
				{
					RuleID:      DummyRuleIDForTesting,
					RuleType:    RegExpLintLintRule,
					Explanation: "test explanation",
					Severity:    protocol.DiagnosticSeverityError,
					Filter:      yaml.Get("incorrect filter type"),
				},
			},
			shouldErr: true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&realWorkDir, func() (string, error) {
				return "", nil
			})
			linter := &RegExpLinter{}
			recs, err := linter.Lint(test.configFile, test.rules)
			t.CheckError(test.shouldErr, err)
			t.CheckDeepEqual(test.expected, recs)
		})
	}
}

func TestYamlFieldLinter(t *testing.T) {
	tests := []struct {
		description string
		configFile  ConfigFile
		rules       *[]Rule
		profiles    []string
		module      []string
		shouldErr   bool
		expected    *[]Result
	}{
		{
			description: "valid yaml field lint rule w/ match",
			configFile:  k8sYamlFile,
			rules: &[]Rule{
				{
					RuleID:      DummyRuleIDForTesting,
					RuleType:    YamlFieldLintRule,
					Explanation: "test explanation",
					Severity:    protocol.DiagnosticSeverityError,
					Filter: YamlFieldFilter{
						Filter: yaml.FieldMatcher{Name: "apiVersion", StringValue: "apps/v1"},
					},
				},
			},
			expected: &[]Result{
				{
					Rule: &Rule{
						RuleID:      DummyRuleIDForTesting,
						RuleType:    YamlFieldLintRule,
						Explanation: "test explanation",
						Severity:    protocol.DiagnosticSeverityError,
						Filter: YamlFieldFilter{
							Filter: yaml.FieldMatcher{Name: "apiVersion", StringValue: "apps/v1"},
						},
					},
					AbsFilePath: "/abs/rel/path",
					RelFilePath: "rel/path",
					Line:        1,
					Column:      12,
				},
			},
		},
		{
			description: "valid yaml field lint rule with no match",
			configFile:  k8sYamlFile,
			rules: &[]Rule{
				{
					RuleID:      DummyRuleIDForTesting,
					RuleType:    YamlFieldLintRule,
					Explanation: "test explanation",
					Severity:    protocol.DiagnosticSeverityError,
					Filter: YamlFieldFilter{
						Filter: yaml.FieldMatcher{Name: "missingField"},
					},
				},
			},
			expected: &[]Result{},
		},
		{
			description: "valid yaml field lint rule match using InvertMatch",
			configFile:  k8sYamlFile,
			rules: &[]Rule{
				{
					RuleID:      DummyRuleIDForTesting,
					RuleType:    YamlFieldLintRule,
					Explanation: "test explanation",
					Severity:    protocol.DiagnosticSeverityError,
					Filter: YamlFieldFilter{
						Filter:      yaml.FieldMatcher{Name: "missingField"},
						InvertMatch: true,
					},
				},
			},
			expected: &[]Result{
				{
					Rule: &Rule{
						RuleID:      DummyRuleIDForTesting,
						RuleType:    YamlFieldLintRule,
						Explanation: "test explanation",
						Severity:    protocol.DiagnosticSeverityError,
						Filter: YamlFieldFilter{
							Filter:      yaml.FieldMatcher{Name: "missingField"},
							InvertMatch: true,
						},
					},
					AbsFilePath: "/abs/rel/path",
					RelFilePath: "rel/path",
					Line:        23,
					Column:      0,
				},
			},
		},
		{
			description: "yaml field linter w/ an different type lint rule",
			configFile:  k8sYamlFile,
			rules: &[]Rule{
				{
					RuleID:   DummyRuleIDForTesting,
					RuleType: RegExpLintLintRule,
				},
			},
			expected: &[]Result{},
		},
		{
			description: "yaml field command linter w/ an incorrect Filter type",
			configFile:  k8sYamlFile,
			rules: &[]Rule{
				{
					RuleID:      DummyRuleIDForTesting,
					RuleType:    YamlFieldLintRule,
					Explanation: "test explanation",
					Severity:    protocol.DiagnosticSeverityError,
					Filter:      "incorrect filter type",
				},
			},
			shouldErr: true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&realWorkDir, func() (string, error) {
				return "", nil
			})
			linter := &YamlFieldLinter{}
			recs, err := linter.Lint(test.configFile, test.rules)
			t.CheckError(test.shouldErr, err)
			t.CheckDeepEqual(test.expected, recs)
		})
	}
}
