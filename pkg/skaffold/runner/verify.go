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

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	deployutil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/util"
	eventV2 "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
)

// VerifyAndLog deploys a list of already built artifacts and optionally show the logs.
func (r *SkaffoldRunner) VerifyAndLog(ctx context.Context, out io.Writer, artifacts []graph.Artifact) error {
	defer r.verifier.GetLogger().Stop()

	// Logs should be retrieved up to just before the verify tests run
	r.verifier.GetLogger().SetSince(time.Now())

	// Start logger immediately as it needs to be running to get test logs immediately
	if err := r.verifier.GetLogger().Start(ctx, out); err != nil {
		return fmt.Errorf("starting logger: %w", err)
	}

	// Run verify
	if err := r.Verify(ctx, out, artifacts); err != nil {
		return err
	}

	return nil
}

func (r *SkaffoldRunner) Verify(ctx context.Context, out io.Writer, artifacts []graph.Artifact) error {
	defer r.verifier.GetStatusMonitor().Reset()

	out, ctx = output.WithEventContext(ctx, out, constants.Verify, constants.SubtaskIDNone)

	if len(artifacts) > 0 {
		output.Default.Fprintln(out, "Tags used in verification:")

		for _, artifact := range artifacts {
			output.Default.Fprintf(out, " - %s -> ", artifact.ImageName)
			fmt.Fprintln(out, artifact.Tag)
		}
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

	r.verifier.RegisterLocalImages(localImages)
	err = r.verifier.Verify(ctx, deployOut, artifacts)
	postDeployFn()
	if err != nil {
		eventV2.TaskFailed(constants.Verify, err)
		endTrace(instrumentation.TraceEndError(err))
		return err
	}

	_, postStatusCheckFn, err := deployutil.WithStatusCheckLogFile(time.Now().Format(deployutil.TimeFormat)+".log", out, r.runCtx.Muted())
	defer postStatusCheckFn()
	if err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}
	eventV2.TaskSucceeded(constants.Verify)
	return nil
}
