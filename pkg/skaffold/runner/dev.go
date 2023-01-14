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
	"strconv"
	"time"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event"
	eventV2 "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/filemon"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util/term"
	timeutil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util/time"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
)

var (
	// For testing
	fileSyncInProgress = event.FileSyncInProgress
	fileSyncFailed     = event.FileSyncFailed
	fileSyncSucceeded  = event.FileSyncSucceeded
)

func (r *SkaffoldRunner) doDev(ctx context.Context, out io.Writer) error {
	// never queue intents from user, even if they're not used
	defer r.intents.Reset()

	if r.changeSet.NeedsReload() {
		return ErrorConfigurationChanged
	}

	buildIntent, syncIntent, deployIntent := r.intents.GetIntents()
	log.Entry(ctx).Tracef("dev intents: build %t, sync %t, deploy %t\n", buildIntent, syncIntent, deployIntent)
	needsBuild := buildIntent && len(r.changeSet.NeedsRebuild()) > 0
	needsSync := syncIntent && (len(r.changeSet.NeedsResync()) > 0 || needsBuild)
	needsTest := len(r.changeSet.NeedsRetest()) > 0
	needsDeploy := deployIntent && (r.changeSet.NeedsRedeploy() || needsBuild)
	if !needsSync && !needsBuild && !needsTest && !needsDeploy {
		return nil
	}
	log.Entry(ctx).Debugf(" devloop: build %t, sync %t, deploy %t\n", needsBuild, needsSync, needsDeploy)

	r.deployer.GetLogger().Mute()
	// if any action is going to be performed, reset the monitor's changed component tracker for debouncing
	defer r.monitor.Reset()
	defer r.listener.LogWatchToUser(out)

	event.DevLoopInProgress(r.devIteration)
	eventV2.InitializeState(r.runCtx)
	eventV2.TaskInProgress(constants.DevLoop, "")
	defer func() { r.devIteration++ }()
	eventV2.LogMetaEvent()
	ctx, endTrace := instrumentation.StartTrace(ctx, "doDev_DevLoopInProgress", map[string]string{
		"devIteration": strconv.Itoa(r.devIteration),
	})

	meterUpdated := false
	if needsSync {
		childCtx, endTrace := instrumentation.StartTrace(ctx, "doDev_needsSync")
		defer func() {
			r.changeSet.ResetSync()
			r.intents.ResetSync()
		}()
		instrumentation.AddDevIteration("sync")
		meterUpdated = true
		for _, s := range r.changeSet.NeedsResync() {
			fileCount := len(s.Copy) + len(s.Delete)
			output.Default.Fprintf(out, "Syncing %d files for %s\n", fileCount, s.Image)
			fileSyncInProgress(fileCount, s.Image)

			if err := r.deployer.GetSyncer().Sync(childCtx, out, s); err != nil {
				log.Entry(ctx).Warn("Skipping deploy due to sync error:", err)
				fileSyncFailed(fileCount, s.Image, err)
				event.DevLoopFailedInPhase(r.devIteration, constants.Sync, err)
				eventV2.TaskFailed(constants.DevLoop, err)
				endTrace(instrumentation.TraceEndError(err))

				return nil
			}

			fileSyncSucceeded(fileCount, s.Image)
		}
		endTrace()
	}

	var bRes []graph.Artifact
	if needsBuild {
		childCtx, endTrace := instrumentation.StartTrace(ctx, "doDev_needsBuild")
		event.ResetStateOnBuild()
		defer func() {
			r.changeSet.ResetBuild()
			r.intents.ResetBuild()
		}()
		if !meterUpdated {
			instrumentation.AddDevIteration("build")
			meterUpdated = true
		}

		var err error
		bRes, err = r.Build(childCtx, out, r.changeSet.NeedsRebuild())
		if err != nil {
			log.Entry(ctx).Warn("Skipping test and deploy due to build error:", err)
			event.DevLoopFailedInPhase(r.devIteration, constants.Build, err)
			eventV2.TaskFailed(constants.DevLoop, err)
			endTrace(instrumentation.TraceEndError(err))
			return nil
		}
		r.changeSet.Redeploy()
		needsDeploy = deployIntent
		endTrace()
	}

	// Trigger retest when there are newly rebuilt artifacts or untested previous artifacts; and it's not explicitly skipped
	if (len(bRes) > 0 || needsTest) && r.runCtx.IsTestPhaseActive() {
		childCtx, endTrace := instrumentation.StartTrace(ctx, "doDev_needsTest")
		event.ResetStateOnTest()
		defer func() {
			r.changeSet.ResetTest()
		}()
		for _, a := range bRes {
			delete(r.changeSet.NeedsRetest(), a.ImageName)
		}
		for _, a := range r.Builds {
			if r.changeSet.NeedsRetest()[a.ImageName] {
				bRes = append(bRes, a)
			}
		}
		if err := r.Test(childCtx, out, bRes); err != nil {
			if needsDeploy {
				log.Entry(ctx).Warn("Skipping deploy due to test error:", err)
			}
			event.DevLoopFailedInPhase(r.devIteration, constants.Test, err)
			eventV2.TaskFailed(constants.DevLoop, err)
			endTrace(instrumentation.TraceEndError(err))
			return nil
		}
		endTrace()
	}

	if needsDeploy {
		childCtx, endTrace := instrumentation.StartTrace(ctx, "doDev_needsDeploy")
		event.ResetStateOnDeploy()
		defer func() {
			r.changeSet.ResetDeploy()
			r.intents.ResetDeploy()
		}()

		log.Entry(ctx).Debug("stopping accessor")
		r.deployer.GetAccessor().Stop()

		log.Entry(ctx).Debug("stopping debugger")
		r.deployer.GetDebugger().Stop()

		if !meterUpdated {
			instrumentation.AddDevIteration("deploy")
		}
		manifests, err := r.Render(childCtx, out, r.Builds, false)
		if err != nil {
			log.Entry(ctx).Warn("Skipping render due to error:", err)
			event.DevLoopFailedInPhase(r.devIteration, constants.Render, err)
			eventV2.TaskFailed(constants.DevLoop, err)
			endTrace(instrumentation.TraceEndError(err))
			return nil
		}
		r.deployManifests = manifests

		if err := r.Deploy(childCtx, out, r.Builds, manifests); err != nil {
			log.Entry(ctx).Warn("Skipping deploy due to error:", err)
			event.DevLoopFailedInPhase(r.devIteration, constants.Deploy, err)
			eventV2.TaskFailed(constants.DevLoop, err)
			endTrace(instrumentation.TraceEndError(err))
			return nil
		}

		if err := r.deployer.GetAccessor().Start(childCtx, out); err != nil {
			log.Entry(ctx).Warnf("failed to start accessor: %v", err)
		}

		if err := r.deployer.GetDebugger().Start(childCtx); err != nil {
			log.Entry(ctx).Warnf("failed to start debugger: %v", err)
		}

		endTrace()
	}
	event.DevLoopComplete(r.devIteration)
	eventV2.TaskSucceeded(constants.DevLoop)
	endTrace()
	r.deployer.GetLogger().Unmute()
	return nil
}

