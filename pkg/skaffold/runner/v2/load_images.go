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

package v2

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/docker/distribution/reference"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// loadImagesInKindNodes loads artifact images into every node of a kind cluster.
func (r *SkaffoldRunner) loadImagesInKindNodes(ctx context.Context, out io.Writer, kindCluster string, artifacts []graph.Artifact) error {
	output.Default.Fprintln(out, "Loading images into kind cluster nodes...")
	return r.loadImages(ctx, out, artifacts, func(tag string) *exec.Cmd {
		return exec.CommandContext(ctx, "kind", "load", "docker-image", "--name", kindCluster, tag)
	})
}

// loadImagesInK3dNodes loads artifact images into every node of a k3s cluster.
func (r *SkaffoldRunner) loadImagesInK3dNodes(ctx context.Context, out io.Writer, k3dCluster string, artifacts []graph.Artifact) error {
	output.Default.Fprintln(out, "Loading images into k3d cluster nodes...")
	return r.loadImages(ctx, out, artifacts, func(tag string) *exec.Cmd {
		return exec.CommandContext(ctx, "k3d", "image", "import", "--cluster", k3dCluster, tag)
	})
}

func (r *SkaffoldRunner) loadImages(ctx context.Context, out io.Writer, artifacts []graph.Artifact, createCmd func(tag string) *exec.Cmd) error {
	start := time.Now()

	var knownImages []string

	for _, artifact := range artifacts {
		// Only load images that this runner built
		if !r.wasBuilt(artifact.Tag) {
			continue
		}

		output.Default.Fprintf(out, " - %s -> ", artifact.Tag)

		// Only load images that are unknown to the node
		if knownImages == nil {
			var err error
			if knownImages, err = findKnownImages(ctx, r.kubectlCLI); err != nil {
				return fmt.Errorf("unable to retrieve node's images: %w", err)
			}
		}
		normalizedImageRef, err := reference.ParseNormalizedNamed(artifact.Tag)
		if err != nil {
			return err
		}
		if util.StrSliceContains(knownImages, normalizedImageRef.String()) {
			output.Green.Fprintln(out, "Found")
			continue
		}

		cmd := createCmd(artifact.Tag)
		if cmdOut, err := util.RunCmdOut(cmd); err != nil {
			output.Red.Fprintln(out, "Failed")
			return fmt.Errorf("unable to load image %q into cluster: %w, %s", artifact.Tag, err, cmdOut)
		}

		output.Green.Fprintln(out, "Loaded")
	}

	output.Default.Fprintln(out, "Images loaded in", util.ShowHumanizeTime(time.Since(start)))
	return nil
}

func findKnownImages(ctx context.Context, cli *kubectl.CLI) ([]string, error) {
	nodeGetOut, err := cli.RunOut(ctx, "get", "nodes", `-ojsonpath='{@.items[*].status.images[*].names[*]}'`)
	if err != nil {
		return nil, fmt.Errorf("unable to inspect the nodes: %w", err)
	}

	knownImages := strings.Split(string(nodeGetOut), " ")
	return knownImages, nil
}

func (r *SkaffoldRunner) wasBuilt(tag string) bool {
	for _, built := range r.Builds {
		if built.Tag == tag {
			return true
		}
	}

	return false
}
