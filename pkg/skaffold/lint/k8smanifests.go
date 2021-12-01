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
	"io/ioutil"
	"path/filepath"

	"go.lsp.dev/protocol"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// for testing
var k8sManifestRules = &k8sManifestLintRules

var K8sManifestLinters = []Linter{
	&YamlFieldLinter{},
}

var k8sManifestLintRules = []Rule{
	{
		RuleID:   K8sManifestManagedByLabelInUse,
		RuleType: YamlFieldLintRule,
		Severity: protocol.DiagnosticSeverityWarning,
		Filter: YamlFieldFilter{
			Filter:     yaml.Lookup("metadata", "labels"),
			FieldMatch: "app.kubernetes.io/managed-by",
		},
		ExplanationTemplate: "Found usage of label 'app.kubernetes.io/managed-by'.  skaffold overwrites the 'app.kubernetes.io/managed-by' field to 'app.kubernetes.io/managed-by: skaffold'. " +
			"and as such is recommended to remove this label",
	},
}

func GetK8sManifestsLintResults(ctx context.Context, opts Options) (*[]Result, error) {
	cfgs, err := getConfigSet(ctx, config.SkaffoldOptions{
		ConfigurationFile:   opts.Filename,
		ConfigurationFilter: opts.Modules,
		RepoCacheDir:        opts.RepoCacheDir,
		Profiles:            opts.Profiles,
	})
	if err != nil {
		return nil, err
	}

	l := []Result{}
	workdir, err := realWorkDir()
	if err != nil {
		return nil, err
	}

	for _, c := range cfgs {
		if c.Deploy.KubectlDeploy == nil {
			continue
		}
		for _, pattern := range c.Deploy.KubectlDeploy.Manifests {
			// NOTE: pattern is a pattern that can have wildcards, eg: leeroy-app/kubernetes/*
			if util.IsURL(pattern) {
				log.Entry(ctx).Debugf("skaffold lint found url manifest and is skipping lint rules for: %s", pattern)
				continue
			}
			// filepaths are all absolute from config parsing step via tags.MakeFilePathsAbsolute
			expanded, err := filepath.Glob(pattern)
			if err != nil {
				return nil, err
			}

			for _, relPath := range expanded {
				b, err := ioutil.ReadFile(relPath)
				if err != nil {
					return nil, err
				}
				k8syaml := ConfigFile{
					AbsPath: filepath.Join(workdir, relPath),
					RelPath: relPath,
					Text:    string(b),
				}
				for _, r := range K8sManifestLinters {
					recs, err := r.Lint(InputParams{
						ConfigFile:     k8syaml,
						SkaffoldConfig: c,
					}, k8sManifestRules)
					if err != nil {
						return nil, err
					}
					l = append(l, *recs...)
				}
			}
		}
	}
	return &l, nil
}
