/*
Copyright 2021 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or impliecf.
See the License for the specific language governing permissions and
limitations under the License.
*/

package lint

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	"github.com/moby/buildkit/frontend/dockerfile/command"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/moby/buildkit/frontend/dockerfile/shell"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type LintRuleId int

const (
	COPY_DOT_DOT_REGEXP       LintRuleId = iota // 000001
	DOCKERFILE_PLACEHOLDER                      // 000002
	SKAFFOLD_YAML_PLACEHOLDER                   // 000003
	K8S_YAML_PLACEHOLDER                        // 000004
)

func (a LintRuleId) String() string {
	// TODO(aaron-prindle) fix, hacky af
	return fmt.Sprintf("ID%06d", a+1)
	// return fmt.Sprintf("ID%06d", a+1)
	// return fmt.Sprintf("REGEXP%06d", a+1)
}

type LintRuleType int

const (
	StringEqualsCheck LintRuleType = iota
	RegexpMatchCheck
	YamlFieldCheck
	DockerfileCommandCheck
	BuildGraphConditionCheck
	MetricsConditionCheck
)

func (a LintRuleType) String() string {
	return [...]string{"StringEqualsCheck", "RegexpMatchCheck", "YamlFieldCheck", "DockerfileCommandCheck", "BuildGraphConditionCheck", "MetricsConditionCheck"}[a]
}

var ignoreLintRules = map[LintRuleId]bool{}

type LinterOpts struct{}

type FileLinter interface {
	// TODO(aaron-prindle) wait to put LinterOpts in
	// Lint(ConfigFile, []LintRule, LinterOpts) (*[]LintRule, error)
	Lint(ConfigFile, *[]LintRule) (*[]MatchResult, error)
}

type Graph struct{}

type GraphLinter interface {
	Lint(Graph, []LintRule) (*[]LintRule, error)
}

type StringEqualsLinter struct{}

func (*StringEqualsLinter) Lint(cf ConfigFile, rc *[]LintRule) (*[]MatchResult, error) {
	mrs := &[]MatchResult{}
	for _, rec := range *rc {
		if ignoreLintRules[rec.LintRuleId] || rec.LintRuleType != StringEqualsCheck {
			continue
		}
		// TODO(aaron-prindle) ignore case might make sense as an option here if speed is a concern later
		if idx := strings.Index(strings.ToUpper(cf.Text), strings.ToUpper(string(rec.MatchString))); idx != -1 {
			// if idx := strings.Index(cf.Text, strings.ToUpper(string(rec.MatchString))); idx != -1 {
			logrus.Infof("stringequals match found: [%d %d]\n", idx, idx+len(rec.MatchString))
			mr := MatchResult{
				LintRuleId:     rec.LintRuleId,
				AbsFilePath:    cf.AbsPath,
				RelFilePath:    cf.RelPath,
				TextStartIndex: idx,
				TextEndIndex:   idx + len(rec.MatchString),
				Explanation:    rec.Explanation,
				LintRuleType:   rec.LintRuleType,
			}
			*mrs = append(*mrs, mr)
		}
	}
	return mrs, nil
}

type RegexpLinter struct{}

func (*RegexpLinter) Lint(cf ConfigFile, rc *[]LintRule) (*[]MatchResult, error) {
	mrs := &[]MatchResult{}
	for _, rec := range *rc {
		if ignoreLintRules[rec.LintRuleId] || rec.LintRuleType != RegexpMatchCheck {
			continue
		}
		r, err := regexp.Compile(rec.Regexp)
		if err != nil {
			return nil, err
		}
		matches := r.FindAllStringSubmatchIndex(cf.Text, -1)
		for _, m := range matches {
			logrus.Infof("regexp match found for %s: %v\n", rec.Regexp, m)
			// TODO(aaron-prindle) support matches with more than 2 values for m?
			mr := MatchResult{
				LintRuleId:     rec.LintRuleId,
				AbsFilePath:    cf.AbsPath,
				RelFilePath:    cf.RelPath,
				TextStartIndex: m[0],
				TextEndIndex:   m[1],
				Explanation:    rec.Explanation,
				LintRuleType:   rec.LintRuleType,
			}
			*mrs = append(*mrs, mr)
		}
	}
	return mrs, nil
}

type DockerfileCommandLinter struct{}

