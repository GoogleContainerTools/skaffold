/*
Copyright 2018 The Skaffold Authors

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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/docker/docker/builder/dockerignore"
	"github.com/docker/docker/pkg/fileutils"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/karrick/godirwalk"
	"github.com/moby/buildkit/frontend/dockerfile/command"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/moby/buildkit/frontend/dockerfile/shell"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type from struct {
	image string
	as    string
}

// RetrieveImage is overridden for unit testing
var RetrieveImage = retrieveImage

func ValidateDockerfile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		logrus.Warnf("opening file %s: %s", path, err.Error())
		return false
	}
	res, err := parser.Parse(f)
	if err != nil || res == nil || len(res.AST.Children) == 0 {
		return false
	}
	// validate each node contains valid dockerfile directive
	for _, child := range res.AST.Children {
		_, ok := command.Commands[child.Value]
		if !ok {
			return false
		}
	}

	return true
}

func expandBuildArgs(nodes []*parser.Node, buildArgs map[string]*string) {
	for i, node := range nodes {
		if node.Value != command.Arg {
			continue
		}

		// build arg's key
		keyValue := strings.Split(node.Next.Value, "=")
		key := keyValue[0]

		// build arg's value
		var value string
		if buildArgs[key] != nil {
			value = *buildArgs[key]
		} else if len(keyValue) > 1 {
			value = keyValue[1]
		}

		for _, node := range nodes[i+1:] {
			// Stop replacements if an arg is redefined with the same key
			if node.Value == command.Arg && strings.Split(node.Next.Value, "=")[0] == key {
				break
			}

			// replace $key with value
			for curr := node; curr != nil; curr = curr.Next {
				curr.Value = util.Expand(curr.Value, key, value)
			}
		}
	}
}

func fromInstructions(nodes []*parser.Node) []from {
	var list []from

	for _, node := range nodes {
		if node.Value == command.From {
			list = append(list, fromInstruction(node))
		}
	}

	return list
}

func fromInstruction(node *parser.Node) from {
	var as string
	if next := node.Next.Next; next != nil && strings.ToLower(next.Value) == "as" && next.Next != nil {
		as = next.Next.Value
	}

	return from{
		image: node.Next.Value,
		as:    strings.ToLower(as),
	}
}

func onbuildInstructions(nodes []*parser.Node) ([]*parser.Node, error) {
	var instructions []string

	stages := map[string]bool{}
	for _, from := range fromInstructions(nodes) {
		// Stage names are case insensitive
		stages[strings.ToLower(from.as)] = true

		// `scratch` is case insensitive
		if strings.ToLower(from.image) == "scratch" {
			continue
		}

		// Stage names are case insensitive
		if _, found := stages[strings.ToLower(from.image)]; found {
			continue
		}

		logrus.Debugf("Checking base image %s for ONBUILD triggers.", from.image)

		// Image names are case SENSITIVE
		img, err := RetrieveImage(from.image)
		if err != nil {
			logrus.Warnf("Error processing base image (%s) for ONBUILD triggers: %s. Dependencies may be incomplete.", from.image, err)
			continue
		}

		if len(img.Config.OnBuild) > 0 {
			logrus.Debugf("Found ONBUILD triggers %v in image %s", img.Config.OnBuild, from.image)
			instructions = append(instructions, img.Config.OnBuild...)
		}
	}

	obRes, err := parser.Parse(strings.NewReader(strings.Join(instructions, "\n")))
	if err != nil {
		return nil, errors.Wrap(err, "parsing ONBUILD instructions")
	}

	return obRes.AST.Children, nil
}

func copiedFiles(nodes []*parser.Node) ([][]string, error) {
	var copied [][]string

	envs := map[string]string{}
	for _, node := range nodes {
		switch node.Value {
		case command.Add, command.Copy:
			files, err := processCopy(node, envs)
			if err != nil {
				return nil, err
			}

			if len(files) > 0 {
				copied = append(copied, files)
			}
		case command.Env:
			envs[node.Next.Value] = node.Next.Next.Value
		}
	}

	return copied, nil
}

func readDockerfile(workspace, absDockerfilePath string, target string, buildArgs map[string]*string) ([]string, error) {
	f, err := os.Open(absDockerfilePath)
	if err != nil {
		return nil, errors.Wrapf(err, "opening dockerfile: %s", absDockerfilePath)
	}
	defer f.Close()

	res, err := parser.Parse(f)
	if err != nil {
		return nil, errors.Wrap(err, "parsing dockerfile")
	}

	dockerfileLines := res.AST.Children

	expandBuildArgs(dockerfileLines, buildArgs)

	dockerfileLines, err = forTarget(target, dockerfileLines)
	if err != nil {
		return nil, errors.Wrap(err, "filtering on target")
	}

	instructions, err := onbuildInstructions(dockerfileLines)
	if err != nil {
		return nil, errors.Wrap(err, "listing ONBUILD instructions")
	}

	copied, err := copiedFiles(append(instructions, dockerfileLines...))
	if err != nil {
		return nil, errors.Wrap(err, "listing copied files")
	}

	return expandPaths(workspace, copied)
}

func forTarget(target string, nodes []*parser.Node) ([]*parser.Node, error) {
	if target == "" {
		return nodes, nil
	}

	byTarget := make(map[string][]*parser.Node)

	var currentTarget string
	for _, node := range nodes {
		if node.Value == command.From {
			currentTarget = fromInstruction(node).as
		}

		byTarget[currentTarget] = append(byTarget[currentTarget], node)
	}

	if _, present := byTarget[target]; !present {
		return nil, fmt.Errorf("failed to reach build target %s in Dockerfile", target)
	}

	return nodesForTarget(target, byTarget), nil
}

func nodesForTarget(target string, nodesByTarget map[string][]*parser.Node) []*parser.Node {
	var nodes []*parser.Node

	for _, node := range nodesByTarget[target] {
		if node.Value == command.From {
			inst := fromInstruction(node)
			nodes = append(nodes, nodesForTarget(inst.image, nodesByTarget)...)
		}

		nodes = append(nodes, node)
	}

	return nodes
}

func expandPaths(workspace string, copied [][]string) ([]string, error) {
	expandedPaths := make(map[string]bool)
	for _, files := range copied {
		matchesOne := false

		for _, p := range files {
			path := filepath.Join(workspace, p)
			if _, err := os.Stat(path); err == nil {
				expandedPaths[p] = true
				matchesOne = true
				continue
			}

			files, err := filepath.Glob(path)
			if err != nil {
				return nil, errors.Wrap(err, "invalid glob pattern")
			}
			if files == nil {
				continue
			}

			for _, f := range files {
				rel, err := filepath.Rel(workspace, f)
				if err != nil {
					return nil, fmt.Errorf("getting relative path of %s", f)
				}

				expandedPaths[rel] = true
			}
			matchesOne = true
		}

		if !matchesOne {
			return nil, fmt.Errorf("file pattern %s must match at least one file", files)
		}
	}

	var deps []string
	for dep := range expandedPaths {
		deps = append(deps, dep)
	}
	logrus.Debugf("Found dependencies for dockerfile: %v", deps)

	return deps, nil
}

// NormalizeDockerfilePath returns the absolute path to the dockerfile.
func NormalizeDockerfilePath(context, dockerfile string) (string, error) {
	if filepath.IsAbs(dockerfile) {
		return dockerfile, nil
	}

	if !strings.HasPrefix(dockerfile, context) {
		dockerfile = filepath.Join(context, dockerfile)
	}
	return filepath.Abs(dockerfile)
}

// GetDependencies finds the sources dependencies for the given docker artifact.
// All paths are relative to the workspace.
func GetDependencies(ctx context.Context, workspace string, a *latest.DockerArtifact) ([]string, error) {
	absDockerfilePath, err := NormalizeDockerfilePath(workspace, a.DockerfilePath)
	if err != nil {
		return nil, errors.Wrap(err, "normalizing dockerfile path")
	}

	deps, err := readDockerfile(workspace, absDockerfilePath, a.Target, a.BuildArgs)
	if err != nil {
		return nil, err
	}

	// Read patterns to ignore
	var excludes []string
	dockerignorePath := filepath.Join(workspace, ".dockerignore")
	if _, err := os.Stat(dockerignorePath); !os.IsNotExist(err) {
		r, err := os.Open(dockerignorePath)
		if err != nil {
			return nil, err
		}
		defer r.Close()

		excludes, err = dockerignore.ReadAll(r)
		if err != nil {
			return nil, err
		}
	}

	pExclude, err := fileutils.NewPatternMatcher(excludes)
	if err != nil {
		return nil, errors.Wrap(err, "invalid exclude patterns")
	}

	// Walk the workspace
	files := make(map[string]bool)
	for _, dep := range deps {
		dep = filepath.Clean(dep)
		absDep := filepath.Join(workspace, dep)

		fi, err := os.Stat(absDep)
		if err != nil {
			return nil, errors.Wrapf(err, "stating file %s", absDep)
		}

		switch mode := fi.Mode(); {
		case mode.IsDir():
			if err := godirwalk.Walk(absDep, &godirwalk.Options{
				Unsorted: true,
				Callback: func(fpath string, info *godirwalk.Dirent) error {
					if fpath == absDep {
						return nil
					}

					relPath, err := filepath.Rel(workspace, fpath)
					if err != nil {
						return err
					}

					ignored, err := pExclude.Matches(relPath)
					if err != nil {
						return err
					}

					if info.IsDir() {
						if ignored {
							return filepath.SkipDir
						}
					} else if !ignored {
						files[relPath] = true
					}

					return nil
				},
			}); err != nil {
				return nil, errors.Wrapf(err, "walking folder %s", absDep)
			}
		case mode.IsRegular():
			ignored, err := pExclude.Matches(dep)
			if err != nil {
				return nil, err
			}

			if !ignored {
				files[dep] = true
			}
		}
	}

	// Always add dockerfile even if it's .dockerignored. The daemon will need it anyways.
	if !filepath.IsAbs(a.DockerfilePath) {
		files[a.DockerfilePath] = true
	} else {
		files[absDockerfilePath] = true
	}

	// Ignore .dockerignore
	delete(files, ".dockerignore")

	var dependencies []string
	for file := range files {
		dependencies = append(dependencies, file)
	}
	sort.Strings(dependencies)

	return dependencies, nil
}

func retrieveImage(image string) (*v1.ConfigFile, error) {
	localDaemon, err := NewAPIClient() // Cached after first call
	if err != nil {
		return nil, errors.Wrap(err, "getting docker client")
	}

	return localDaemon.ConfigFile(context.Background(), image)
}

func processCopy(value *parser.Node, envs map[string]string) ([]string, error) {
	var copied []string

	slex := shell.NewLex('\\')
	for {
		// Skip last node, since it is the destination, and stop if we arrive at a comment
		if value.Next.Next == nil || strings.HasPrefix(value.Next.Next.Value, "#") {
			break
		}
		src, err := processShellWord(slex, value.Next.Value, envs)
		if err != nil {
			return nil, errors.Wrap(err, "processing word")
		}
		// If the --from flag is provided, we are dealing with a multi-stage dockerfile
		// Adding a dependency from a different stage does not imply a source dependency
		if hasMultiStageFlag(value.Flags) {
			return nil, nil
		}
		if !strings.HasPrefix(src, "http://") && !strings.HasPrefix(src, "https://") {
			copied = append(copied, src)
		} else {
			logrus.Debugf("Skipping watch on remote dependency %s", src)
		}

		value = value.Next
	}

	return copied, nil
}

func processShellWord(lex *shell.Lex, word string, envs map[string]string) (string, error) {
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
