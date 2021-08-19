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
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/inspect"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/moby/buildkit/frontend/dockerfile/command"
	"github.com/sirupsen/logrus"
)

// TODO(aaron-prindle) FIX, not correct
var DockerfileLinters = []FileLinter{
	&StringEqualsLinter{},
	&RegexpLinter{},
	&DockerfileCommandLinter{},
}

var DockerfileLintRules = []LintRule{
	{
		// TODO(aaron-prindle), why doesn't start of line anchor work for regexp? Because it is one big string? eg: "(?i)^COPY [.] [.]"
		DockerCommand:    command.Copy,
		DockerCopySource: ".",
		LintRuleId:       DOCKERFILE_PLACEHOLDER,
		LintRuleType:     DockerfileCommandCheck,
		// TODO(aaron-prindle) figure out how to best do conditions...
		// can do them here which is better or can hardcode them in the linters with the specific IDs
		LintConditions: []func(string) bool{func(sourcePath string) bool {
			files := 0
			err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					logrus.Errorf("DOCKERFILE_PLACEHOLDER lint condition encountered error: %v", err)
					return err
				}
				files++
				return nil
			})
			if err != nil {
				logrus.Errorf("DOCKERFILE_PLACEHOLDER lint condition encountered error: %v", err)
				return false
			}
			return files > 100
		}},
		Explanation: "Found 'COPY . <DEST>', for a source directory that has > 100 files.  This has the potential to dramatically slow 'skaffold dev' down by " +
			"having skaffold watch all of the files in the copied directory for changes. " +
			"If you notice skaffold rebuilding images unnecessarily when non-image-critical files are " +
			"modified, consider changing this to `COPY $REQUIRED_SOURCE_FILE <DEST>` for each required source file instead of " +
			"using 'COPY . <DEST>'",
	},
}

func GetDockerfilesList(ctx context.Context, out io.Writer, opts inspect.Options) (*DockerfileLintRulesList, error) {
	cfgs, err := inspect.GetConfigSet(ctx, config.SkaffoldOptions{
		ConfigurationFile:   opts.Filename,
		ConfigurationFilter: opts.Modules,
		RepoCacheDir:        opts.RepoCacheDir,
		Profiles:            opts.LintProfiles,
	})
	if err != nil {
		return nil, nil
	}

	l := &DockerfileLintRulesList{}
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
				mrs := []MatchResult{}
				for _, r := range DockerfileLinters {
					recs, err := r.Lint(dockerfile, &DockerfileLintRules)
					if err != nil {
						return nil, err
					}
					mrs = append(mrs, *recs...)
				}
				l.DockerfileLintRules = append(l.DockerfileLintRules, mrs...)
				l.Dockerfiles = append(l.Dockerfiles, dockerfile)
			}
		}
	}
	return l, nil
}
