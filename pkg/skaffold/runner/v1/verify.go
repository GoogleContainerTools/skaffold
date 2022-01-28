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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	deployutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/util"

	eventV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
)

// VerifyAndLog deploys a list of already built artifacts and optionally show the logs.
func (r *SkaffoldRunner) VerifyAndLog(ctx context.Context, out io.Writer, artifacts []graph.Artifact) error {
	defer r.verifier.GetLogger().Stop()

	// Logs should be retrieved up to just before the deploys
	r.verifier.GetLogger().SetSince(time.Now())

	// Start printing the logs after deploy is finished
	if err := r.verifier.GetLogger().Start(ctx, out); err != nil {
		return fmt.Errorf("starting logger: %w", err)
	}

	// First deploy
	if err := r.Verify(ctx, out, artifacts); err != nil {
		return err
	}

	defer r.verifier.GetAccessor().Stop()

	if err := r.verifier.GetAccessor().Start(ctx, out); err != nil {
		log.Entry(ctx).Warn("Error starting port forwarding:", err)
	}

	return nil
}

func (r *SkaffoldRunner) Verify(ctx context.Context, out io.Writer, artifacts []graph.Artifact) error {
	defer r.verifier.GetStatusMonitor().Reset()

	out, ctx = output.WithEventContext(ctx, out, constants.Verify, constants.SubtaskIDNone)

	output.Default.Fprintln(out, "Tags used in verification:")

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

	eventV2.TaskInProgress(constants.Verify, "Running verification test(s) locally")
	ctx, endTrace := instrumentation.StartTrace(ctx, "Verify_Verifying")
	defer endTrace()

	// we only want to register images that are local AND were built by this runner OR forced to load via flag
	var localAndBuiltImages []graph.Artifact
	for _, image := range localImages {
		if r.runCtx.ForceLoadImages() || r.wasBuilt(image.Tag) {
			localAndBuiltImages = append(localAndBuiltImages, image)
		}
	}

	r.verifier.RegisterLocalImages(localAndBuiltImages)
	// TODO(aaron-prindle) need artifacts to be correct below here vvvvv
	err = r.verifier.Deploy(ctx, deployOut, artifacts)
	r.hasDeployed = true // set even if deploy may have failed, because we want to cleanup any partially created resources
	postDeployFn()
	if err != nil {
		eventV2.TaskFailed(constants.Verify, err)
		endTrace(instrumentation.TraceEndError(err))
		return err
	}

	statusCheckOut, postStatusCheckFn, err := deployutil.WithStatusCheckLogFile(time.Now().Format(deployutil.TimeFormat)+".log", out, r.runCtx.Muted())
	defer postStatusCheckFn()
	if err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}

	if !r.runCtx.Opts.IterativeStatusCheck {
		// run final aggregated status check only if iterative status check is turned off.
		if err = r.verifier.GetStatusMonitor().Check(ctx, statusCheckOut); err != nil {
			eventV2.TaskFailed(constants.Verify, err)
			return err
		}
	}
	eventV2.TaskSucceeded(constants.Verify)
	return nil
}
