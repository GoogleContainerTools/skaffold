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
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"

	"go.lsp.dev/protocol"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestLintOutput(t *testing.T) {
	tests := []struct {
		description string
		outFormat   string
		results     interface{}
		text        string
		shouldErr   bool
		expected    string
	}{
		{
			description: "verify plain-text lint output is as expected",
			outFormat:   PlainTextOutput,
			results: []Result{
				{
					Rule: &Rule{
						RuleID:              DummyRuleIDForTesting,
						RuleType:            RegExpLintLintRule,
						ExplanationTemplate: "",
					},
					Explanation: "test explanation",
					AbsFilePath: "/abs/rel/path",
					RelFilePath: "rel/path",
					Line:        1,
					Column:      1,
				},
			},
			text:     "first column of this line should be flagged in the result [1,1]",
			expected: "rel/path:1:1: ID000000: RegExpLintLintRule: test explanation\nfirst column of this line should be flagged in the result [1,1]\n^\n",
		},
		{
			description: "verify json lint output is as expected",
			outFormat:   JSONOutput,
			results: []Result{
				{
					Rule: &Rule{
						RuleID:              DummyRuleIDForTesting,
						RuleType:            RegExpLintLintRule,
						ExplanationTemplate: "",
						Severity:            protocol.DiagnosticSeverityError,
					},
					AbsFilePath: "/abs/rel/path",
					RelFilePath: "rel/path",
					Line:        1,
					Column:      1,
					Explanation: "test explanation",
				},
			},
			text:     "first column of this line should be flagged in the result [1,1]",
			expected: `[{"Rule":{"RuleID":0,"RuleType":0,"ExplanationTemplate":"","Severity":1,"Filter":null},"AbsFilePath":%#v,"RelFilePath":"rel/path","Explanation":"test explanation","Line":1,"Column":1}]` + "\n",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&realWorkDir, func() (string, error) {
				return "", nil
			})
			resultList := test.results.([]Result)
			tmpdir := t.TempDir()
			f, err := ioutil.TempFile(tmpdir, "TestLintOutput-tmpfile")
			if err != nil {
				t.Fatalf("error creating dockerfile: %v", err)
			}
			_, err = f.Write([]byte(test.text))
			if err != nil {
				t.Fatalf("error writing dockerfile text to file: %v", err)
			}
			err = f.Close()
			if err != nil {
				t.Fatalf("error closing dockerfile handle: %v", err)
			}
			// TODO(aaron-prindle) make this work for len(results) > 1
			resultList[0].AbsFilePath = f.Name()

			var b bytes.Buffer
			formatter := OutputFormatter(&b, test.outFormat)
			err = formatter.Write(resultList)
			if err != nil {
				t.Fatalf("error occurred attempting to write output: %v", err)
			}
			if test.outFormat == PlainTextOutput {
				t.CheckDeepEqual(b.String(), test.expected)
			} else {
				t.CheckDeepEqual(b.String(), fmt.Sprintf(test.expected, f.Name()))
			}
		})
	}
}
