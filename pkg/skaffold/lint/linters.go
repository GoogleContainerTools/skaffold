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

type StringEqualsLinter struct{}

func (*StringEqualsLinter) Lint(cf ConfigFile, rules *[]Rule) (*[]Result, error) {
	results := &[]Result{}
	for _, rule := range *rules {
		if ignoreRules[rule.RuleID] || rule.RuleType != StringEqualsLintRule {
			continue
		}
		// TODO(aaron-prindle) ignore case might make sense as an option here if speed is a concern later
		if idx := strings.Index(strings.ToUpper(cf.Text), strings.ToUpper(rule.LintString)); idx != -1 {
			// if idx := strings.Index(cf.Text, strings.ToUpper(string(rule.LintString))); idx != -1 {
			logrus.Infof("stringequals match found: [%d %d]\n", idx, idx+len(rule.LintString))
			line, col := convert1DFileIndexTo2D(cf.Text, idx)
			mr := Result{
				RuleID:      rule.RuleID,
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

type RegExpLinter struct{}

func (*RegExpLinter) Lint(cf ConfigFile, rules *[]Rule) (*[]Result, error) {
	results := &[]Result{}
	for _, rule := range *rules {
		if ignoreRules[rule.RuleID] || rule.RuleType != RegExpLintLintRule {
			continue
		}
		r, err := regexp.Compile(rule.RegExp)
		if err != nil {
			return nil, err
		}
		matches := r.FindAllStringSubmatchIndex(cf.Text, -1)
		for _, m := range matches {
			logrus.Infof("regexp match found for %s: %v\n", rule.RegExp, m)
			// TODO(aaron-prindle) support matches with more than 2 values for m?
			line, col := convert1DFileIndexTo2D(cf.Text, m[0])
			mr := Result{
				RuleID:      rule.RuleID,
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

type DockerfileCommandLinter struct{}

func (*DockerfileCommandLinter) Lint(cf ConfigFile, rules *[]Rule) (*[]Result, error) {
	results := &[]Result{}
	res, err := parser.Parse(strings.NewReader(cf.Text))
	if err != nil {
		return nil, fmt.Errorf("parsing dockerfile %q: %w", cf.AbsPath, err)
	}
	// TODO(aaron-prindle) add a parser/extractor for more/all dockerfile keywords
	copyCommands, err := extractCopyCommands(res.AST.Children)
	if err != nil {
		return nil, err
	}
	for _, rule := range *rules {
		if ignoreRules[rule.RuleID] || rule.RuleType != DockerfileCommandLintRule {
			continue
		}
		// NOTE: ADD and COPY are both treated the same from the linter perspective - eg: if you have linter look at COPY src/dest it will also check ADD src/dest
		if rule.DockerCommand != command.Copy && rule.DockerCommand != command.Add {
			logrus.Errorf("unsupported docker command found for DockerfileCommandLinter: %v", rule.DockerCommand)
			continue
		}
		for _, cpyCmd := range copyCommands {
			for _, src := range cpyCmd.srcs {
				if rule.DockerCopySource != src {
					continue
				}
				logrus.Infof("docker command 'copy' match found for source: %s\n", rule.DockerCopySource)
				allPassed := true
				for _, f := range rule.LintConditions {
					if !f(filepath.Join(filepath.Dir(cf.AbsPath), src)) {
						allPassed = false
						break
					}
				}
				if allPassed {
					mr := Result{
						RuleID:      rule.RuleID,
						AbsFilePath: cf.AbsPath,
						RelFilePath: cf.RelPath,
						Line:        cpyCmd.startLine,
						Column:      0, // column info not accessible via buildkit parse library, use 0 index as a stub when displaying the flagged line
					}
					*results = append(*results, mr)
				}
			}
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
		if ignoreRules[rule.RuleID] || rule.RuleType != YamlFieldLintRule {
			continue
		}
		node, err := obj.Pipe(rule.YamlFilter)
		if err != nil {
			return nil, err
		}
		if node == nil {
			continue
		}
		// TODO(aaron-prindle) perhaps handle the below case to be an actual regexp for .*
		if rule.YamlValue == ".*" { // only field existence matters
			*results = append(*results, yamlMatchToResult(rule, cf, node.Document().Line-1, 0))
		}
		// case occures when value itself is key/value mapping eg: apiVersion
		if node.Document().Value != "" {
			if node.Document().Value == rule.YamlValue {
				logrus.Infof("yaml field and value match found for %s\n", rule.YamlValue)
				*results = append(*results, yamlMatchToResult(rule, cf, node.Document().Line, node.Document().Column-1))

			}
		} else { // case occurs when value is a nested object, for example metadata.labels
			for _, n := range node.Content() {
				if n.Value == rule.YamlValue {
					logrus.Infof("yaml field and value match found for %s\n", rule.YamlValue)

					*results = append(*results, yamlMatchToResult(rule, cf, node.Document().Line, node.Document().Column))
				}
			}
		}
	}
	return results, nil
}

func yamlMatchToResult(rule Rule, cf ConfigFile, line, col int) Result {
	return Result{
		RuleID:      rule.RuleID,
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
