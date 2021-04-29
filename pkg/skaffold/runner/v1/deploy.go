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

package v1

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	deployutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	eventV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	kubernetesclient "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/client"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// DeployAndLog deploys a list of already built artifacts and optionally show the logs.
func (r *SkaffoldRunner) DeployAndLog(ctx context.Context, out io.Writer, artifacts []graph.Artifact) error {
	defer r.deployer.StopLogger()

	// Logs should be retrieved up to just before the deploy
	r.deployer.SetSince(time.Now())
	// First deploy
	if err := r.Deploy(ctx, out, artifacts); err != nil {
		return err
	}

	defer r.deployer.StopResourcePreview()

	if err := r.deployer.StartResourcePreview(ctx, out, r.runCtx.GetNamespaces()); err != nil {
		logrus.Warnln("Error starting port forwarding:", err)
	}

	// Start printing the logs after deploy is finished
	if err := r.deployer.StartLogger(ctx, out, r.runCtx.GetNamespaces()); err != nil {
		return fmt.Errorf("starting logger: %w", err)
	}

	if r.runCtx.Tail() || r.runCtx.PortForward() {
		output.Yellow.Fprintln(out, "Press Ctrl+C to exit")
		<-ctx.Done()
	}

	return nil
}

func (r *SkaffoldRunner) Deploy(ctx context.Context, out io.Writer, artifacts []graph.Artifact) error {
	if r.runCtx.RenderOnly() {
		return r.Render(ctx, out, artifacts, false, r.runCtx.RenderOutput())
	}

	output.Default.Fprintln(out, "Tags used in deployment:")

	for _, artifact := range artifacts {
		output.Default.Fprintf(out, " - %s -> ", artifact.ImageName)
		fmt.Fprintln(out, artifact.Tag)
	}

	var localImages []graph.Artifact
	for _, a := range artifacts {
		if isLocal, err := r.isLocalImage(a.ImageName); err != nil {
			return err
		} else if isLocal {
			localImages = append(localImages, a)
		}
	}

	if len(localImages) > 0 {
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

	if len(localImages) > 0 && r.runCtx.Cluster.LoadImages {
		err := r.loadImagesIntoCluster(ctx, out, localImages)
		if err != nil {
			return err
		}
	}

	deployOut, postDeployFn, err := deployutil.WithLogFile(time.Now().Format(deployutil.TimeFormat)+".log", out, r.runCtx.Muted())
	if err != nil {
		return err
	}

	event.DeployInProgress()
	eventV2.TaskInProgress(constants.Deploy, "Deploy to cluster")
	ctx, endTrace := instrumentation.StartTrace(ctx, "Deploy_Deploying")
	defer endTrace()

	namespaces, err := r.deployer.Deploy(ctx, deployOut, artifacts)
	postDeployFn()
	if err != nil {
		event.DeployFailed(err)
		eventV2.TaskFailed(constants.Deploy, err)
		endTrace(instrumentation.TraceEndError(err))
		return err
	}

	r.hasDeployed = true

	statusCheckOut, postStatusCheckFn, err := deployutil.WithStatusCheckLogFile(time.Now().Format(deployutil.TimeFormat)+".log", out, r.runCtx.Muted())
	defer postStatusCheckFn()
	if err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}

	event.DeployComplete()
	eventV2.TaskSucceeded(constants.Deploy)
	r.runCtx.UpdateNamespaces(namespaces)
	sErr := r.performStatusCheck(ctx, statusCheckOut)
	return sErr
}

func (r *SkaffoldRunner) loadImagesIntoCluster(ctx context.Context, out io.Writer, artifacts []graph.Artifact) error {
	currentContext, err := r.getCurrentContext()
	if err != nil {
		return err
	}

	if config.IsKindCluster(r.runCtx.GetKubeContext()) {
		kindCluster := config.KindClusterName(currentContext.Cluster)

		// With `kind`, docker images have to be loaded with the `kind` CLI.
		if err := r.loadImagesInKindNodes(ctx, out, kindCluster, artifacts); err != nil {
			return fmt.Errorf("loading images into kind nodes: %w", err)
		}
	}

	if config.IsK3dCluster(r.runCtx.GetKubeContext()) {
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

	currentContext, present := currentCfg.Contexts[r.runCtx.GetKubeContext()]
	if !present {
		return nil, fmt.Errorf("unable to get current kubernetes context: %w", err)
	}
	return currentContext, nil
}

// failIfClusterIsNotReachable checks that Kubernetes is reachable.
// This gives a clear early error when the cluster can't be reached.
func failIfClusterIsNotReachable() error {
	client, err := kubernetesclient.Client()
	if err != nil {
		return err
	}

	_, err = client.Discovery().ServerVersion()
	return err
}

func (r *SkaffoldRunner) performStatusCheck(ctx context.Context, out io.Writer) error {
	// Check if we need to perform deploy status
	enabled, err := r.runCtx.StatusCheck()
	if err != nil {
		return err
	}
	if enabled != nil && !*enabled {
		return nil
	}

	eventV2.TaskInProgress(constants.StatusCheck, "")
	ctx, endTrace := instrumentation.StartTrace(ctx, "performStatusCheck_WaitForDeploymentToStabilize")
	defer endTrace()

	start := time.Now()
	output.Default.Fprintln(out, "Waiting for deployments to stabilize...")

	s := runner.NewStatusCheck(r.runCtx, r.labeller)
	if err := s.Check(ctx, out); err != nil {
		eventV2.TaskFailed(constants.StatusCheck, err)
		return err
	}

	output.Default.Fprintln(out, "Deployments stabilized in", util.ShowHumanizeTime(time.Since(start)))
	eventV2.TaskSucceeded(constants.StatusCheck)
	return nil
}
