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
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/moby/buildkit/frontend/dockerfile/command"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/moby/buildkit/frontend/dockerfile/shell"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
)

const buildkitUnresolvedImagePlaceholder = "image:latest"

type FromTo struct {
	// From is the relative path (wrt. the skaffold root directory) of the dependency on the host system.
	From string
	// To is the destination location in the container. Must use slashes as path separator.
	To string
	// ToIsDir indicates if the `to` path must be treated as directory
	ToIsDir bool
	// StartLine indicates the starting line in the dockerfile of the copy command
	StartLine int
	// EndLine indiciates the ending line in the dockerfile of the copy command
	EndLine int
}

func (f *FromTo) String() string {
	return fmt.Sprintf("From:%s, To:%s, ToIsDir:%t, StartLine: %d, EndLine: %d", f.From, f.To, f.ToIsDir, f.StartLine, f.EndLine)
}

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
	// startLine indicates the starting line in the dockerfile of the copy command
	startLine int
	// endLine indiciates the ending line in the dockerfile of the copy command
	endLine int
}

var (
	// RetrieveImage is overridden for unit testing
	RetrieveImage = retrieveImage
)

// ReadCopyCmdsFromDockerfile parses a given dockerfile for COPY commands accounting for build args, env vars, globs, etc
// and returns an array of FromTos specifying the files that will be copied 'from' local dirs 'to' container dirs in the COPY statements
func ReadCopyCmdsFromDockerfile(ctx context.Context, onlyLastImage bool, absDockerfilePath, workspace string, buildArgs map[string]*string, cfg Config) ([]FromTo, error) {
	r, err := os.ReadFile(absDockerfilePath)
	if err != nil {
		return nil, err
	}

	res, err := parser.Parse(bytes.NewReader(r))
	if err != nil {
		return nil, fmt.Errorf("parsing dockerfile %q: %w", absDockerfilePath, err)
	}

	if err := validateParsedDockerfile(bytes.NewReader(r), res); err != nil {
		return nil, fmt.Errorf("parsing dockerfile %q: %w", absDockerfilePath, err)
	}

	dockerfileLines := res.AST.Children

	if err := expandBuildArgs(dockerfileLines, buildArgs); err != nil {
		return nil, fmt.Errorf("putting build arguments: %w", err)
	}

	dockerfileLinesWithOnbuild, err := expandOnbuildInstructions(ctx, dockerfileLines, cfg)
	if err != nil {
		return nil, err
	}

	cpCmds, err := extractCopyCommands(ctx, dockerfileLinesWithOnbuild, onlyLastImage, cfg)
	if err != nil {
		return nil, fmt.Errorf("listing copied files: %w", err)
	}

	return expandSrcGlobPatterns(workspace, cpCmds)
}

func ExtractOnlyCopyCommands(absDockerfilePath string) ([]FromTo, error) {
	r, err := os.ReadFile(absDockerfilePath)
	if err != nil {
		return nil, err
	}

	res, err := parser.Parse(bytes.NewReader(r))
	if err != nil {
		return nil, fmt.Errorf("parsing dockerfile %q: %w", absDockerfilePath, err)
	}

	var copied []FromTo
	workdir := "/"
	envs := make([]string, 0)
	for _, node := range res.AST.Children {
		switch strings.ToLower(node.Value) {
		case command.Add, command.Copy:
			cpCmd, err := readCopyCommand(node, envs, workdir)
			if err != nil {
				return nil, err
			}

			if cpCmd != nil && len(cpCmd.srcs) > 0 {
				for _, src := range cpCmd.srcs {
					copied = append(copied, FromTo{From: src, To: cpCmd.dest, ToIsDir: cpCmd.destIsDir, StartLine: cpCmd.startLine, EndLine: cpCmd.endLine})
				}
			}
		}
	}
	return copied, nil
}

// filterUnusedBuildArgs removes entries from the build arguments map that are not found in the dockerfile
func filterUnusedBuildArgs(dockerFile io.Reader, buildArgs map[string]*string) (map[string]*string, error) {
	res, err := parser.Parse(dockerFile)
	if err != nil {
		return nil, fmt.Errorf("parsing dockerfile: %w", err)
	}
	m := make(map[string]*string)
	for _, n := range res.AST.Children {
		if strings.ToLower(n.Value) != command.Arg {
			continue
		}
		k := strings.SplitN(n.Next.Value, "=", 2)[0]
		if v, ok := buildArgs[k]; ok {
			m[k] = v
		}
	}
	return m, nil
}

