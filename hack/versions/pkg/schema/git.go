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
	"github.com/pkg/errors"
)

type gitClient interface {
	getChangedFiles() ([]string, error)
	getFileFromRef(path string, ref string) ([]byte, error)
	diffWithRef(path string, ref string) ([]byte, error)
}

type git struct {
	path string
}

func newGit() (gitClient, error) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return nil, errors.Wrap(err, "failed to find git on PATH")
	}
	return &git{
		path: gitPath,
	}, nil
}

func (g *git) getChangedFiles() ([]string, error) {
	out, err := g.run("diff", "--name-only", "master", "--", "pkg/skaffold/schema")
	if err != nil {
		return nil, err
	}

	return strings.Split(string(out), "\n"), nil
}

func (g *git) getFileFromRef(path string, ref string) ([]byte, error) {
	out, err := g.run("show", fmt.Sprintf("%s:%s", ref, path))
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (g *git) diffWithRef(path string, ref string) ([]byte, error) {
	out, err := g.run("diff", ref, "--", path)
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
