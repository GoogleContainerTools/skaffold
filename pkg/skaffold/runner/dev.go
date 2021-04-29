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
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	eventV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/filemon"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/portforward"
	latest_v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
)

// ErrorConfigurationChanged is a special error that's returned when the skaffold configuration was changed.
var ErrorConfigurationChanged = errors.New("configuration changed")

var (
	// For testing
	fileSyncInProgress = event.FileSyncInProgress
	fileSyncFailed     = event.FileSyncFailed
	fileSyncSucceeded  = event.FileSyncSucceeded
)

func (r *SkaffoldRunner) doDev(ctx context.Context, out io.Writer, logger *kubernetes.LogAggregator, forwarderManager portforward.Forwarder) error {
	// never queue intents from user, even if they're not used
	defer r.intents.reset()

	if r.changeSet.needsReload {
		return ErrorConfigurationChanged
	}

	buildIntent, syncIntent, deployIntent := r.intents.GetIntents()
	logrus.Tracef("dev intents: build %t, sync %t, deploy %t\n", buildIntent, syncIntent, deployIntent)
	needsSync := syncIntent && len(r.changeSet.needsResync) > 0
	needsBuild := buildIntent && len(r.changeSet.needsRebuild) > 0
	needsTest := len(r.changeSet.needsRetest) > 0
	needsDeploy := deployIntent && r.changeSet.needsRedeploy
	if !needsSync && !needsBuild && !needsTest && !needsDeploy {
		return nil
	}

	logger.Mute()
	// if any action is going to be performed, reset the monitor's changed component tracker for debouncing
	defer r.monitor.Reset()
	defer r.listener.LogWatchToUser(out)
	event.DevLoopInProgress(r.devIteration)
	eventV2.TaskInProgress(constants.DevLoop)
	defer func() { r.devIteration++ }()

	meterUpdated := false
	if needsSync {
		defer func() {
			r.changeSet.resetSync()
			r.intents.resetSync()
		}()
		instrumentation.AddDevIteration("sync")
		meterUpdated = true
		for _, s := range r.changeSet.needsResync {
			fileCount := len(s.Copy) + len(s.Delete)
			color.Default.Fprintf(out, "Syncing %d files for %s\n", fileCount, s.Image)
			fileSyncInProgress(fileCount, s.Image)

			if err := r.syncer.Sync(ctx, s); err != nil {
				logrus.Warnln("Skipping deploy due to sync error:", err)
				fileSyncFailed(fileCount, s.Image, err)
				event.DevLoopFailedInPhase(r.devIteration, constants.Sync, err)
				eventV2.TaskFailed(constants.DevLoop, err)
				return nil
			}

			fileSyncSucceeded(fileCount, s.Image)
		}
	}

	var bRes []graph.Artifact
	if needsBuild {
		event.ResetStateOnBuild()
		defer func() {
			r.changeSet.resetBuild()
			r.intents.resetBuild()
		}()
		if !meterUpdated {
			instrumentation.AddDevIteration("build")
			meterUpdated = true
		}

		var err error
		bRes, err = r.Build(ctx, out, r.changeSet.needsRebuild)
		if err != nil {
			logrus.Warnln("Skipping test and deploy due to build error:", err)
			event.DevLoopFailedInPhase(r.devIteration, constants.Build, err)
			eventV2.TaskFailed(constants.DevLoop, err)
			return nil
		}
		r.changeSet.needsRedeploy = true
		needsDeploy = deployIntent
	}

	// Trigger retest when there are newly rebuilt artifacts or untested previous artifacts; and it's not explicitly skipped
	if (len(bRes) > 0 || needsTest) && !r.runCtx.SkipTests() {
		event.ResetStateOnTest()
		defer func() {
			r.changeSet.resetTest()
		}()
		for _, a := range bRes {
			delete(r.changeSet.needsRetest, a.ImageName)
		}
		for _, a := range r.builds {
			if r.changeSet.needsRetest[a.ImageName] {
				bRes = append(bRes, a)
			}
		}
		if err := r.Test(ctx, out, bRes); err != nil {
			if needsDeploy {
				logrus.Warnln("Skipping deploy due to test error:", err)
			}
			event.DevLoopFailedInPhase(r.devIteration, constants.Test, err)
			eventV2.TaskFailed(constants.DevLoop, err)
			return nil
		}
	}

	if needsDeploy {
		event.ResetStateOnDeploy()
		defer func() {
			r.changeSet.resetDeploy()
			r.intents.resetDeploy()
		}()

		forwarderManager.Stop()
		if !meterUpdated {
			instrumentation.AddDevIteration("deploy")
		}
		if err := r.Deploy(ctx, out, r.builds); err != nil {
			logrus.Warnln("Skipping deploy due to error:", err)
			event.DevLoopFailedInPhase(r.devIteration, constants.Deploy, err)
			eventV2.TaskFailed(constants.DevLoop, err)
			return nil
		}
		if err := forwarderManager.Start(ctx, r.runCtx.GetNamespaces()); err != nil {
			logrus.Warnln("Port forwarding failed:", err)
		}
	}
	event.DevLoopComplete(r.devIteration)
	eventV2.TaskSucceeded(constants.DevLoop)
	logger.Unmute()
	return nil
}

