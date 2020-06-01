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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/filemon"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/portforward"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/proto"
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
	if r.changeSet.needsReload {
		return ErrorConfigurationChanged
	}

	buildIntent, syncIntent, deployIntent := r.intents.GetIntents()
	needsSync := syncIntent && len(r.changeSet.needsResync) > 0
	needsBuild := buildIntent && len(r.changeSet.needsRebuild) > 0
	needsDeploy := deployIntent && r.changeSet.needsRedeploy
	if !needsSync && !needsBuild && !needsDeploy {
		return nil
	}

	logger.Mute()
	// if any action is going to be performed, reset the monitor's changed component tracker for debouncing
	defer r.monitor.Reset()
	defer r.listener.LogWatchToUser(out)
	event.DevLoopInProgress(r.devIteration)
	defer func() { r.devIteration++ }()
	if needsSync {
		defer func() {
			r.changeSet.resetSync()
			r.intents.resetSync()
		}()

		for _, s := range r.changeSet.needsResync {
			fileCount := len(s.Copy) + len(s.Delete)
			color.Default.Fprintf(out, "Syncing %d files for %s\n", fileCount, s.Image)
			fileSyncInProgress(fileCount, s.Image)

			if err := r.syncer.Sync(ctx, s); err != nil {
				logrus.Warnln("Skipping deploy due to sync error:", err)
				fileSyncFailed(fileCount, s.Image, err)
				event.DevLoopFailedInPhase(r.devIteration, sErrors.FileSync, err)
				return nil
			}

			fileSyncSucceeded(fileCount, s.Image)
		}
	}

	if needsBuild {
		event.ResetStateOnBuild()
		defer func() {
			r.changeSet.resetBuild()
			r.intents.resetBuild()
		}()

		if _, err := r.BuildAndTest(ctx, out, r.changeSet.needsRebuild); err != nil {
			logrus.Warnln("Skipping deploy due to error:", err)
			event.DevLoopFailedInPhase(r.devIteration, sErrors.Build, err)
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
		if err := r.Deploy(ctx, out, r.builds); err != nil {
			logrus.Warnln("Skipping deploy due to error:", err)
			event.DevLoopFailedInPhase(r.devIteration, sErrors.Deploy, err)
			return nil
		}
		if err := forwarderManager.Start(ctx); err != nil {
			logrus.Warnln("Port forwarding failed:", err)
		}
	}
	event.DevLoopComplete(r.devIteration)
	logger.Unmute()
	return nil
}

// Dev watches for changes and runs the skaffold build and deploy
// config until interrupted by the user.
func (r *SkaffoldRunner) Dev(ctx context.Context, out io.Writer, artifacts []*latest.Artifact) error {
	event.DevLoopInProgress(r.devIteration)
	defer func() { r.devIteration++ }()

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
					return build.DependenciesForArtifact(ctx, artifact, r.runCtx.InsecureRegistries)
				},
				func(e filemon.Events) {
					s, err := sync.NewItem(ctx, artifact, e, r.builds, r.runCtx.InsecureRegistries)
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
				return fmt.Errorf("watching files for artifact %q: %w", artifact.ImageName, err)
			}
		}
	}

	// Watch test configuration
	if err := r.monitor.Register(
		r.tester.TestDependencies,
		func(filemon.Events) { r.changeSet.needsRedeploy = true },
	); err != nil {
		event.DevLoopFailedWithErrorCode(r.devIteration, proto.StatusCode_DEVINIT_REGISTER_TEST_DEPS, err)
		return fmt.Errorf("watching test files: %w", err)
	}

	// Watch deployment configuration
	if err := r.monitor.Register(
		r.deployer.Dependencies,
		func(filemon.Events) { r.changeSet.needsRedeploy = true },
	); err != nil {
		event.DevLoopFailedWithErrorCode(r.devIteration, proto.StatusCode_DEVINIT_REGISTER_DEPLOY_DEPS, err)
		return fmt.Errorf("watching files for deployer: %w", err)
	}

	// Watch Skaffold configuration
	if err := r.monitor.Register(
		func() ([]string, error) { return []string{r.runCtx.Opts.ConfigurationFile}, nil },
		func(filemon.Events) { r.changeSet.needsReload = true },
	); err != nil {
		event.DevLoopFailedWithErrorCode(r.devIteration, proto.StatusCode_DEVINIT_REGISTER_CONFIG_DEP, err)
		return fmt.Errorf("watching skaffold configuration %q: %w", r.runCtx.Opts.ConfigurationFile, err)
	}

	logrus.Infoln("List generated in", time.Since(start))

	// Init Sync State
	if err := sync.Init(ctx, artifacts); err != nil {
		event.DevLoopFailedWithErrorCode(r.devIteration, proto.StatusCode_SYNC_INIT_ERROR, err)
		return fmt.Errorf("exiting dev mode because initializing sync state failed: %w", err)
	}

	// First build
	bRes, err := r.BuildAndTest(ctx, out, artifacts)
	if err != nil {
		event.DevLoopFailedInPhase(r.devIteration, sErrors.Build, err)
		return fmt.Errorf("exiting dev mode because first build failed: %w", err)
	}

	logger := r.createLogger(out, bRes)
	defer logger.Stop()

	forwarderManager := r.createForwarder(out)
	defer forwarderManager.Stop()

	debugContainerManager := r.createContainerManager()
	defer debugContainerManager.Stop()

	// Logs should be retrieved up to just before the deploy
	logger.SetSince(time.Now())

	// First deploy
	if err := r.Deploy(ctx, out, r.builds); err != nil {
		event.DevLoopFailedInPhase(r.devIteration, sErrors.Deploy, err)
		return fmt.Errorf("exiting dev mode because first deploy failed: %w", err)
	}

	if err := forwarderManager.Start(ctx); err != nil {
		logrus.Warnln("Error starting port forwarding:", err)
	}
	if err := debugContainerManager.Start(ctx); err != nil {
		logrus.Warnln("Error starting debug container notification:", err)
	}
	// Start printing the logs after deploy is finished
	if err := logger.Start(ctx); err != nil {
		return fmt.Errorf("starting logger: %w", err)
	}

	color.Yellow.Fprintln(out, "Press Ctrl+C to exit")

	event.DevLoopComplete(0)
	return r.listener.WatchForChanges(ctx, out, func() error {
		return r.doDev(ctx, out, logger, forwarderManager)
	})
}
