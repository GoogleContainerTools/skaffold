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

	"github.com/moby/buildkit/frontend/dockerfile/command"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
)

// for testing
var readCopyCmdsFromDockerfile = docker.ReadCopyCmdsFromDockerfile

type DockerfileCommandLinter struct{}

func (*DockerfileCommandLinter) Lint(params InputParams, rules *[]Rule) (*[]Result, error) {
	results := &[]Result{}
	fromTos, err := readCopyCmdsFromDockerfile(context.TODO(), false, params.ConfigFile.AbsPath, params.WorkspacePath, map[string]*string{}, params.DockerConfig)
	if err != nil {
		return nil, err
	}
	for _, rule := range *rules {
		if rule.RuleType != DockerfileCommandLintRule {
			continue
		}
		var dockerCommandFilter DockerCommandFilter
		switch v := rule.Filter.(type) {
		case DockerCommandFilter:
			dockerCommandFilter = v
		default:
			return nil, fmt.Errorf("unknown filter type found for DockerfileCommandLinter lint rule: %v", rule)
		}
		// NOTE: ADD and COPY are both treated the same from the linter perspective - eg: if you have linter look at COPY src/dest it will also check ADD src/dest
		if dockerCommandFilter.DockerCommand != command.Copy && dockerCommandFilter.DockerCommand != command.Add {
			log.Entry(context.TODO()).Errorf("unsupported docker command found for DockerfileCommandLinter: %v", dockerCommandFilter.DockerCommand)
			return nil, fmt.Errorf("unsupported docker command found for DockerfileCommandLinter: %v", dockerCommandFilter.DockerCommand)
		}
		for _, fromTo := range fromTos {
			r, err := regexp.Compile(dockerCommandFilter.DockerCopySourceRegExp)
			if err != nil {
				return nil, err
			}
			if !r.MatchString(fromTo.From) {
				continue
			}
			log.Entry(context.TODO()).Infof("docker command 'copy' match found for source: %s\n", fromTo.From)
			// TODO(aaron-prindle) modify so that there are input and output params s.t. it is more obvious what fields need to be updated
			params.DockerCopyCommandInfo = fromTo
			appendRuleIfConditionsAndExplanationPopulationsSucceed(params, results, rule, fromTo.StartLine, 1, fromTo.EndLine, 0)
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
			// TODO(aaron-prindle) this type of message (last line of file) does not work well in an IDE via the LSP
			// consider not using this and pinning to somewhere in the yaml or using some type of different messaging (window/showMessage, etc.)
			line, col := getLastLineAndColOfFile(lintInputs.ConfigFile.Text)
			// TODO(aaron-prindle) verify this looks correct on an IDE
			appendRuleIfConditionsAndExplanationPopulationsSucceed(lintInputs, results, rule, line, col, line+1, 0)
			continue
		}
		if yamlFilter.FieldMatch != "" {
			mapnode := node.Field(yamlFilter.FieldMatch)
			if mapnode != nil {
				ks, err := mapnode.Key.String()
				if err != nil {
					return nil, err
				}
				lineLen, endCol := getLinesAndColsOfString(ks)
				appendRuleIfConditionsAndExplanationPopulationsSucceed(lintInputs, results, rule, mapnode.Key.YNode().Line, mapnode.Key.YNode().Column,
					mapnode.Key.YNode().Line+lineLen, endCol,
				)
			}
			continue
		}
		if node.YNode().Kind == yaml.ScalarNode {
			ns, err := node.String()
			if err != nil {
				return nil, err
			}
			lineLen, endCol := getLinesAndColsOfString(ns)
			appendRuleIfConditionsAndExplanationPopulationsSucceed(lintInputs, results, rule, node.Document().Line, node.Document().Column,
				node.Document().Line+lineLen, endCol,
			)
		}
		for _, n := range node.Content() {
			ns, err := node.String()
			if err != nil {
				return nil, err
			}
			lineLen, endCol := getLinesAndColsOfString(ns)
			appendRuleIfConditionsAndExplanationPopulationsSucceed(lintInputs, results, rule, n.Line, n.Column,
				n.Line+lineLen, endCol,
			)
		}
	}
	return results, nil
}

func appendRuleIfConditionsAndExplanationPopulationsSucceed(lintInputs InputParams, results *[]Result, rule Rule, startline, startcol, endline, endcol int) {
	if startline == endline {
		endline++ // this is done to highlight entire line when used w/ an IDE
	}

	for _, f := range rule.LintConditions {
		if !f(lintInputs) {
			// lint condition failed, no rule is trigggered
			return
		}
	}
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

	mr := Result{
		Rule:        &rule,
		Explanation: explanation,
		AbsFilePath: lintInputs.ConfigFile.AbsPath,
		RelFilePath: lintInputs.ConfigFile.RelPath,
		StartLine:   startline,
		EndLine:     endline,
		StartColumn: startcol,
		EndColumn:   endcol,
	}
	*results = append(*results, mr)
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

func getLinesAndColsOfString(str string) (int, int) {
	line := 0
	col := 0
	for i := range str {
		col++
		if str[i] == '\n' {
			line++
			col = 0
		}
	}
	return line, col
}