func (*DockerfileCommandLinter) Lint(cf ConfigFile, rc *[]LintRule) (*[]MatchResult, error) {
	mrs := &[]MatchResult{}
	res, err := parser.Parse(strings.NewReader(cf.Text))
	if err != nil {
		return nil, fmt.Errorf("parsing dockerfile %q: %w", cf.AbsPath, err)
	}
	// TODO(aaron-prindle) add a parser/extractor for more/all dockerfile keywords
	copyCommands, err := extractCopyCommands(res.AST.Children)
	if err != nil {
		return nil, err
	}
	for _, rec := range *rc {
		if ignoreLintRules[rec.LintRuleId] || rec.LintRuleType != DockerfileCommandCheck {
			continue
		}
		// NOTE: ADD and COPY are both treated the same from the linter perspective - eg: if you have linter look at COPY src/dest it will also check ADD src/dest
		if rec.DockerCommand == command.Copy || rec.DockerCommand == command.Add {
			for _, cpyCmd := range copyCommands {
				for _, src := range cpyCmd.srcs {
					logrus.Infof("src: %s", src)
					if rec.DockerCopySource == src {
						logrus.Infof("docker command 'copy' match found for source: %s\n", rec.DockerCopySource)
						for _, f := range rec.LintConditions {
							if f(filepath.Join(filepath.Dir(cf.AbsPath), src)) {
								var line *int
								a := cpyCmd.startLine
								line = &a
								var column *int
								b := 0 // TODO(aaron-prindle) hack, perhaps use endLine somehow...
								column = &b
								mr := MatchResult{
									LintRuleId:   rec.LintRuleId,
									AbsFilePath:  cf.AbsPath,
									RelFilePath:  cf.RelPath,
									Line:         line,
									Column:       column,
									Explanation:  rec.Explanation,
									LintRuleType: rec.LintRuleType,
								}
								*mrs = append(*mrs, mr)
							}
						}

					}
				}
			}
		}

	}
	return mrs, nil
}

// copyCommand records a docker COPY/ADD command.
type copyCommand struct {
	// srcs records the source glob patterns.
	srcs []string
	// dest records the destination which may be a directory.
	dest string
	// destIsDir indicates if dest must be treated as directory.
	destIsDir bool
	// startLine is the starting line number of the copy command
	startLine int
	// endLine is the ending line number of the copy command
	endLine int
}

func extractCopyCommands(nodes []*parser.Node) ([]*copyCommand, error) {
	var copied []*copyCommand

	workdir := "/"
	envs := make([]string, 0)
	for _, node := range nodes {
		switch node.Value {
		case command.Add, command.Copy:
			cpCmd, err := readCopyCommand(node, envs, workdir)
			if err != nil {
				return nil, err
			}

			if cpCmd != nil && len(cpCmd.srcs) > 0 {
				copied = append(copied, cpCmd)
			}
		}
	}

	return copied, nil
}

func readCopyCommand(value *parser.Node, envs []string, workdir string) (*copyCommand, error) {
	// If the --from flag is provided, we are dealing with a multi-stage dockerfile
	// Adding a dependency from a different stage does not imply a source dependency
	if hasMultiStageFlag(value.Flags) {
		return nil, nil
	}

	var paths []string
	slex := shell.NewLex('\\')
	for value := value.Next; value != nil && !strings.HasPrefix(value.Value, "#"); value = value.Next {
		path, err := slex.ProcessWord(value.Value, envs)
		if err != nil {
			return nil, fmt.Errorf("expanding src: %w", err)
		}
		paths = append(paths, path)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("invalid dockerfile instruction: %q", value.Original)
	}

	// All paths are sources except the last one
	var srcs []string
	for _, src := range paths[0 : len(paths)-1] {
		if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
			log.Entry(context.TODO()).Debugln("Skipping watch on remote dependency", src)
			continue
		}

		srcs = append(srcs, src)
	}

	// Destination is last
	dest := paths[len(paths)-1]
	destIsDir := strings.HasSuffix(dest, "/") || path.Base(dest) == "." || path.Base(dest) == ".."
	dest = resolveDir(workdir, dest)

	return &copyCommand{
		srcs:      srcs,
		dest:      dest,
		destIsDir: destIsDir,
		// TODO(aaron-prindle) verify this is correct for 'values' with lots of paths... (might not be if nodes are not contained in the initial node start & end lines	)
		startLine: value.StartLine,
		endLine:   value.EndLine,
	}, nil
}

func hasMultiStageFlag(flags []string) bool {
	for _, f := range flags {
		if strings.HasPrefix(f, "--from=") {
			return true
		}
	}
	return false
}

// resolveDir determines the resulting directory as if a change-dir to targetDir was executed in cwd.
func resolveDir(cwd, targetDir string) string {
	if path.IsAbs(targetDir) {
		return path.Clean(targetDir)
	}
	return path.Clean(path.Join(cwd, targetDir))
}

type YamlFieldLinter struct{}

