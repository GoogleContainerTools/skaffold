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

	"github.com/moby/buildkit/frontend/dockerfile/command"
	"go.lsp.dev/protocol"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
)

// for testing
var getDockerDependenciesForEachFromTo = docker.GetDependenciesByDockerCopyFromTo
var dockerfileRules = &dockerfileLintRules

var DockerfileLinters = []Linter{
	&RegExpLinter{},
	&DockerfileCommandLinter{},
}

var dockerfileLintRules = []Rule{
	{
		RuleID:   DockerfileCopyOver1000Files,
		RuleType: DockerfileCommandLintRule,
		Severity: protocol.DiagnosticSeverityWarning,
		Filter: DockerCommandFilter{
			DockerCommand:          command.Copy,
			DockerCopySourceRegExp: `.*`,
		},
		ExplanationTemplate: `Found docker 'COPY' command where the source directory "{{index .FieldMap "src"}}" has over 1000 files.  This has the potential to dramatically slow 'skaffold dev' down ` +
			`as skaffold watches all sources files referenced in dockerfile COPY directives for changes. ` +
			`If you notice skaffold rebuilding images unnecessarily when non-image-critical files are ` +
			`modified, consider changing this to 'COPY $REQUIRED_SOURCE_FILE(s) {{index .FieldMap "dest"}}' for each required source file instead of ` +
			`or adding a .dockerignore file (https://docs.docker.com/engine/reference/builder/#dockerignore-file) ignoring non-image-critical files.  skaffold respects files ignored via the .dockerignore`,
		ExplanationPopulator: func(params InputParams) (explanationInfo, error) {
			return explanationInfo{
				FieldMap: map[string]interface{}{
					"src":  params.DockerCopyCommandInfo.From,
					"dest": params.DockerCopyCommandInfo.To,
				},
			}, nil
		},
		LintConditions: []func(InputParams) bool{func(params InputParams) bool {
			files := 0
			for range params.DockerfileToFromToToDeps[params.ConfigFile.AbsPath][params.DockerCopyCommandInfo.String()] {
				files++
			}
			return files > 1000
		}},
	},
	{
		RuleID:   DockerfileCopyContainsGitDir,
		RuleType: DockerfileCommandLintRule,
		Severity: protocol.DiagnosticSeverityWarning,
		Filter: DockerCommandFilter{
			DockerCommand:          command.Copy,
			DockerCopySourceRegExp: `.*`,
		},
		// TODO(aaron-prindle) suggest a full .dockerignore sample - .dockerignore:**/.git
		ExplanationTemplate: `Found docker 'COPY' command where the source directory "{{index .FieldMap "src"}}" contains a '.git' directory at {{index .FieldMap "gitDirectoryAbsPath"}}.  This has the potential to dramatically slow 'skaffold dev' down ` +
			`as skaffold will watch all of the files in the .git directory as skaffold watches all sources files referenced in dockerfile COPY directives for changes. ` +
			`skaffold will likely rebuild images unnecessarily when non-image-critical files are ` +
			`modified during any git related operation. Consider adding a .dockerignore file (https://docs.docker.com/engine/reference/builder/#dockerignore-file) ignoring the '.git' directory. skaffold respects files ignored via the .dockerignore`,
		ExplanationPopulator: func(params InputParams) (explanationInfo, error) {
			var gitDirectoryAbsPath string
			for _, dep := range params.DockerfileToFromToToDeps[params.ConfigFile.AbsPath][params.DockerCopyCommandInfo.String()] {
				if filepath.Dir(dep) == ".git" {
					gitDirectoryAbsPath = filepath.Join(params.WorkspacePath, filepath.Dir(dep))
					break
				}
			}
			return explanationInfo{
				FieldMap: map[string]interface{}{
					"src":                 params.DockerCopyCommandInfo.From,
					"gitDirectoryAbsPath": gitDirectoryAbsPath,
				},
			}, nil
		},

		// TODO(aaron-prindle) currently the LintCondition runs w/ deps that map to a dockerfile and not a specific COPY command.  Can make certain rules infeasible
		LintConditions: []func(InputParams) bool{func(params InputParams) bool {
			for _, dep := range params.DockerfileToFromToToDeps[params.ConfigFile.AbsPath][params.DockerCopyCommandInfo.String()] {
				if filepath.Dir(dep) == ".git" {
					return true
				}
			}
			return false
		}},
	},
}

func GetDockerfilesLintResults(ctx context.Context, opts Options, dockerCfg docker.Config) (*[]Result, error) {
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
	seen := map[string]bool{}
	dockerfileToFromToToDepMap := map[string]map[string][]string{}
	workdir, err := realWorkDir()
	if err != nil {
		return nil, err
	}

	for _, c := range cfgs {
		for _, a := range c.Build.Artifacts {
			if a.DockerArtifact != nil {
				// TODO(aaron-prindle) HACK - multi-module configs use abs path for a.Workspace vs single module which has rel path
				// see if there is a built-in/better way of handling this.  This is currently working for multi-module
				ws := a.Workspace
				if !filepath.IsAbs(ws) {
					ws = filepath.Join(workdir, a.Workspace)
				}
				fp := filepath.Join(ws, a.DockerArtifact.DockerfilePath)
				if _, ok := seen[fp]; ok {
					continue
				}
				seen[fp] = true
				b, err := ioutil.ReadFile(fp)
				if err != nil {
					return nil, err
				}
				dockerfile := ConfigFile{
					AbsPath: fp,
					RelPath: filepath.Join(a.Workspace, a.DockerArtifact.DockerfilePath),
					Text:    string(b),
				}
				// TODO(aaron-prindle) currently this dep map is computed twice; here and in skaffoldyamls.go, make a singleton/share-the-info
				// TODO(aaron-prindle) currently copy commands are parsed twice; here and in linters.go
				fromToToDepMap, err := getDockerDependenciesForEachFromTo(context.TODO(),
					docker.NewBuildConfig(ws, a.ImageName, fp, map[string]*string{}), nil)
				if err != nil {
					return nil, err
				}
				dockerfileToFromToToDepMap[fp] = fromToToDepMap
				for _, r := range DockerfileLinters {
					recs, err := r.Lint(InputParams{
						ConfigFile:               dockerfile,
						SkaffoldConfig:           c,
						DockerfileToFromToToDeps: dockerfileToFromToToDepMap,
						WorkspacePath:            ws,
						DockerConfig:             dockerCfg,
					}, dockerfileRules)
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
