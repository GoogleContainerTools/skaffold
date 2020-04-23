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

package docker

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/moby/buildkit/frontend/dockerfile/command"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/moby/buildkit/frontend/dockerfile/shell"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

type from struct {
	image string
	as    string
}

// copyCommand records a docker COPY/ADD command.
type copyCommand struct {
	// srcs records the source glob patterns.
	srcs []string
	// dest records the destination which may be a directory.
	dest string
	// destIsDir indicates if dest must be treated as directory.
	destIsDir bool
}

type fromTo struct {
	// from is the relative path (wrt. the skaffold root directory) of the dependency on the host system.
	from string
	// to is the destination location in the container. Must use slashes as path separator.
	to string
	// toIsDir indicates if the `to` path must be treated as directory
	toIsDir bool
}

var (
	// RetrieveImage is overridden for unit testing
	RetrieveImage = retrieveImage
)

func readCopyCmdsFromDockerfile(onlyLastImage bool, absDockerfilePath, workspace string, buildArgs map[string]*string, insecureRegistries map[string]bool) ([]fromTo, error) {
	f, err := os.Open(absDockerfilePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	res, err := parser.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("parsing dockerfile %q: %w", absDockerfilePath, err)
	}

	dockerfileLines := res.AST.Children

	if err := expandBuildArgs(dockerfileLines, buildArgs); err != nil {
		return nil, fmt.Errorf("putting build arguments: %w", err)
	}

	dockerfileLinesWithOnbuild, err := expandOnbuildInstructions(dockerfileLines, insecureRegistries)
	if err != nil {
		return nil, err
	}

	cpCmds, err := extractCopyCommands(dockerfileLinesWithOnbuild, onlyLastImage, insecureRegistries)
	if err != nil {
		return nil, fmt.Errorf("listing copied files: %w", err)
	}

	return expandSrcGlobPatterns(workspace, cpCmds)
}

