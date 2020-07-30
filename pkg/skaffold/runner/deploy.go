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

package runner

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
)

func (r *SkaffoldRunner) Deploy(ctx context.Context, out io.Writer, artifacts []build.Artifact) error {
	if r.runCtx.Opts.RenderOnly {
		return r.Render(ctx, out, artifacts, false, r.runCtx.Opts.RenderOutput)
	}

	color.Default.Fprintln(out, "Tags used in deployment:")

	for _, artifact := range artifacts {
		color.Default.Fprintf(out, " - %s -> ", artifact.ImageName)
		fmt.Fprintln(out, artifact.Tag)
	}

	if r.imagesAreLocal && len(artifacts) > 0 {
		logrus.Debugln(`Local images can't be referenced by digest.
They are tagged and referenced by a unique, local only, tag instead.
See https://skaffold.dev/docs/pipeline-stages/taggers/#how-tagging-works`)
	}

	// Check that the cluster is reachable.
	// This gives a better error message when the cluster can't
	// be reached.
	if err := failIfClusterIsNotReachable(); err != nil {
		return fmt.Errorf("unable to connect to Kubernetes: %w", err)
	}

	if config.IsImageLoadingRequired(r.runCtx.KubeContext) {
		err := r.loadImagesIntoCluster(ctx, out, artifacts)
		if err != nil {
			return err
		}
	}

	event.DeployInProgress()
	namespaces, err := r.deployer.Deploy(ctx, out, artifacts)
	r.hasDeployed = true
	if err != nil {
		event.DeployFailed(err)
		return err
	}

	event.DeployComplete()
	r.runCtx.UpdateNamespaces(namespaces)
	return r.performStatusCheck(ctx, out)
}

func (r *SkaffoldRunner) loadImagesIntoCluster(ctx context.Context, out io.Writer, artifacts []build.Artifact) error {
	currentContext, err := r.getCurrentContext()
	if err != nil {
		return err
	}

	if config.IsKindCluster(r.runCtx.KubeContext) {
		kindCluster := config.KindClusterName(currentContext.Cluster)

		// With `kind`, docker images have to be loaded with the `kind` CLI.
		if err := r.loadImagesInKindNodes(ctx, out, kindCluster, artifacts); err != nil {
			return fmt.Errorf("loading images into kind nodes: %w", err)
		}
	}

	if config.IsK3dCluster(r.runCtx.KubeContext) {
		k3dCluster := config.K3dClusterName(currentContext.Cluster)

		// With `k3d`, docker images have to be loaded with the `k3d` CLI.
		if err := r.loadImagesInK3dNodes(ctx, out, k3dCluster, artifacts); err != nil {
			return fmt.Errorf("loading images into k3d nodes: %w", err)
		}
	}

	return nil
}

func (r *SkaffoldRunner) getCurrentContext() (*api.Context, error) {
	currentCfg, err := kubectx.CurrentConfig()
	if err != nil {
		return nil, fmt.Errorf("unable to get kubernetes config: %w", err)
	}

	currentContext, present := currentCfg.Contexts[r.runCtx.KubeContext]
	if !present {
		return nil, fmt.Errorf("unable to get current kubernetes context: %w", err)
	}
	return currentContext, nil
}

// failIfClusterIsNotReachable checks that Kubernetes is reachable.
// This gives a clear early error when the cluster can't be reached.
func failIfClusterIsNotReachable() error {
	client, err := kubernetes.Client()
	if err != nil {
		return err
	}

	_, err = client.Discovery().ServerVersion()
	return err
}

func (r *SkaffoldRunner) performStatusCheck(ctx context.Context, out io.Writer) error {
	// Check if we need to perform deploy status
	if !r.runCtx.Opts.StatusCheck {
		return nil
	}

	start := time.Now()
	color.Default.Fprintln(out, "Waiting for deployments to stabilize...")

	err := statusCheck(ctx, r.labeller, r.runCtx, out)
	if err != nil {
		return err
	}

	color.Default.Fprintln(out, "Deployments stabilized in", time.Since(start))
	return nil
}
