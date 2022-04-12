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
	"io/ioutil"
	"path/filepath"
	"testing"
	"text/template"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/parser"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util/stringslice"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

var testDockerfile = `ARG BASE
FROM golang:1.15 as builder{{range .}}
COPY {{.From}} {{.To}}{{end}}
COPY local.txt /container-dir
ARG SKAFFOLD_GO_GCFLAGS
RUN go build -gcflags="${SKAFFOLD_GO_GCFLAGS}" -o /app .

FROM $BASE
COPY --from=builder /app .
`

func TestGetDockerfilesLintResults(t *testing.T) {
	ruleIDToDockerfileRule := map[RuleID]*Rule{}
	for i := range dockerfileLintRules {
		ruleIDToDockerfileRule[dockerfileLintRules[i].RuleID] = &dockerfileLintRules[i]
	}
	tests := []struct {
		description            string
		rules                  []RuleID
		moduleAndSkaffoldYamls map[string]string
		profiles               []string
		modules                []string
		dockerFromTo           []docker.FromTo
		shouldErr              bool
		err                    error
		expected               map[string]*[]Result
	}{
		{
			description:            "verify DockerfileCopyOver1000Files rule works as intended",
			rules:                  []RuleID{DockerfileCopyOver1000Files},
			moduleAndSkaffoldYamls: map[string]string{"cfg0": testSkaffoldYaml},
			modules:                []string{"cfg0"},
			dockerFromTo: []docker.FromTo{
				{
					From:      ".",
					To:        "/",
					ToIsDir:   true,
					StartLine: 3,
					EndLine:   3,
				},
				{
					From:      "local.txt",
					To:        "/container-dir",
					ToIsDir:   true,
					StartLine: 3,
					EndLine:   3,
				},
			},
			expected: map[string]*[]Result{
				"cfg0": {
					{
						Rule:        ruleIDToDockerfileRule[DockerfileCopyOver1000Files],
						StartLine:   3,
						EndLine:     4,
						StartColumn: 1,
						EndColumn:   0,
						Explanation: `Found docker 'COPY' command where the source directory "." has over 1000 files.  This has the potential ` +
							`to dramatically slow 'skaffold dev' down as skaffold watches all sources files referenced in dockerfile COPY directives ` +
							`for changes. If you notice skaffold rebuilding images unnecessarily when non-image-critical files are modified, consider ` +
							`changing this to 'COPY $REQUIRED_SOURCE_FILE(s) /' for each required source file instead of or adding a .dockerignore file ` +
							`(https://docs.docker.com/engine/reference/builder/#dockerignore-file) ignoring non-image-critical files.  skaffold respects ` +
							`files ignored via the .dockerignore`,
					},
				},
			},
		},
		{
			description:            "verify DockerfileCopyContainsGitDir rule works as intended",
			rules:                  []RuleID{DockerfileCopyContainsGitDir},
			moduleAndSkaffoldYamls: map[string]string{"cfg0": testSkaffoldYaml},
			modules:                []string{"cfg0"},
			dockerFromTo: []docker.FromTo{
				{
					From:      ".",
					To:        "/",
					ToIsDir:   true,
					StartLine: 3,
					EndLine:   3,
				},
				{
					From:      "local.txt",
					To:        "/container-dir",
					ToIsDir:   true,
					StartLine: 3,
					EndLine:   3,
				},
			},
			expected: map[string]*[]Result{
				"cfg0": {
					{
						Rule:        ruleIDToDockerfileRule[DockerfileCopyContainsGitDir],
						StartLine:   3,
						EndLine:     4,
						StartColumn: 1,
						EndColumn:   0,
						Explanation: `Found docker 'COPY' command where the source directory "." contains a '.git' directory at .git.  This has the potential ` +
							`to dramatically slow 'skaffold dev' down as skaffold will watch all of the files in the .git directory as skaffold watches all sources ` +
							`files referenced in dockerfile COPY directives for changes. skaffold will likely rebuild images unnecessarily when non-image-critical ` +
							`files are modified during any git related operation. Consider adding a .dockerignore file ` +
							`(https://docs.docker.com/engine/reference/builder/#dockerignore-file) ignoring the '.git' directory. skaffold respects files ignored ` +
							`via the .dockerignore`,
					},
				},
			},
		},
		{
			rules:       []RuleID{DockerfileCopyContainsGitDir},
			description: "invalid dockerfile file",
			dockerFromTo: []docker.FromTo{
				{
					From: "",
					To:   "",
				},
			},
			moduleAndSkaffoldYamls: map[string]string{"cfg0": testSkaffoldYaml},
			shouldErr:              true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			testRules := []Rule{}
			for _, ruleID := range test.rules {
				testRules = append(testRules, *(ruleIDToDockerfileRule[ruleID]))
			}
			t.Override(dockerfileRules, testRules)
			t.Override(&realWorkDir, func() (string, error) {
				return "", nil
			})
			t.Override(&getDockerDependenciesForEachFromTo, func(ctx context.Context, buildCfg docker.BuildConfig, cfg docker.Config) (map[string][]string, error) {
				deps := make([]string, 1001)
				for i := 0; i < 1001; i++ {
					deps[i] = fmt.Sprintf(".git/%d", i)
				}
				m := map[string][]string{}
				for _, fromTo := range test.dockerFromTo {
					if fromTo.From == "." {
						m[fromTo.String()] = deps
						continue
					}
					m[fromTo.String()] = []string{fromTo.From}
				}
				return m, nil
			})
			t.Override(&readCopyCmdsFromDockerfile, func(ctx context.Context, onlyLastImage bool, absDockerfilePath, workspace string, buildArgs map[string]*string, cfg docker.Config) ([]docker.FromTo, error) {
				return docker.ExtractOnlyCopyCommands(absDockerfilePath)
			})
			tmpdir := t.TempDir()
			configSet := parser.SkaffoldConfigSet{}
			// iteration done to enforce result order
			for i := 0; i < len(test.moduleAndSkaffoldYamls); i++ {
				module := fmt.Sprintf("cfg%d", i)
				skaffoldyamlText := test.moduleAndSkaffoldYamls[module]
				fp := filepath.Join(tmpdir, fmt.Sprintf("%s.yaml", module))
				err := ioutil.WriteFile(fp, []byte(skaffoldyamlText), 0644)
				if err != nil {
					t.Fatalf("error creating skaffold.yaml file with name %s: %v", fp, err)
				}
				dfp := filepath.Join(tmpdir, "Dockerfile")
				var b bytes.Buffer
				tmpl, err := template.New("dockerfileText").Parse(testDockerfile)
				if err != nil {
					t.Fatalf("error parsing dockerfileText go template: %v", err)
				}
				err = tmpl.Execute(&b, test.dockerFromTo)
				if err != nil {
					t.Fatalf("error executing dockerfileText go template: %v", err)
				}
				err = ioutil.WriteFile(dfp, b.Bytes(), 0644)
				if err != nil {
					t.Fatalf("error creating Dockerfile %s: %v", dfp, err)
				}
				configSet = append(configSet, &parser.SkaffoldConfigEntry{SkaffoldConfig: &latest.SkaffoldConfig{
					Metadata: latest.Metadata{Name: module},
					Pipeline: latest.Pipeline{Build: latest.BuildConfig{Artifacts: []*latest.Artifact{{Workspace: "",
						ArtifactType: latest.ArtifactType{DockerArtifact: &latest.DockerArtifact{DockerfilePath: dfp}}}}}},
				},
					SourceFile: fp,
				})
				// test overwrites file paths for expected DockerfileRules as they are made dynamically
				results := test.expected[module]
				if results == nil {
					continue
				}
				for i := range *results {
					(*results)[i].AbsFilePath = dfp
					(*results)[i].RelFilePath = dfp
				}
			}
			t.Override(&getConfigSet, func(_ context.Context, opts config.SkaffoldOptions) (parser.SkaffoldConfigSet, error) {
				// mock profile activation
				var set parser.SkaffoldConfigSet
				for _, c := range configSet {
					if len(opts.ConfigurationFilter) > 0 && !stringslice.Contains(opts.ConfigurationFilter, c.Metadata.Name) {
						continue
					}
					for _, pName := range opts.Profiles {
						for _, profile := range c.Profiles {
							if profile.Name != pName {
								continue
							}
							c.Test = profile.Test
						}
					}
					set = append(set, c)
				}
				return set, test.err
			})
			results, err := GetDockerfilesLintResults(context.Background(), Options{
				OutFormat: "json", Modules: test.modules, Profiles: test.profiles}, &runcontext.RunContext{})
			t.CheckError(test.shouldErr, err)
			if !test.shouldErr {
				expectedResults := &[]Result{}
				// this is done to enforce result order
				for i := 0; i < len(test.expected); i++ {
					*expectedResults = append(*expectedResults, *test.expected[fmt.Sprintf("cfg%d", i)]...)
					(*expectedResults)[0].Rule.ExplanationPopulator = nil
					(*expectedResults)[0].Rule.LintConditions = nil
				}

				if results == nil {
					t.CheckDeepEqual(expectedResults, results)
					return
				}
				for i := 0; i < len(*results); i++ {
					(*results)[i].Rule.ExplanationPopulator = nil
					(*results)[i].Rule.LintConditions = nil
				}
				t.CheckDeepEqual(expectedResults, results)
			}
		})
	}
}
