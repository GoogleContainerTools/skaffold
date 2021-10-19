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
	"regexp"

	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
)

type RegExpLinter struct{}

func (*RegExpLinter) Lint(cf ConfigFile, rules *[]Rule) (*[]Result, error) {
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
		matches := r.FindAllStringSubmatchIndex(cf.Text, -1)
		for _, m := range matches {
			log.Entry(context.TODO()).Infof("regexp match found for %s: %v\n", regexpFilter, m)
			// TODO(aaron-prindle) support matches with more than 2 values for m?
			line, col := convert1DFileIndexTo2D(cf.Text, m[0])
			mr := Result{
				Rule:        &rule,
				AbsFilePath: cf.AbsPath,
				RelFilePath: cf.RelPath,
				Line:        line,
				Column:      col,
			}
			*results = append(*results, mr)
		}
	}
	return results, nil
}

type YamlFieldLinter struct{}

func (*YamlFieldLinter) Lint(cf ConfigFile, rules *[]Rule) (*[]Result, error) {
	results := &[]Result{}
	obj, err := yaml.Parse(cf.Text)
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
			return nil, fmt.Errorf("unknown filter type found for YamlFieldLinter lint rule: %v", rule)
		}
		// TODO(aaron-prindle) - use Field property of kyaml where needed- https://github.com/kubernetes-sigs/kustomize/issues/4181
		node, err := obj.Pipe(yamlFilter.Filter)
		if err != nil {
			return nil, err
		}
		if (node == nil && !yamlFilter.InvertMatch) || node != nil && yamlFilter.InvertMatch {
			continue
		} else if node == nil && yamlFilter.InvertMatch {
			line, col := getLastLineAndColOfFile(cf.Text)
			*results = append(*results, yamlMatchToResult(rule, cf, line, col))
			continue
		}
		if node.YNode().Kind == yaml.ScalarNode {
			*results = append(*results, yamlMatchToResult(rule, cf, node.Document().Line, node.Document().Column-1))
		}
		for _, n := range node.Content() {
			*results = append(*results, yamlMatchToResult(rule, cf, n.Line, n.Column))
		}
	}
	return results, nil
}

func yamlMatchToResult(rule Rule, cf ConfigFile, line, col int) Result {
	return Result{
		Rule:        &rule,
		AbsFilePath: cf.AbsPath,
		RelFilePath: cf.RelPath,
		Line:        line,
		Column:      col,
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
	col := 0
	for i := 0; i < len(input); i++ {
		col++
		if input[i] == '\n' {
			line++
			col = 0
		}
	}
	return line, col
}
