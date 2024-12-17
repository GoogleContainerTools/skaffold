// Copyright 2020 ko Build Authors All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kind

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"golang.org/x/sync/errgroup"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"

	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
)

// Supported since kind 0.8.0 (https://github.com/kubernetes-sigs/kind/releases/tag/v0.8.0)
const clusterNameEnvKey = "KIND_CLUSTER_NAME"

// provider is an interface for kind providers to facilitate testing.
type provider interface {
	ListInternalNodes(name string) ([]nodes.Node, error)
}

// GetProvider is a variable so we can override in tests.
var GetProvider = func() provider {
	return cluster.NewProvider()
}

// Tag adds a tag to an already existent image.
func Tag(ctx context.Context, src, dest name.Tag) error {
	return onEachNode(func(n nodes.Node) error {
		var buf bytes.Buffer
		cmd := n.CommandContext(ctx, "ctr", "--namespace=k8s.io", "images", "tag", "--force", src.String(), dest.String())
		cmd.SetStdout(&buf)
		cmd.SetStderr(&buf)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to tag image: %w\n%s", err, buf.String())
		}
		return nil
	})
}

// Write saves the image into the kind nodes as the given tag.
func Write(ctx context.Context, tag name.Tag, img v1.Image) error {
	return onEachNode(func(n nodes.Node) error {
		pr, pw := io.Pipe()

		grp := errgroup.Group{}
		grp.Go(func() error {
			return pw.CloseWithError(tarball.Write(tag, img, pw))
		})

		var buf bytes.Buffer
		cmd := n.CommandContext(ctx, "ctr", "--namespace=k8s.io", "images", "import", "--all-platforms", "-").SetStdin(pr)
		cmd.SetStdout(&buf)
		cmd.SetStderr(&buf)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to load image to node %q: %w\n%s", n, err, buf.String())
		}

		if err := grp.Wait(); err != nil {
			return fmt.Errorf("failed to write intermediate tarball representation: %w", err)
		}

		return nil
	})
}

// onEachNode executes the given function on each node. Exits on first error.
func onEachNode(f func(nodes.Node) error) error {
	nodeList, err := getNodes()
	if err != nil {
		return err
	}

	for _, n := range nodeList {
		if err := f(n); err != nil {
			return err
		}
	}
	return nil
}

// getNodes gets all the nodes of the default cluster.
// Returns an error if none were found.
func getNodes() ([]nodes.Node, error) {
	provider := GetProvider()

	clusterName := os.Getenv(clusterNameEnvKey)
	if clusterName == "" {
		clusterName = cluster.DefaultName
	}

	nodeList, err := provider.ListInternalNodes(clusterName)
	if err != nil {
		return nil, err
	}
	if len(nodeList) == 0 {
		return nil, fmt.Errorf("no nodes found for cluster %q", clusterName)
	}

	return nodeList, nil
}
