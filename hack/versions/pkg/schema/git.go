/*
Copyright 2019 The Skaffold Authors

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

package schema

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

type gitClient interface {
	getChangedFiles() ([]string, error)
	getFileFromBaseline(path string) ([]byte, error)
	diffWithBaseline(path string) ([]byte, error)
}

type git struct {
	path    string
	baseRef string
}

func newGit(baseRef string) (gitClient, error) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return nil, fmt.Errorf("failed to find git on PATH: %w", err)
	}
	return &git{
		path:    gitPath,
		baseRef: baseRef,
	}, nil
}

func (g *git) getChangedFiles() ([]string, error) {
	out, err := g.run("diff", "--name-only", g.baseRef, "--", "pkg/skaffold/schema")
	if err != nil {
		return nil, err
	}

	return strings.Split(string(out), "\n"), nil
}

func (g *git) getFileFromBaseline(path string) ([]byte, error) {
	out, err := g.run("show", fmt.Sprintf("%s:%s", g.baseRef, path))
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (g *git) diffWithBaseline(path string) ([]byte, error) {
	out, err := g.run("diff", g.baseRef, "--", path)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (g *git) run(args ...string) ([]byte, error) {
	cmd := exec.Command(g.path, args...)
	out, err := util.RunCmdOut(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed running %v: %s\n%s", cmd.Args, err, string(out))
	}
	return out, nil
}
