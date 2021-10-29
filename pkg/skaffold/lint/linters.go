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
	"context"
	"fmt"
	"regexp"
	"text/template"

	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
)

type RegExpLinter struct{}

func (*RegExpLinter) Lint(lintInputs InputParams, rules *[]Rule) (*[]Result, error) {
	results := &[]Result{}
	for _, rule := range *rules {
		if rule.RuleType != RegExpLintLintRule {
			continue
		}
		var regexpFilter string
		switch v := rule.Filter.(type) {
		case string:
			regexpFilter = v
		default:
			return nil, fmt.Errorf("unknown filter type found for RegExpLinter lint rule: %v", rule)
		}
		r, err := regexp.Compile(regexpFilter)
		if err != nil {
			return nil, err
		}
		matches := r.FindAllStringSubmatchIndex(lintInputs.ConfigFile.Text, -1)
		for _, m := range matches {
			log.Entry(context.TODO()).Infof("regexp match found for %s: %v\n", regexpFilter, m)
			// TODO(aaron-prindle) support matches with more than 2 values for m?
			line, col := convert1DFileIndexTo2D(lintInputs.ConfigFile.Text, m[0])
			appendRuleIfLintConditionsPass(lintInputs, results, rule, line, col)
		}
	}
	return results, nil
}

type YamlFieldLinter struct{}

func (*YamlFieldLinter) Lint(lintInputs InputParams, rules *[]Rule) (*[]Result, error) {
	results := &[]Result{}
	obj, err := yaml.Parse(lintInputs.ConfigFile.Text)
	if err != nil {
		return nil, err
	}
	for _, rule := range *rules {
		if rule.RuleType != YamlFieldLintRule {
			continue
		}
		var yamlFilter YamlFieldFilter
		switch v := rule.Filter.(type) {
		case YamlFieldFilter:
			yamlFilter = v
		default:
			return nil, fmt.Errorf("unknown filter type found for YamlFieldLinter lint rule %v with type: %s", rule, rule.RuleType)
		}
		// TODO(aaron-prindle) - use Field property of kyaml where needed- https://github.com/kubernetes-sigs/kustomize/issues/4181
		node, err := obj.Pipe(yamlFilter.Filter)
		if err != nil {
			return nil, err
		}
		if (node == nil && !yamlFilter.InvertMatch) || node != nil && yamlFilter.InvertMatch {
			continue
		} else if node == nil && yamlFilter.InvertMatch {
			line, col := getLastLineAndColOfFile(lintInputs.ConfigFile.Text)
			appendRuleIfLintConditionsPass(lintInputs, results, rule, line, col)
			continue
		}
		if yamlFilter.FieldOnly != "" {
			fieldNode := obj.Field(yamlFilter.FieldOnly)
			appendRuleIfLintConditionsPass(lintInputs, results, rule, fieldNode.Key.YNode().Line, fieldNode.Key.YNode().Column)
			continue
		}
		if node.YNode().Kind == yaml.ScalarNode {
			appendRuleIfLintConditionsPass(lintInputs, results, rule, node.Document().Line, node.Document().Column)
		}
		for _, n := range node.Content() {
			appendRuleIfLintConditionsPass(lintInputs, results, rule, n.Line, n.Column)
		}
	}
	return results, nil
}

func appendRuleIfLintConditionsPass(lintInputs InputParams, results *[]Result, rule Rule, line, col int) {
	allPassed := true
	explanation := rule.ExplanationTemplate
	if rule.ExplanationPopulator != nil {
		ei, err := rule.ExplanationPopulator(lintInputs)
		if err != nil {
			log.Entry(context.TODO()).Debugf("error attempting to populate explanation for rule %s with inputs %v: %v", rule.RuleID, lintInputs, err)
			return
		}
		var b bytes.Buffer
		tmpl, err := template.New("explanation").Parse(rule.ExplanationTemplate)
		if err != nil {
			log.Entry(context.TODO()).Debugf("error attempting to parse go template for rule %s with template %s: %v", rule.RuleID, rule.ExplanationTemplate, err)
			return
		}

		err = tmpl.Execute(&b, ei)
		if err != nil {
			log.Entry(context.TODO()).Debugf("error attempting to execute go template for rule %s with inputs %v: %v", rule.RuleID, ei, err)
			return
		}
		explanation = b.String()
	}

	for _, f := range rule.LintConditions {
		if !f(lintInputs) {
			allPassed = false
			break
		}
	}
	if allPassed {
		mr := Result{
			Rule:        &rule,
			Explanation: explanation,
			AbsFilePath: lintInputs.ConfigFile.AbsPath,
			RelFilePath: lintInputs.ConfigFile.RelPath,
			Line:        line,
			Column:      col,
		}
		*results = append(*results, mr)
	}
}

func convert1DFileIndexTo2D(input string, idx int) (int, int) {
	line := 1
	col := 0
	for i := 0; i < idx; i++ {
		col++
		if input[i] == '\n' {
			line++
			col = 0
		}
	}
	return line, col
}

func getLastLineAndColOfFile(input string) (int, int) {
	line := 1
	col := 1
	for i := 0; i < len(input); i++ {
		col++
		if input[i] == '\n' {
			line++
			col = 1
		}
	}
	return line, col
}