func expandBuildArgs(nodes []*parser.Node, buildArgs map[string]*string) error {
	args, err := EvaluateBuildArgs(buildArgs)
	if err != nil {
		return fmt.Errorf("unable to evaluate build args: %w", err)
	}

	for i, node := range nodes {
		if node.Value != command.Arg {
			continue
		}

		// build arg's key
		keyValue := strings.Split(node.Next.Value, "=")
		key := keyValue[0]

		// build arg's value
		var value string
		if args[key] != nil {
			value = *args[key]
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
	return nil
}

func expandSrcGlobPatterns(workspace string, cpCmds []*copyCommand) ([]fromTo, error) {
	var fts []fromTo
	for _, cpCmd := range cpCmds {
		matchesOne := false

		for _, p := range cpCmd.srcs {
			path := filepath.Join(workspace, p)
			if _, err := os.Stat(path); err == nil {
				fts = append(fts, fromTo{from: filepath.Clean(p), to: cpCmd.dest, toIsDir: cpCmd.destIsDir})
				matchesOne = true
				continue
			}

			files, err := filepath.Glob(path)
			if err != nil {
				return nil, fmt.Errorf("invalid glob pattern: %w", err)
			}
			if files == nil {
				continue
			}

			for _, f := range files {
				rel, err := filepath.Rel(workspace, f)
				if err != nil {
					return nil, fmt.Errorf("getting relative path of %s", f)
				}

				fts = append(fts, fromTo{from: rel, to: cpCmd.dest, toIsDir: cpCmd.destIsDir})
			}
			matchesOne = true
		}

		if !matchesOne {
			return nil, fmt.Errorf("file pattern %s must match at least one file", cpCmd.srcs)
		}
	}

	logrus.Debugf("Found dependencies for dockerfile: %v", fts)
	return fts, nil
}

func extractCopyCommands(nodes []*parser.Node, onlyLastImage bool, insecureRegistries map[string]bool) ([]*copyCommand, error) {
	stages := map[string]bool{
		"scratch": true,
	}

	slex := shell.NewLex('\\')
	var copied []*copyCommand

	workdir := "/"
	envs := make([]string, 0)
	for _, node := range nodes {
		switch node.Value {
		case command.From:
			from := fromInstruction(node)
			if from.as != "" {
				// Stage names are case insensitive
				stages[strings.ToLower(from.as)] = true
			}

			// If `from` references a previous stage, then the `workdir`
			// was already changed.
			if !stages[strings.ToLower(from.image)] {
				img, err := RetrieveImage(from.image, insecureRegistries)
				if err != nil {
					return nil, err
				}

				workdir = img.Config.WorkingDir
				if workdir == "" {
					workdir = "/"
				}
			}

			if onlyLastImage {
				copied = nil
			}
		case command.Workdir:
			value, err := slex.ProcessWord(node.Next.Value, envs)
			if err != nil {
				return nil, fmt.Errorf("processing word: %w", err)
			}
			workdir = resolveDir(workdir, value)
		case command.Add, command.Copy:
			cpCmd, err := readCopyCommand(node, envs, workdir)
			if err != nil {
				return nil, err
			}

			if cpCmd != nil && len(cpCmd.srcs) > 0 {
				copied = append(copied, cpCmd)
			}
		case command.Env:
			// one env command may define multiple variables
			for node := node.Next; node != nil && node.Next != nil; node = node.Next.Next {
				envs = append(envs, fmt.Sprintf("%s=%s", node.Value, node.Next.Value))
			}
		}
	}

	return copied, nil
}

func readCopyCommand(value *parser.Node, envs []string, workdir string) (*copyCommand, error) {
	// If the --from flag is provided, we are dealing with a multi-stage dockerfile
	// Adding a dependency from a different stage does not imply a source dependency
	if hasMultiStageFlag(value.Flags) {
		return nil, nil
	}

	var paths []string
	slex := shell.NewLex('\\')
	for value := value.Next; value != nil && !strings.HasPrefix(value.Value, "#"); value = value.Next {
		path, err := slex.ProcessWord(value.Value, envs)
		if err != nil {
			return nil, fmt.Errorf("expanding src: %w", err)
		}

		paths = append(paths, path)
	}

	// All paths are sources except the last one
	var srcs []string
	for _, src := range paths[0 : len(paths)-1] {
		if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
			logrus.Debugln("Skipping watch on remote dependency", src)
			continue
		}

		srcs = append(srcs, src)
	}

	// Destination is last
	dest := paths[len(paths)-1]
	destIsDir := strings.HasSuffix(dest, "/") || path.Base(dest) == "." || path.Base(dest) == ".."
	dest = resolveDir(workdir, dest)

	return &copyCommand{
		srcs:      srcs,
		dest:      dest,
		destIsDir: destIsDir,
	}, nil
}

func expandOnbuildInstructions(nodes []*parser.Node, insecureRegistries map[string]bool) ([]*parser.Node, error) {
	onbuildNodesCache := map[string][]*parser.Node{
		"scratch": nil,
	}
	var expandedNodes []*parser.Node
	n := 0
	for m, node := range nodes {
		if node.Value == command.From {
			from := fromInstruction(node)

			// onbuild should immediately follow the from command
			expandedNodes = append(expandedNodes, nodes[n:m+1]...)
			n = m + 1

			var onbuildNodes []*parser.Node
			if ons, found := onbuildNodesCache[strings.ToLower(from.image)]; found {
				onbuildNodes = ons
			} else if ons, err := parseOnbuild(from.image, insecureRegistries); err == nil {
				onbuildNodes = ons
			} else {
				return nil, fmt.Errorf("parsing ONBUILD instructions: %w", err)
			}

			// Stage names are case insensitive
			onbuildNodesCache[strings.ToLower(from.as)] = nodes
			onbuildNodesCache[strings.ToLower(from.image)] = nodes

			expandedNodes = append(expandedNodes, onbuildNodes...)
		}
	}
	expandedNodes = append(expandedNodes, nodes[n:]...)

	return expandedNodes, nil
}

func parseOnbuild(image string, insecureRegistries map[string]bool) ([]*parser.Node, error) {
	logrus.Tracef("Checking base image %s for ONBUILD triggers.", image)

	// Image names are case SENSITIVE
	img, err := RetrieveImage(image, insecureRegistries)
	if err != nil {
		return nil, fmt.Errorf("retrieving image %q: %w", image, err)
	}

	if len(img.Config.OnBuild) == 0 {
		return []*parser.Node{}, nil
	}

	logrus.Tracef("Found ONBUILD triggers %v in image %s", img.Config.OnBuild, image)

	obRes, err := parser.Parse(strings.NewReader(strings.Join(img.Config.OnBuild, "\n")))
	if err != nil {
		return nil, err
	}

	return obRes.AST.Children, nil
}

func fromInstruction(node *parser.Node) from {
	var as string
	if next := node.Next.Next; next != nil && strings.ToLower(next.Value) == "as" && next.Next != nil {
		as = next.Next.Value
	}

	return from{
		image: unquote(node.Next.Value),
		as:    strings.ToLower(as),
	}
}

// unquote remove single quote/double quote pairs around a string value.
// It looks like FROM "scratch" and FROM 'scratch' and FROM """scratch"""...
// are valid forms of FROM scratch.
func unquote(v string) string {
	unquoted := strings.TrimFunc(v, func(r rune) bool { return r == '"' })
	if unquoted != v {
		return unquoted
	}

	unquoted = strings.TrimFunc(v, func(r rune) bool { return r == '\'' })
	return unquoted
}

func retrieveImage(image string, insecureRegistries map[string]bool) (*v1.ConfigFile, error) {
	// TODO: use the proper RunContext
	localDaemon, err := NewAPIClient(&runcontext.RunContext{
		InsecureRegistries: insecureRegistries,
	})
	if err != nil {
		return nil, fmt.Errorf("getting docker client: %w", err)
	}

	return localDaemon.ConfigFile(context.Background(), image)
}

func hasMultiStageFlag(flags []string) bool {
	for _, f := range flags {
		if strings.HasPrefix(f, "--from=") {
			return true
		}
	}
	return false
}

// resolveDir determines the resulting directory as if a change-dir to targetDir was executed in cwd.
func resolveDir(cwd, targetDir string) string {
	if path.IsAbs(targetDir) {
		return path.Clean(targetDir)
	}
	return path.Clean(path.Join(cwd, targetDir))
}
