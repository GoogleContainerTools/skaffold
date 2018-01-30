/*
Copyright 2018 Google LLC

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

package docker

import (
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/GoogleCloudPlatform/skaffold/third_party/moby/moby/dockerfile"
	"github.com/moby/moby/builder/dockerfile/parser"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	add  = "add"
	copy = "copy"
	env  = "env"
)

// GetDockerfileDependencies parses a dockerfile and returns the full paths
// of all the source files that the resulting docker image depends on.
// GetDockerfileDependencies does not expand paths and may contain both a directory and
// files within it.
func GetDockerfileDependencies(workspace string, r io.Reader) ([]string, error) {
	res, err := parser.Parse(r)
	if err != nil {
		return nil, errors.Wrap(err, "parsing dockerfile")
	}
	slex := dockerfile.NewShellLex('\\')
	deps := []string{}
	envs := map[string]string{}
	seen := map[string]struct{}{}
	for _, value := range res.AST.Children {
		logrus.Debugf("Dockerfile instruction: %+v", value)
		switch value.Value {
		case add, copy:
			src, err := processShellWord(slex, value.Next.Value, envs)
			if err != nil {
				return nil, errors.Wrap(err, "processing word")
			}
			// If the --from flag is provided, we are dealing with a multi-stage dockerfile
			// Adding a dependency from a different stage does not imply a source dependency
			if hasMultiStageFlag(value.Flags) {
				continue
			}
			depPath := path.Join(workspace, src)
			if _, ok := seen[depPath]; ok {
				// If we've already seen this file, only add it once.
				continue
			}
			seen[depPath] = struct{}{}
			deps = append(deps, depPath)
		case env:
			envs[value.Next.Value] = value.Next.Next.Value
		}
	}
	return deps, nil
}

func processShellWord(lex *dockerfile.ShellLex, word string, envs map[string]string) (string, error) {
	envSlice := []string{}
	for envKey, envVal := range envs {
		envSlice = append(envSlice, fmt.Sprintf("%s=%s", envKey, envVal))
	}
	return lex.ProcessWord(word, envSlice)
}

func hasMultiStageFlag(flags []string) bool {
	for _, f := range flags {
		if strings.HasPrefix(f, "--from=") {
			return true
		}
	}
	return false
}
