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
package v1

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	deployutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	eventV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	kubernetesclient "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/logger"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/portforward"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// DeployAndLog deploys a list of already built artifacts and optionally show the logs.
func (r *SkaffoldRunner) DeployAndLog(ctx context.Context, out io.Writer, artifacts []graph.Artifact) error {
	eventV2.TaskInProgress(constants.Deploy)

	// Update which images are logged.
	r.AddTagsToPodSelector(artifacts)

	if !r.RunCtx.Tail() {
		return nil
	}

	var imageNames []string
	for _, artifact := range artifacts {
		imageNames = append(imageNames, artifact.Tag)
	}

	logger := logger.NewLogAggregator(out, r.kubectlCLI, imageNames, r.podSelector, r.RunCtx)
	defer logger.Stop()

	// Logs should be retrieved up to just before the deploy
	logger.SetSince(time.Now())
	// First deploy
	if err := r.Deploy(ctx, out, artifacts); err != nil {
		eventV2.TaskFailed(constants.Deploy, err)
		return err
	}

	if !r.RunCtx.PortForward() {
		return nil
	}
	forwarderManager := portforward.NewForwarderManager(out,
		r.kubectlCLI,
		r.podSelector,
		r.labeller.RunIDSelector(),
		r.RunCtx.Mode(),
		r.RunCtx.Opts.PortForward,
		r.RunCtx.PortForwardResources())

	defer forwarderManager.Stop()

	if err := forwarderManager.Start(ctx, r.RunCtx.GetNamespaces()); err != nil {
		logrus.Warnln("Error starting port forwarding:", err)
	}

	// Start printing the logs after deploy is finished
	if err := logger.Start(ctx, r.RunCtx.GetNamespaces()); err != nil {
		eventV2.TaskFailed(constants.Deploy, err)
		return fmt.Errorf("starting logger: %w", err)
	}

	if r.RunCtx.Tail() || r.RunCtx.PortForward() {
		color.Yellow.Fprintln(out, "Press Ctrl+C to exit")
		<-ctx.Done()
	}

	eventV2.TaskSucceeded(constants.Deploy)
	return nil
}

func (r *SkaffoldRunner) Deploy(ctx context.Context, out io.Writer, artifacts []graph.Artifact) error {
	if r.RunCtx.RenderOnly() {
		return r.Render(ctx, out, artifacts, false, r.RunCtx.RenderOutput())
	}

	color.Default.Fprintln(out, "Tags used in deployment:")

	for _, artifact := range artifacts {
		color.Default.Fprintf(out, " - %s -> ", artifact.ImageName)
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

	if len(localImages) > 0 && r.RunCtx.Cluster.LoadImages {
		err := r.LoadImagesIntoCluster(ctx, out, localImages)
		if err != nil {
			return err
		}
	}

	deployOut, postDeployFn, err := deployutil.WithLogFile(time.Now().Format(deployutil.TimeFormat)+".log", out, r.RunCtx.Muted())
	if err != nil {
		return err
	}

	event.DeployInProgress()
	eventV2.TaskInProgress(constants.Deploy)
	namespaces, err := r.Deployer.Deploy(ctx, deployOut, artifacts)
	postDeployFn()
	if err != nil {
		event.DeployFailed(err)
		eventV2.TaskFailed(constants.Deploy, err)
		return err
	}

	r.hasDeployed = true

	statusCheckOut, postStatusCheckFn, err := deployutil.WithStatusCheckLogFile(time.Now().Format(deployutil.TimeFormat)+".log", out, r.RunCtx.Muted())
	defer postStatusCheckFn()
	if err != nil {
		return err
	}
	event.DeployComplete()
	eventV2.TaskSucceeded(constants.Deploy)
	r.RunCtx.UpdateNamespaces(namespaces)
	sErr := r.performStatusCheck(ctx, statusCheckOut)
	return sErr
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
	enabled, err := r.RunCtx.StatusCheck()
	if err != nil {
		return err
	}
	if enabled != nil && !*enabled {
		return nil
	}

	eventV2.TaskInProgress(constants.StatusCheck)
	start := time.Now()
	color.Default.Fprintln(out, "Waiting for deployments to stabilize...")

	s := newStatusCheck(r.RunCtx, r.labeller)
	if err := s.Check(ctx, out); err != nil {
		eventV2.TaskFailed(constants.StatusCheck, err)
		return err
	}

	color.Default.Fprintln(out, "Deployments stabilized in", util.ShowHumanizeTime(time.Since(start)))
	eventV2.TaskSucceeded(constants.StatusCheck)
	return nil
}
