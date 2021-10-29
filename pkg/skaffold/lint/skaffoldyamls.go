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
	"strings"

	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/parser"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
)

// for testing
var getConfigSet = parser.GetConfigSet

var SkaffoldYamlLinters = []Linter{
	&RegExpLinter{},
	&YamlFieldLinter{},
}

// TODO(aaron-prindle) add highest priority lint rules in later PRs
var skaffoldYamlRules = []Rule{
	{
		Filter: YamlFieldFilter{
			Filter: yaml.FieldMatcher{Name: "apiVersion", StringRegexValue: fmt.Sprintf("[^%s]", version.Get().ConfigVersion)},
		},
		RuleID:   SkaffoldYamlAPIVersionOutOfDate,
		RuleType: YamlFieldLintRule,
		Explanation: fmt.Sprintf("Found 'apiVersion' field with value that is not the latest skaffold apiVersion. Modify the apiVersion to the latest version: `apiVersion: %s` "+
			"or run the 'skaffold fix' command to have skaffold upgrade this for you.", version.Get().ConfigVersion),
	},
}

func GetSkaffoldYamlsLintResults(ctx context.Context, opts Options) (*[]Result, error) {
	cfgs, err := getConfigSet(ctx, config.SkaffoldOptions{
		ConfigurationFile:   opts.Filename,
		ConfigurationFilter: opts.Modules,
		RepoCacheDir:        opts.RepoCacheDir,
		Profiles:            opts.Profiles,
	})
	if err != nil {
		return nil, err
	}

	workdir, err := realWorkDir()
	if err != nil {
		return nil, err
	}
	l := []Result{}
	for _, c := range cfgs {
		b, err := ioutil.ReadFile(c.SourceFile)
		if err != nil {
			return nil, err
		}
		skaffoldyaml := ConfigFile{
			AbsPath: c.SourceFile,
			RelPath: strings.TrimPrefix(c.SourceFile, workdir),
			Text:    string(b),
		}
		results := []Result{}
		for _, r := range SkaffoldYamlLinters {
			recs, err := r.Lint(skaffoldyaml, &skaffoldYamlRules)
			if err != nil {
				return nil, err
			}
			results = append(results, *recs...)
		}
		l = append(l, results...)
	}
	return &l, nil
}
