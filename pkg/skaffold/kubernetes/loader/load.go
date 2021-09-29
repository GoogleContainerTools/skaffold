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

package loader

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/docker/distribution/reference"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

type ImageLoader struct {
	kubeContext string
	cli         *kubectl.CLI
}

type Config interface {
	kubectl.Config

	GetKubeContext() string
	LoadImages() bool
}

func NewImageLoader(kubeContext string, cli *kubectl.CLI) *ImageLoader {
	return &ImageLoader{
		kubeContext: kubeContext,
		cli:         cli,
	}
}

// We only load images that
//    1) Were identified as local images by the Runner, and
//    2) Are part of the set of images being deployed by a given Deployer, so we don't duplicate effort
func imagesToLoad(localImages, deployerImages, images []graph.Artifact) []graph.Artifact {
	local := map[string]bool{}
	for _, image := range localImages {
		local[docker.SanitizeImageName(image.ImageName)] = true
	}

	tracked := map[string]bool{}
	for _, image := range deployerImages {
		if local[image.ImageName] {
			tracked[image.ImageName] = true
		}
	}

	var res []graph.Artifact
	for _, image := range images {
		if tracked[docker.SanitizeImageName(image.ImageName)] {
			res = append(res, image)
		}
	}
	return res
}

// LoadImages loads images into a local cluster.
// imagesToLoad is used to determine the set of images we should load, based on images that are
// marked as local by the Runner, and part of the calling Deployer's set of manifests
func (i *ImageLoader) LoadImages(ctx context.Context, out io.Writer, localImages, deployerImages, images []graph.Artifact) error {
	currentContext, err := i.getCurrentContext()
	if err != nil {
		return err
	}

	artifacts := imagesToLoad(localImages, deployerImages, images)

	if config.IsKindCluster(i.kubeContext) {
		kindCluster := config.KindClusterName(currentContext.Cluster)

		// With `kind`, docker images have to be loaded with the `kind` CLI.
		if err := i.loadImagesInKindNodes(ctx, out, kindCluster, artifacts); err != nil {
			return fmt.Errorf("loading images into kind nodes: %w", err)
		}
	}

	if config.IsK3dCluster(i.kubeContext) {
		k3dCluster := config.K3dClusterName(currentContext.Cluster)

		// With `k3d`, docker images have to be loaded with the `k3d` CLI.
		if err := i.loadImagesInK3dNodes(ctx, out, k3dCluster, artifacts); err != nil {
			return fmt.Errorf("loading images into k3d nodes: %w", err)
		}
	}

	return nil
}

// loadImagesInKindNodes loads artifact images into every node of a kind cluster.
func (i *ImageLoader) loadImagesInKindNodes(ctx context.Context, out io.Writer, kindCluster string, artifacts []graph.Artifact) error {
	output.Default.Fprintln(out, "Loading images into kind cluster nodes...")
	return i.loadImages(ctx, out, artifacts, func(tag string) *exec.Cmd {
		return exec.CommandContext(ctx, "kind", "load", "docker-image", "--name", kindCluster, tag)
	})
}

// loadImagesInK3dNodes loads artifact images into every node of a k3s cluster.
func (i *ImageLoader) loadImagesInK3dNodes(ctx context.Context, out io.Writer, k3dCluster string, artifacts []graph.Artifact) error {
	output.Default.Fprintln(out, "Loading images into k3d cluster nodes...")
	return i.loadImages(ctx, out, artifacts, func(tag string) *exec.Cmd {
		return exec.CommandContext(ctx, "k3d", "image", "import", "--cluster", k3dCluster, tag)
	})
}

func (i *ImageLoader) loadImages(ctx context.Context, out io.Writer, artifacts []graph.Artifact, createCmd func(tag string) *exec.Cmd) error {
	start := time.Now()

	var knownImages []string

	for _, artifact := range artifacts {
		output.Default.Fprintf(out, " - %s -> ", artifact.Tag)

		// Only load images that are unknown to the node
		if knownImages == nil {
			var err error
			if knownImages, err = findKnownImages(ctx, i.cli); err != nil {
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
		if cmdOut, err := util.RunCmdOut(ctx, cmd); err != nil {
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

func (i *ImageLoader) getCurrentContext() (*api.Context, error) {
	currentCfg, err := kubectx.CurrentConfig()
	if err != nil {
		return nil, fmt.Errorf("unable to get kubernetes config: %w", err)
	}

	currentContext, present := currentCfg.Contexts[i.kubeContext]
	if !present {
		return nil, fmt.Errorf("unable to get current kubernetes context: %w", err)
	}
	return currentContext, nil
}
