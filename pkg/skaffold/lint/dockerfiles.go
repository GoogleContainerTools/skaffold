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
	"path/filepath"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/inspect"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// TODO(aaron-prindle) FIX, not correct

var DockerfileLinters = []Linter{
	&StringEqualsLinter{},
	&RegExpLinter{},
	&DockerfileCommandLinter{},
}

var DockerfileRules = []Rule{
	RuleIDToLintRuleMap[DOCKERFILE_COPY_DOT_OVER_100_FILES],
}

func GetDockerfilesList(ctx context.Context, out io.Writer, opts inspect.Options) (*DockerfileRulesList, error) {
	cfgs, err := inspect.GetConfigSet(ctx, config.SkaffoldOptions{
		ConfigurationFile:   opts.Filename,
		ConfigurationFilter: opts.Modules,
		RepoCacheDir:        opts.RepoCacheDir,
		Profiles:            opts.LintProfiles,
	})
	if err != nil {
		return nil, nil
	}

	l := &DockerfileRulesList{}
	seen := map[string]bool{}
	workdir, err := util.RealWorkDir()
	if err != nil {
		return nil, err
	}
	for _, c := range cfgs {
		for _, a := range c.Build.Artifacts {
			if a.DockerArtifact != nil {
				fp := filepath.Join(workdir, a.Workspace, a.DockerArtifact.DockerfilePath)
				if _, ok := seen[fp]; ok {
					continue
				}
				seen[fp] = true
				b, err := ioutil.ReadFile(fp)
				if err != nil {
					return nil, nil
				}
				dockerfile := ConfigFile{
					AbsPath: fp,
					RelPath: filepath.Join(a.Workspace, a.DockerArtifact.DockerfilePath),
					Text:    string(b),
				}
				results := []Result{}
				for _, r := range DockerfileLinters {
					recs, err := r.Lint(dockerfile, &DockerfileRules)
					if err != nil {
						return nil, err
					}
					results = append(results, *recs...)
				}
				l.DockerfileRules = append(l.DockerfileRules, results...)
			}
		}
	}
	return l, nil
}