// Dev watches for changes and runs the skaffold build, test and deploy
// config until interrupted by the user.
func (r *SkaffoldRunner) Dev(ctx context.Context, out io.Writer, artifacts []*latest_v1.Artifact) error {
	event.DevLoopInProgress(r.devIteration)
	eventV2.TaskInProgress(constants.DevLoop)
	defer func() { r.devIteration++ }()
	g := getTransposeGraph(artifacts)
	// Watch artifacts
	start := time.Now()
	color.Default.Fprintln(out, "Listing files to watch...")

	for i := range artifacts {
		artifact := artifacts[i]
		if !r.runCtx.Opts.IsTargetImage(artifact) {
			continue
		}

		color.Default.Fprintf(out, " - %s\n", artifact.ImageName)

		select {
		case <-ctx.Done():
			return context.Canceled
		default:
			if err := r.monitor.Register(
				func() ([]string, error) {
					return r.sourceDependencies.TransitiveArtifactDependencies(ctx, artifact)
				},
				func(e filemon.Events) {
					s, err := sync.NewItem(ctx, artifact, e, r.builds, r.runCtx, len(g[artifact.ImageName]))
					switch {
					case err != nil:
						logrus.Warnf("error adding dirty artifact to changeset: %s", err.Error())
					case s != nil:
						r.changeSet.AddResync(s)
					default:
						r.changeSet.AddRebuild(artifact)
					}
				},
			); err != nil {
				event.DevLoopFailedWithErrorCode(r.devIteration, proto.StatusCode_DEVINIT_REGISTER_BUILD_DEPS, err)
				eventV2.TaskFailed(constants.DevLoop, err)
				return fmt.Errorf("watching files for artifact %q: %w", artifact.ImageName, err)
			}
		}
	}

	// Watch test configuration
	for i := range artifacts {
		artifact := artifacts[i]
		if err := r.monitor.Register(
			func() ([]string, error) { return r.tester.TestDependencies(artifact) },
			func(filemon.Events) { r.changeSet.AddRetest(artifact) },
		); err != nil {
			event.DevLoopFailedWithErrorCode(r.devIteration, proto.StatusCode_DEVINIT_REGISTER_TEST_DEPS, err)
			eventV2.TaskFailed(constants.DevLoop, err)
			return fmt.Errorf("watching test files: %w", err)
		}
	}

	// Watch deployment configuration
	if err := r.monitor.Register(
		r.deployer.Dependencies,
		func(filemon.Events) { r.changeSet.needsRedeploy = true },
	); err != nil {
		event.DevLoopFailedWithErrorCode(r.devIteration, proto.StatusCode_DEVINIT_REGISTER_DEPLOY_DEPS, err)
		eventV2.TaskFailed(constants.DevLoop, err)
		return fmt.Errorf("watching files for deployer: %w", err)
	}

	// Watch Skaffold configuration
	if err := r.monitor.Register(
		func() ([]string, error) { return []string{r.runCtx.ConfigurationFile()}, nil },
		func(filemon.Events) { r.changeSet.needsReload = true },
	); err != nil {
		event.DevLoopFailedWithErrorCode(r.devIteration, proto.StatusCode_DEVINIT_REGISTER_CONFIG_DEP, err)
		eventV2.TaskFailed(constants.DevLoop, err)
		return fmt.Errorf("watching skaffold configuration %q: %w", r.runCtx.ConfigurationFile(), err)
	}

	logrus.Infoln("List generated in", util.ShowHumanizeTime(time.Since(start)))

	// Init Sync State
	if err := sync.Init(ctx, artifacts); err != nil {
		event.DevLoopFailedWithErrorCode(r.devIteration, proto.StatusCode_SYNC_INIT_ERROR, err)
		eventV2.TaskFailed(constants.DevLoop, err)
		return fmt.Errorf("exiting dev mode because initializing sync state failed: %w", err)
	}

	// First build
	bRes, err := r.Build(ctx, out, artifacts)
	if err != nil {
		event.DevLoopFailedInPhase(r.devIteration, constants.Build, err)
		eventV2.TaskFailed(constants.DevLoop, err)
		return fmt.Errorf("exiting dev mode because first build failed: %w", err)
	}
	// First test
	if !r.runCtx.SkipTests() {
		if err = r.Test(ctx, out, bRes); err != nil {
			event.DevLoopFailedInPhase(r.devIteration, constants.Build, err)
			eventV2.TaskFailed(constants.DevLoop, err)
			return fmt.Errorf("exiting dev mode because test failed after first build: %w", err)
		}
	}

	logger := r.createLogger(out, bRes)
	defer logger.Stop()

	debugContainerManager := r.createContainerManager()
	defer debugContainerManager.Stop()

	// Logs should be retrieved up to just before the deploy
	logger.SetSince(time.Now())

	// First deploy
	if err := r.Deploy(ctx, out, r.builds); err != nil {
		event.DevLoopFailedInPhase(r.devIteration, constants.Deploy, err)
		eventV2.TaskFailed(constants.DevLoop, err)
		return fmt.Errorf("exiting dev mode because first deploy failed: %w", err)
	}

	forwarderManager := r.createForwarder(out)
	defer forwarderManager.Stop()

	if err := forwarderManager.Start(ctx, r.runCtx.GetNamespaces()); err != nil {
		logrus.Warnln("Error starting port forwarding:", err)
	}
	if err := debugContainerManager.Start(ctx, r.runCtx.GetNamespaces()); err != nil {
		logrus.Warnln("Error starting debug container notification:", err)
	}
	// Start printing the logs after deploy is finished
	if err := logger.Start(ctx, r.runCtx.GetNamespaces()); err != nil {
		return fmt.Errorf("starting logger: %w", err)
	}

	color.Yellow.Fprintln(out, "Press Ctrl+C to exit")

	event.DevLoopComplete(r.devIteration)
	eventV2.TaskSucceeded(constants.DevLoop)
	r.devIteration++
	return r.listener.WatchForChanges(ctx, out, func() error {
		return r.doDev(ctx, out, logger, forwarderManager)
	})
}

// graph represents the artifact graph
type devGraph map[string][]*latest_v1.Artifact

// getTransposeGraph builds the transpose of the graph represented by the artifacts slice, with edges directed from required artifact to the dependent artifact.
func getTransposeGraph(artifacts []*latest_v1.Artifact) devGraph {
	g := make(map[string][]*latest_v1.Artifact)
	for _, a := range artifacts {
		for _, d := range a.Dependencies {
			g[d.ImageName] = append(g[d.ImageName], a)
		}
	}
	return g
}
