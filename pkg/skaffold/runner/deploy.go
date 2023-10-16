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

package runner

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	deployutil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event"
	eventV2 "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
)

// DeployAndLog deploys a list of already built artifacts and optionally show the logs.
func (r *SkaffoldRunner) DeployAndLog(ctx context.Context, out io.Writer, artifacts []graph.Artifact, list manifest.ManifestListByConfig) error {
	defer r.deployer.GetLogger().Stop()

	// Logs should be retrieved up to just before the deploy
	r.deployer.GetLogger().SetSince(time.Now())
	// First deploy
	if err := r.Deploy(ctx, out, artifacts, list); err != nil {
		return err
	}

	defer r.deployer.GetAccessor().Stop()

	if err := r.deployer.GetAccessor().Start(ctx, out); err != nil {
		log.Entry(ctx).Warn("Error starting port forwarding:", err)
	}

	// Start printing the logs after deploy is finished
	if err := r.deployer.GetLogger().Start(ctx, out); err != nil {
		return fmt.Errorf("starting logger: %w", err)
	}

	if r.runCtx.Tail() || r.runCtx.PortForward() {
		output.Yellow.Fprintln(out, "Press Ctrl+C to exit")
		<-ctx.Done()
	}

	return nil
}

func (r *SkaffoldRunner) Deploy(ctx context.Context, out io.Writer, artifacts []graph.Artifact, list manifest.ManifestListByConfig) error {
	defer r.deployer.GetStatusMonitor().Reset()

	out, ctx = output.WithEventContext(ctx, out, constants.Deploy, constants.SubtaskIDNone)

	if len(artifacts) > 0 {
		output.Default.Fprintln(out, "Tags used in deployment:")
	}

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
		log.Entry(ctx).Debug(`Local images can't be referenced by digest.
They are tagged and referenced by a unique, local only, tag instead.
See https://skaffold.dev/docs/pipeline-stages/taggers/#how-tagging-works`)
	}

	deployOut, postDeployFn, err := deployutil.WithLogFile(time.Now().Format(deployutil.TimeFormat)+".log", out, r.runCtx.Muted())
	if err != nil {
		return err
	}

	event.DeployInProgress()
	eventV2.TaskInProgress(constants.Deploy, "Deploy to cluster")
	ctx, endTrace := instrumentation.StartTrace(ctx, "Deploy_Deploying")
	defer endTrace()

	// we only want to register images that are local AND were built by this runner OR forced to load via flag
	var localAndBuiltImages []graph.Artifact
	for _, image := range localImages {
		if r.runCtx.ForceLoadImages() || r.wasBuilt(image.Tag) {
			localAndBuiltImages = append(localAndBuiltImages, image)
		}
	}

	r.deployer.RegisterLocalImages(localAndBuiltImages)
	err = r.deployer.Deploy(ctx, deployOut, artifacts, list)
	r.deployManifests = list // set even if deploy may have failed, because we want to cleanup any partially created resources
	postDeployFn()
	if err != nil {
		event.DeployFailed(err)
		eventV2.TaskFailed(constants.Deploy, err)
		endTrace(instrumentation.TraceEndError(err))
		return err
	}

	statusCheckOut, postStatusCheckFn, err := deployutil.WithStatusCheckLogFile(time.Now().Format(deployutil.TimeFormat)+".log", out, r.runCtx.Muted())
	defer postStatusCheckFn()
	if err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}

	event.DeployComplete()
	if !r.runCtx.IterativeStatusCheck() {
		// run final aggregated status check only if iterative status check is turned off.
		if err = r.deployer.GetStatusMonitor().Check(ctx, statusCheckOut); err != nil {
			eventV2.TaskFailed(constants.Deploy, err)
			return err
		}
	}
	eventV2.TaskSucceeded(constants.Deploy)
	return nil
}

func (r *SkaffoldRunner) wasBuilt(tag string) bool {
	for _, built := range r.Builds {
		if built.Tag == tag {
			return true
		}
	}
	return false
}
