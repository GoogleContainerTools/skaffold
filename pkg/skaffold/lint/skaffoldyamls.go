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
	"io"
	"io/ioutil"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/inspect"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

var SkaffoldYamlLinters = []Linter{
	&StringEqualsLinter{},
	&RegExpLinter{},
	&YamlFieldLinter{},
}

var SkaffoldYamlRules = []Rule{
	RuleIDToLintRuleMap[SKAFFOLD_YAML_API_VERSION_OUT_OF_DATE],
	RuleIDToLintRuleMap[SKAFFOLD_YAML_REPO_IS_HARD_CODED],
	RuleIDToLintRuleMap[SKAFFOLD_YAML_SUGGEST_INFER_STANZA],
	// ideas
	// if: see multiple contexts for image builds
	//   &: those contexts have similar structure for deployments
	// then: suggest splitting app into modules
	//   have suggested module text?

	// if manifest prefix matches

	// if repo prefix used, recommend removing so default-repo works
}

func GetSkaffoldYamlsList(ctx context.Context, out io.Writer, opts inspect.Options) (*SkaffoldYamlRuleList, error) {
	cfgs, err := inspect.GetConfigSet(ctx, config.SkaffoldOptions{
		ConfigurationFile:   opts.Filename,
		ConfigurationFilter: opts.Modules,
		RepoCacheDir:        opts.RepoCacheDir,
		Profiles:            opts.LintProfiles,
	})
	if err != nil {
		return nil, err
	}

	workdir, err := util.RealWorkDir()
	if err != nil {
		return nil, err
	}
	l := &SkaffoldYamlRuleList{SkaffoldYamlRules: []Result{}}
	for _, c := range cfgs {
		b, err := ioutil.ReadFile(c.SourceFile)
		if err != nil {
			return nil, nil
		}
		skaffoldyaml := ConfigFile{
			AbsPath: c.SourceFile,
			RelPath: strings.TrimPrefix(c.SourceFile, workdir),
			Text:    string(b),
		}
		results := []Result{}
		for _, r := range SkaffoldYamlLinters {
			recs, err := r.Lint(skaffoldyaml, &SkaffoldYamlRules)
			if err != nil {
				return nil, err
			}
			results = append(results, *recs...)
		}
		l.SkaffoldYamlRules = append(l.SkaffoldYamlRules, results...)
	}
	return l, nil
}
