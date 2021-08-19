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
	"path/filepath"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/inspect"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var K8sYamlLinters = []FileLinter{
	&StringEqualsLinter{},
	&RegexpLinter{},
	&YamlFieldLinter{},
}

var K8sYamlLintRules = []LintRule{
	{
		// TODO(aaron-prindle) make a better recommendation regexp
		YamlFilter:   yaml.Lookup("metadata", "labels"),
		YamlValue:    "app.kubernetes.io/managed-by",
		LintRuleId:   K8S_YAML_PLACEHOLDER,
		LintRuleType: YamlFieldCheck,
		Explanation: fmt.Sprintln("Found usage of label 'app.kubernetes.io/managed-by'.  skaffold overwrites the 'app.kubernetes.io/managed-by' field to 'app.kubernetes.io/managed-by: skaffold'. " +
			"Remove this label or use the --dont-apply-managed-by-label flag to not have skaffold modify this label"),
	},
	// ideas
	// if: see multiple contexts for image builds
	//   &: those contexts have similar structure for deployments
	// then: suggest splitting app into modules
	//   have suggested module text?

	// if manifest prefix matches

	// if repo prefix used, recommend removing so default-repo works
}

func GetK8sYamlsList(ctx context.Context, out io.Writer, opts inspect.Options) (*K8sYamlLintRuleList, error) {
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
	l := &K8sYamlLintRuleList{K8sYamlLintRules: []MatchResult{}}
	for _, c := range cfgs {
		for _, pattern := range c.Deploy.KubectlDeploy.Manifests {
			// NOTE: pattern is a pattern that can have wildcards, eg: leeroy-app/kubernetes/*
			if util.IsURL(pattern) {
				logrus.Infof("skaffold lint found url manifest and is skipping lint rules for: %s", pattern)
				continue
			}
			// filepaths are all absolute from config parsing step via tags.MakeFilePathsAbsolute
			expanded, err := filepath.Glob(pattern)
			if err != nil {
				return nil, err
				// TODO(aaron-prindle) support returning multiple errors?
				// errs = append(errs, err)
			}

			for _, relPath := range expanded {
				b, err := ioutil.ReadFile(relPath)
				if err != nil {
					return nil, nil
				}
				k8syaml := ConfigFile{
					AbsPath: filepath.Join(workdir, relPath),
					RelPath: relPath,
					Text:    string(b),
				}
				mrs := []MatchResult{}
				for _, r := range K8sYamlLinters {
					recs, err := r.Lint(k8syaml, &K8sYamlLintRules)
					if err != nil {
						return nil, err
					}
					mrs = append(mrs, *recs...)
				}
				l.K8sYamlLintRules = append(l.K8sYamlLintRules, mrs...)
			}
		}
	}
	return l, nil
}
