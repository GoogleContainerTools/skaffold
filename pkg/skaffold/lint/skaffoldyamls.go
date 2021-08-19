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
	"io"
	"io/ioutil"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/inspect"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var SkaffoldYamlLinters = []FileLinter{
	&StringEqualsLinter{},
	&RegexpLinter{},
	&YamlFieldLinter{},
}

var SkaffoldYamlLintRules = []LintRule{
	{
		// TODO(aaron-prindle) make a better recommendation regexp
		Regexp:       "gcr.io/|docker.io/|amazonaws.com/",
		LintRuleId:   SKAFFOLD_YAML_PLACEHOLDER,
		LintRuleType: RegexpMatchCheck,
		Explanation: "Found image registry prefix on an image skaffold manages.  This is not recommended as it reduces the usability of skaffold project. " +
			"The image registry name should be removed and an image registry should be added programatically via skaffold, for example with the --default-repo flag",
	},

	{
		// TODO(aaron-prindle) check to see how kyaml supports regexp and how to best plumb that through
		YamlFilter: yaml.Get("apiVersion"),
		YamlValue:  "skaffold/v2beta21",
		// YamlValue:          version.Get().ConfigVersion,
		LintRuleId:   SKAFFOLD_YAML_PLACEHOLDER,
		LintRuleType: YamlFieldCheck,
		Explanation:  fmt.Sprintf("Found 'apiVersion' field with value that is not the latest skaffold apiVersion. Modify the apiVersion to the latest supported version: `apiVersion: %s`", version.Get().ConfigVersion),
	},

	// ideas
	// if: see multiple contexts for image builds
	//   &: those contexts have similar structure for deployments
	// then: suggest splitting app into modules
	//   have suggested module text?

	// if manifest prefix matches

	// if repo prefix used, recommend removing so default-repo works
}

func GetSkaffoldYamlsList(ctx context.Context, out io.Writer, opts inspect.Options) (*SkaffoldYamlLintRuleList, error) {
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
	l := &SkaffoldYamlLintRuleList{SkaffoldYamlLintRules: []MatchResult{}}
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
		mrs := []MatchResult{}
		for _, r := range SkaffoldYamlLinters {
			recs, err := r.Lint(skaffoldyaml, &SkaffoldYamlLintRules)
			if err != nil {
				return nil, err
			}
			mrs = append(mrs, *recs...)
		}
		l.SkaffoldYamlLintRules = append(l.SkaffoldYamlLintRules, mrs...)
	}
	return l, nil
}