func (*YamlFieldLinter) Lint(cf ConfigFile, rc *[]LintRule) (*[]MatchResult, error) {
	mrs := &[]MatchResult{}
	obj, err := yaml.Parse(cf.Text)
	if err != nil {
		return nil, err
	}
	for _, rec := range *rc {
		if ignoreLintRules[rec.LintRuleId] || rec.LintRuleType != YamlFieldCheck {
			continue
		}
		node, err := obj.Pipe(rec.YamlFilter)
		if err != nil {
			return nil, err
		}
		if node == nil {
			continue
		}
		logrus.Infof("yaml field match found -  field: %s, value: %s\n", rec.YamlField, node.Document().Value)
		if node.Document().Value != "" {
			if node.Document().Value == rec.YamlValue {
				logrus.Infof("yaml field and value match found for %s\n", rec.YamlValue)
				var line *int
				a := node.Document().Line // TODO(aaron-prindle) not sure but the numbers here seem to be +1 than what I would expect...
				line = &a
				var column *int
				b := node.Document().Column // TODO(aaron-prindle) not sure but the numbers here seem to be +1 than what I would expect...
				column = &b
				mr := MatchResult{
					LintRuleId:   rec.LintRuleId,
					AbsFilePath:  cf.AbsPath,
					RelFilePath:  cf.RelPath,
					Line:         line,
					Column:       column,
					Explanation:  rec.Explanation,
					LintRuleType: rec.LintRuleType,
				}
				*mrs = append(*mrs, mr)
			}
		} else {
			for _, n := range node.Content() {
				if n.Value == rec.YamlValue {
					logrus.Infof("yaml field and value match found for %s\n", rec.YamlValue)
					var line *int
					a := n.Line // TODO(aaron-prindle) not sure but the numbers here seem to be +1 than what I would expect...
					line = &a
					var column *int
					b := n.Column // TODO(aaron-prindle) not sure but the numbers here seem to be +1 than what I would expect...
					column = &b
					mr := MatchResult{
						LintRuleId:   rec.LintRuleId,
						AbsFilePath:  cf.AbsPath,
						RelFilePath:  cf.RelPath,
						Line:         line,
						Column:       column,
						Explanation:  rec.Explanation,
						LintRuleType: rec.LintRuleType,
					}
					*mrs = append(*mrs, mr)
				}
			}
		}

	}
	return mrs, nil
}

type MatchResult struct {
	LintRuleId     LintRuleId
	AbsFilePath    string
	RelFilePath    string
	TextStartIndex int
	TextEndIndex   int
	// TODO(aaron-prindle) fix int pointer hack for seeing if using start/end index or line/cols
	Line   *int
	Column *int
	// TODO(aaron-prindle) make it so these are looked up by LintRuleId mapping, currently we just copy them from the LintRule directly when outputting
	Explanation  string
	LintRuleType LintRuleType
}

type LintRule struct {
	LintRuleId   LintRuleId
	LintRuleType LintRuleType
	Explanation  string
	MatchResult  MatchResult
	Severity     string // TODO(aaron-prindle) make this an enum and plumb all throughout
	//
	// TODO(aaron-prindle) split this out or refactor so that not all match types bundled here
	MatchString      string
	Regexp           string
	YamlField        string
	YamlFieldMatcher string
	YamlValue        string
	YamlFilter       yaml.Filter
	DockerCommand    string
	DockerCopyDest   string
	DockerCopySource string
	LintConditions   []func(string) bool
}

// type RegexpLintRule struct {
// 	Regexp string
// 	LintRule
// }

type ConfigFile struct {
	AbsPath string
	RelPath string
	Text    string
}

type DockerfileLintRulesList struct {
	DockerfileLintRules []MatchResult `json:"dockerfileLintRules"`
	// TODO(aaron-prindle) FIX - hack is below for keeping dockerfile text.  Shouldn't work like this ideally
	Dockerfiles []ConfigFile `json:"dockerfiles"`
}

type SkaffoldYamlLintRuleList struct {
	SkaffoldYamlLintRules []MatchResult `json:"skaffoldYamls"`
}

type K8sYamlLintRuleList struct {
	K8sYamlLintRules []MatchResult `json:"k8sYamls"`
}

type AllLintRuleList struct {
	SkaffoldYamlLintRuleList SkaffoldYamlLintRuleList `json:"skaffoldYamlLintRuleList"`
	DockerfileLintRuleList   DockerfileLintRulesList  `json:"dockerfileLintRuleList"`
	K8sYamlLintRuleList      K8sYamlLintRuleList      `json:"k8sYamlLintRuleList"`
}

type LintRuleList struct {
	LinterResultList []MatchResult `json:"recommendationList"`
}