// Dev watches for changes and runs the skaffold build, test and deploy
// config until interrupted by the user.
func (r *SkaffoldRunner) Dev(ctx context.Context, out io.Writer, artifacts []*latest.Artifact) error {
	event.DevLoopInProgress(r.devIteration)
	eventV2.InitializeState(r.runCtx)
	eventV2.TaskInProgress(constants.DevLoop, "")
	defer func() { r.devIteration++ }()
	eventV2.LogMetaEvent()
	ctx, endTrace := instrumentation.StartTrace(ctx, "Dev", map[string]string{
		"devIteration": strconv.Itoa(r.devIteration),
	})

	// First build
	var err error
	bRes, err := r.Build(ctx, out, artifacts)
	for ; err != nil && r.runCtx.Opts.KeepRunningOnFailure; bRes, err = r.Build(ctx, out, artifacts) {
		log.Entry(ctx).Warnf("Failed to build artifacts: %v, please fix the error and press any key to continue.", err)
		errT := term.WaitForKeyPress()
		if errT != nil {
			return errT
		}
	}

	if err != nil {
		event.DevLoopFailedInPhase(r.devIteration, constants.Build, err)
		eventV2.TaskFailed(constants.DevLoop, err)
		endTrace()
		return fmt.Errorf("exiting dev mode because first build failed: %w", err)
	}
	// First test
	if r.runCtx.IsTestPhaseActive() {
		err = r.Test(ctx, out, bRes)
		for ; err != nil && r.runCtx.Opts.KeepRunningOnFailure; err = r.Test(ctx, out, bRes) {
			log.Entry(ctx).Warnf("Failed to run tests :%v, please fix the error and press any key to continue.", err)
			errT := term.WaitForKeyPress()
			if errT != nil {
				return errT
			}
		}
		if err != nil {
			event.DevLoopFailedInPhase(r.devIteration, constants.Build, err)
			eventV2.TaskFailed(constants.DevLoop, err)
			endTrace()
			return fmt.Errorf("exiting dev mode because test failed after first build: %w", err)
		}
	}

	defer r.deployer.GetLogger().Stop()
	defer r.deployer.GetDebugger().Stop()

	// Logs should be retrieved up to just before the deploy
	r.deployer.GetLogger().SetSince(time.Now())

	// First render
	manifests, err := r.Render(ctx, out, r.Builds, false)
	for ; err != nil && r.runCtx.Opts.KeepRunningOnFailure; manifests, err = r.Render(ctx, out, r.Builds, false) {
		log.Entry(ctx).Warnf("Failed to render :%v, please fix the error and press any key to continue.", err)
		errT := term.WaitForKeyPress()
		if errT != nil {
			return errT
		}
	}
	if err != nil {
		event.DevLoopFailedInPhase(r.devIteration, constants.Render, err)
		eventV2.TaskFailed(constants.DevLoop, err)
		endTrace()
		return fmt.Errorf("exiting dev mode because first render failed: %w", err)
	}

	// First deploy
	err = r.Deploy(ctx, out, r.Builds, manifests)
	for ; err != nil && r.runCtx.Opts.KeepRunningOnFailure; err = r.Deploy(ctx, out, r.Builds, manifests) {
		log.Entry(ctx).Warnf("Failed to deploy :%v, please fix the error and press any key to continue.", err)
		errT := term.WaitForKeyPress()
		if errT != nil {
			return errT
		}
		// The previous Render Stage could succeed even for kubernetes resource with unknown fields, this will lead to failure in Deploy Stage
		// users need to fix the problems in their manifests, and skaffold needs to re-render them before re-running Deploy Stage in this case.
		for manifests, err = r.Render(ctx, out, r.Builds, false); err != nil; manifests, err = r.Render(ctx, out, r.Builds, false) {
			log.Entry(ctx).Warnf("Failed to Render, please fix the error and press any key to continue. %v", err)
			errT := term.WaitForKeyPress()
			if errT != nil {
				return errT
			}
		}
	}
	if err != nil {
		event.DevLoopFailedInPhase(r.devIteration, constants.Deploy, err)
		eventV2.TaskFailed(constants.DevLoop, err)
		endTrace()
		return fmt.Errorf("exiting dev mode because first deploy failed: %w", err)
	}
	r.deployManifests = manifests

	defer r.deployer.GetAccessor().Stop()

	if err := r.deployer.GetAccessor().Start(ctx, out); err != nil {
		log.Entry(ctx).Warn("Error starting resource accessor:", err)
	}
	if err := r.deployer.GetDebugger().Start(ctx); err != nil {
		log.Entry(ctx).Warn("Error starting debug container notification:", err)
	}
	// Start printing the logs after deploy is finished
	if err := r.deployer.GetLogger().Start(ctx, out); err != nil {
		return fmt.Errorf("starting logger: %w", err)
	}

	g := getTransposeGraph(artifacts)
	// Watch artifacts
	start := time.Now()

	if len(artifacts) > 0 {
		output.Default.Fprintln(out, "Listing files to watch...")
	} else {
		output.Default.Fprintln(out, "No artifacts found to watch")
	}

	for i := range artifacts {
		artifact := artifacts[i]
		if !r.runCtx.Opts.IsTargetImage(artifact) {
			continue
		}

		output.Default.Fprintf(out, " - %s\n", artifact.ImageName)

		select {
		case <-ctx.Done():
			return context.Canceled
		default:
			if err := r.monitor.Register(
				func() ([]string, error) {
					return r.sourceDependencies.TransitiveArtifactDependencies(ctx, artifact)
				},
				func(e filemon.Events) {
					s, err := sync.NewItem(ctx, artifact, e, r.Builds, r.runCtx, len(g[artifact.ImageName]))
					switch {
					case err != nil:
						log.Entry(ctx).Warnf("error adding dirty artifact to changeset: %s", err.Error())
					case s != nil:
						r.changeSet.AddResync(s)
					default:
						r.changeSet.AddRebuild(artifact)
					}
				},
			); err != nil {
				event.DevLoopFailedWithErrorCode(r.devIteration, proto.StatusCode_DEVINIT_REGISTER_BUILD_DEPS, err)
				eventV2.TaskFailed(constants.DevLoop, err)
				endTrace()
				return fmt.Errorf("watching files for artifact %q: %w", artifact.ImageName, err)
			}
		}
	}

	// Watch test configuration
	for i := range artifacts {
		artifact := artifacts[i]
		if err := r.monitor.Register(
			func() ([]string, error) { return r.tester.TestDependencies(ctx, artifact) },
			func(filemon.Events) { r.changeSet.AddRetest(artifact) },
		); err != nil {
			event.DevLoopFailedWithErrorCode(r.devIteration, proto.StatusCode_DEVINIT_REGISTER_TEST_DEPS, err)
			eventV2.TaskFailed(constants.DevLoop, err)
			endTrace()
			return fmt.Errorf("watching test files: %w", err)
		}
	}

	// Watch render configuration
	if err := r.monitor.Register(
		r.renderer.ManifestDeps,
		func(filemon.Events) { r.changeSet.Redeploy() },
	); err != nil {
		event.DevLoopFailedWithErrorCode(r.devIteration, proto.StatusCode_DEVINIT_REGISTER_RENDER_DEPS, err)
		eventV2.TaskFailed(constants.DevLoop, err)
		endTrace()
		return fmt.Errorf("watching files for renderer: %w", err)
	}

	// Watch deployment configuration
	if err := r.monitor.Register(
		r.deployer.Dependencies,
		func(filemon.Events) { r.changeSet.Redeploy() },
	); err != nil {
		event.DevLoopFailedWithErrorCode(r.devIteration, proto.StatusCode_DEVINIT_REGISTER_DEPLOY_DEPS, err)
		eventV2.TaskFailed(constants.DevLoop, err)
		endTrace()
		return fmt.Errorf("watching files for deployer: %w", err)
	}

	// Watch Skaffold configuration
	if err := r.monitor.Register(
		func() ([]string, error) { return []string{r.runCtx.ConfigurationFile()}, nil },
		func(filemon.Events) { r.changeSet.Reload() },
	); err != nil {
		event.DevLoopFailedWithErrorCode(r.devIteration, proto.StatusCode_DEVINIT_REGISTER_CONFIG_DEP, err)
		eventV2.TaskFailed(constants.DevLoop, err)
		endTrace()
		return fmt.Errorf("watching skaffold configuration %q: %w", r.runCtx.ConfigurationFile(), err)
	}

	log.Entry(ctx).Infoln("List generated in", timeutil.Humanize(time.Since(start)))

	// Init Sync State
	if err := sync.Init(ctx, artifacts); err != nil {
		event.DevLoopFailedWithErrorCode(r.devIteration, proto.StatusCode_SYNC_INIT_ERROR, err)
		eventV2.TaskFailed(constants.DevLoop, err)
		endTrace()
		return fmt.Errorf("exiting dev mode because initializing sync state failed: %w", err)
	}

	output.Yellow.Fprintln(out, "Press Ctrl+C to exit")

	event.DevLoopComplete(r.devIteration)
	eventV2.TaskSucceeded(constants.DevLoop)
	endTrace()
	r.devIteration++
	return r.listener.WatchForChanges(ctx, out, func() error {
		return r.doDev(ctx, out)
	})
}

// graph represents the artifact graph
type devGraph map[string][]*latest.Artifact

// getTransposeGraph builds the transpose of the graph represented by the artifacts slice, with edges directed from required artifact to the dependent artifact.
func getTransposeGraph(artifacts []*latest.Artifact) devGraph {
	g := make(map[string][]*latest.Artifact)
	for _, a := range artifacts {
		for _, d := range a.Dependencies {
			g[d.ImageName] = append(g[d.ImageName], a)
		}
	}
	return g
}