func expandBuildArgs(nodes []*parser.Node, buildArgs map[string]*string) error {
	args, err := util.EvaluateEnvTemplateMap(buildArgs)
	if err != nil {
		return fmt.Errorf("unable to evaluate build args: %w", err)
	}

	for i, node := range nodes {
		if strings.ToLower(node.Value) != command.Arg {
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
			if strings.ToLower(node.Value) == command.Arg && strings.Split(node.Next.Value, "=")[0] == key {
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

func expandSrcGlobPatterns(workspace string, cpCmds []*copyCommand) ([]FromTo, error) {
	var fts []FromTo
	for _, cpCmd := range cpCmds {
		matchesOne := false

		for _, p := range cpCmd.srcs {
			path := filepath.Join(workspace, p)
			if _, err := os.Stat(path); err == nil {
				fts = append(fts, FromTo{From: filepath.Clean(p), To: cpCmd.dest, ToIsDir: cpCmd.destIsDir, StartLine: cpCmd.startLine, EndLine: cpCmd.endLine})
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

				fts = append(fts, FromTo{From: rel, To: cpCmd.dest, ToIsDir: cpCmd.destIsDir, StartLine: cpCmd.startLine, EndLine: cpCmd.endLine})
			}
			matchesOne = true
		}

		if !matchesOne {
			return nil, fmt.Errorf("file pattern %s must match at least one file", cpCmd.srcs)
		}
	}

	log.Entry(context.TODO()).Debugf("Found dependencies for dockerfile: %v", fts)

	return fts, nil
}

func extractCopyCommands(ctx context.Context, nodes []*parser.Node, onlyLastImage bool, cfg Config) ([]*copyCommand, error) {
	stages := map[string]bool{
		"scratch": true,
	}

	slex := shell.NewLex('\\')
	var copied []*copyCommand

	workdir := "/"
	envs := make([]string, 0)
	for _, node := range nodes {
		switch strings.ToLower(node.Value) {
		case command.From:
			from := fromInstruction(node)
			if from.as != "" {
				// Stage names are case insensitive
				stages[strings.ToLower(from.as)] = true
			}

			if from.image == "" {
				// some build args like artifact dependencies are not available until the first build sequence has completed.
				// skip check if there are unavailable images
				continue
			}

			// If `from` references a previous stage, then the `workdir`
			// was already changed.
			if !stages[strings.ToLower(from.image)] {
				img, err := RetrieveImage(ctx, from.image, cfg)
				if err == nil {
					workdir = img.Config.WorkingDir
				} else if _, ok, err := isOldImageManifestProblem(cfg, err); !ok {
					return nil, err
				}
				if workdir == "" {
					workdir = "/"
				}
			}
			if onlyLastImage {
				copied = nil
			}
		case command.Workdir:
			value, _, err := slex.ProcessWord(node.Next.Value, shell.EnvsFromSlice(envs))
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
			for node := node.Next; node != nil && node.Next != nil && node.Next.Next != nil; node = node.Next.Next.Next {
				envs = append(envs, fmt.Sprintf("%s=%s", node.Value, unquote(node.Next.Value)))
			}
		}
	}
	return copied, nil
}

func hasOneOfPrefixes(str string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(str, prefix) {
			return true
		}
	}
	return false
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
		path, _, err := slex.ProcessWord(value.Value, shell.EnvsFromSlice(envs))
		if err != nil {
			return nil, fmt.Errorf("expanding src: %w", err)
		}

		paths = append(paths, path)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("invalid dockerfile instruction: %q", value.Original)
	}

	// All paths are sources except the last one
	var srcs []string
	for _, src := range paths[0 : len(paths)-1] {
		if hasOneOfPrefixes(src, []string{"http://", "https://", "<<"}) {
			log.Entry(context.TODO()).Debugln("Skipping watch on remote/heredoc dependency", src)
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
		startLine: value.StartLine,
		endLine:   value.EndLine,
	}, nil
}

func expandOnbuildInstructions(ctx context.Context, nodes []*parser.Node, cfg Config) ([]*parser.Node, error) {
	onbuildNodesCache := map[string][]*parser.Node{
		"scratch": nil,
	}
	var expandedNodes []*parser.Node
	n := 0
	for m, node := range nodes {
		if strings.ToLower(node.Value) == command.From {
			from := fromInstruction(node)

			// onbuild should immediately follow the from command
			expandedNodes = append(expandedNodes, nodes[n:m+1]...)
			n = m + 1

			var onbuildNodes []*parser.Node
			if ons, found := onbuildNodesCache[strings.ToLower(from.image)]; found {
				onbuildNodes = ons
			} else if from.image == "" || from.image == buildkitUnresolvedImagePlaceholder {
				// some build args like artifact dependencies are not available until the first build sequence has completed.
				// skip check if there are unavailable images
				onbuildNodes = []*parser.Node{}
			} else if ons, err := parseOnbuild(ctx, from.image, cfg); err == nil {
				onbuildNodes = ons
			} else if warnMsg, ok, _ := isOldImageManifestProblem(cfg, err); ok && warnMsg != "" {
				log.Entry(context.TODO()).Warn(warnMsg)
			} else if !ok {
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

func parseOnbuild(ctx context.Context, image string, cfg Config) ([]*parser.Node, error) {
	log.Entry(context.TODO()).Tracef("Checking base image %s for ONBUILD triggers.", image)

	// Image names are case SENSITIVE
	img, err := RetrieveImage(ctx, image, cfg)
	if err != nil {
		return nil, fmt.Errorf("retrieving image %q: %w", image, err)
	}

	if len(img.Config.OnBuild) == 0 {
		return []*parser.Node{}, nil
	}

	log.Entry(context.TODO()).Tracef("Found ONBUILD triggers %v in image %s", img.Config.OnBuild, image)

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
// Quotes are also accepted on tags, e.g. golang:"1.15".
func unquote(v string) string {
	unquoted := strings.ReplaceAll(v, "\"", "")
	if unquoted != v {
		return unquoted
	}

	unquoted = strings.ReplaceAll(v, "'", "")
	return unquoted
}

func retrieveImage(ctx context.Context, image string, cfg Config) (*v1.ConfigFile, error) {
	localDaemon, err := NewAPIClient(ctx, cfg)
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

func validateParsedDockerfile(r io.Reader, res *parser.Result) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	// skip validation for dockerfiles using explicit `syntax` directive
	if _, _, _, usesSyntax := parser.DetectSyntax(b); usesSyntax {
		return nil
	}
	// instructions.Parse will check for malformed Dockerfile
	_, _, err = instructions.Parse(res.AST, nil)
	return err
}

func isOldImageManifestProblem(cfg Config, err error) (string, bool, error) {
	// regex for detecting old manifest images
	retrieveFailedOldManifest := `(.*retrieving image.*\"(.*)\")?.*unsupported MediaType.*manifest\.v1\+prettyjws.*`
	matchExp := regexp.MustCompile(retrieveFailedOldManifest)
	if !matchExp.MatchString(err.Error()) {
		return "", false, nil
	}
	p := sErrors.Problem{
		Description: func(err error) string {
			match := matchExp.FindStringSubmatch(err.Error())
			pre := "Could not retrieve image pushed with the deprecated manifest v1"
			if len(match) >= 3 && match[2] != "" {
				pre = fmt.Sprintf("Could not retrieve image %s pushed with the deprecated manifest v1", match[2])
			}
			return fmt.Sprintf("%s. Ignoring files dependencies for all ONBUILD triggers", pre)
		},
		ErrCode: proto.StatusCode_DEVINIT_UNSUPPORTED_V1_MANIFEST,
		Suggestion: func(i interface{}) []*proto.Suggestion {
			cfg, ok := i.(Config)
			if !ok {
				return nil
			}
			if cfg.Mode() == config.RunModes.Dev || cfg.Mode() == config.RunModes.Debug {
				return []*proto.Suggestion{{
					SuggestionCode: proto.SuggestionCode_RUN_DOCKER_PULL,
					Action:         "To avoid, hit Cntrl-C and run `docker pull` to fetch the specified image and retry",
				}}
			}
			return nil
		},
		Err: err,
	}
	var warnMsg string
	if cfg.Mode() == config.RunModes.Dev || cfg.Mode() == config.RunModes.Debug {
		warnMsg = p.AIError(cfg, err).Error()
	}
	return warnMsg, true, p
}
