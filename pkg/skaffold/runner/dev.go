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
	"io"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/filemon"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ErrorConfigurationChanged is a special error that's returned when the skaffold configuration was changed.
var ErrorConfigurationChanged = errors.New("configuration changed")

func (r *SkaffoldRunner) doDev(ctx context.Context, out io.Writer) error {
	actionPerformed := false

	// acquire the intents
	buildIntent, syncIntent, deployIntent := r.intents.GetIntents()

	if (r.changeSet.needsRedeploy && deployIntent) ||
		(len(r.changeSet.needsRebuild) > 0 && buildIntent) ||
		(len(r.changeSet.needsResync) > 0 && syncIntent) {
		r.logger.Mute()
		// if any action is going to be performed, reset the monitor's changed component tracker for debouncing
		defer r.monitor.Reset()
		defer r.listener.LogWatchToUser(out)
	}

	switch {
	case r.changeSet.needsReload:
		r.changeSet.resetReload()
		return ErrorConfigurationChanged
	case len(r.changeSet.needsResync) > 0 && syncIntent:
		defer func() {
			r.changeSet.resetSync()
			r.intents.resetSync()
		}()
		actionPerformed = true
		for _, s := range r.changeSet.needsResync {
			color.Default.Fprintf(out, "Syncing %d files for %s\n", len(s.Copy)+len(s.Delete), s.Image)

			if err := r.syncer.Sync(ctx, s); err != nil {
				r.changeSet.reset()
				logrus.Warnln("Skipping deploy due to sync error:", err)
				return nil
			}
		}
	case len(r.changeSet.needsRebuild) > 0 && buildIntent:
		defer func() {
			r.changeSet.resetBuild()
			r.intents.resetBuild()
		}()
		// this linter apparently doesn't understand fallthroughs
		//nolint:ineffassign
		actionPerformed = true
		if _, err := r.BuildAndTest(ctx, out, r.changeSet.needsRebuild); err != nil {
			r.changeSet.reset()
			logrus.Warnln("Skipping deploy due to error:", err)
			return nil
		}
		r.changeSet.needsRedeploy = true
		fallthrough // always try a redeploy after a successful build
	case r.changeSet.needsRedeploy && deployIntent:
		if !deployIntent {
			// in case we fell through, but haven't received the go ahead to deploy
			r.logger.Unmute()
			return nil
		}
		actionPerformed = true
		r.forwarderManager.Stop()
		defer func() {
			r.changeSet.reset()
			r.intents.resetDeploy()
		}()
		if err := r.Deploy(ctx, out, r.builds); err != nil {
			logrus.Warnln("Skipping deploy due to error:", err)
			return nil
		}
		if err := r.forwarderManager.Start(ctx); err != nil {
			logrus.Warnln("Port forwarding failed due to error:", err)
		}
	}

	if actionPerformed {
		r.logger.Unmute()
	}
	return nil
}

// Dev watches for changes and runs the skaffold build and deploy
// config until interrupted by the user.
func (r *SkaffoldRunner) Dev(ctx context.Context, out io.Writer, artifacts []*latest.Artifact) error {
	r.createLogger(out, artifacts)
	defer r.logger.Stop()

	kubectlCLI := kubectl.NewFromRunContext(r.runCtx)
	r.createForwarder(out, kubectlCLI)
	defer r.forwarderManager.Stop()

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
				func() ([]string, error) { return r.builder.DependenciesForArtifact(ctx, artifact) },
				func(e filemon.Events) {
					syncMap := func() (map[string][]string, error) { return r.builder.SyncMap(ctx, artifact) }
					s, err := sync.NewItem(artifact, e, r.builds, r.runCtx.InsecureRegistries, syncMap)
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
				return errors.Wrapf(err, "watching files for artifact %s", artifact.ImageName)
			}
		}
	}

	// Watch test configuration
	if err := r.monitor.Register(
		r.tester.TestDependencies,
		func(filemon.Events) { r.changeSet.needsRedeploy = true },
	); err != nil {
		return errors.Wrap(err, "watching test files")
	}

	// Watch deployment configuration
	if err := r.monitor.Register(
		r.deployer.Dependencies,
		func(filemon.Events) { r.changeSet.needsRedeploy = true },
	); err != nil {
		return errors.Wrap(err, "watching files for deployer")
	}

	// Watch Skaffold configuration
	if err := r.monitor.Register(
		func() ([]string, error) { return []string{r.runCtx.Opts.ConfigurationFile}, nil },
		func(filemon.Events) { r.changeSet.needsReload = true },
	); err != nil {
		return errors.Wrapf(err, "watching skaffold configuration %s", r.runCtx.Opts.ConfigurationFile)
	}

	color.Default.Fprintln(out, "List generated in", time.Since(start))

	// First build
	if _, err := r.BuildAndTest(ctx, out, artifacts); err != nil {
		return errors.Wrap(err, "exiting dev mode because first build failed")
	}

	// Logs should be retrieved up to just before the deploy
	r.logger.SetSince(time.Now())

	// First deploy
	if err := r.Deploy(ctx, out, r.builds); err != nil {
		return errors.Wrap(err, "exiting dev mode because first deploy failed")
	}

	if err := r.forwarderManager.Start(ctx); err != nil {
		logrus.Warnln("Error starting port forwarding:", err)
	}

	// Start printing the logs after deploy is finished
	if r.runCtx.Opts.TailDev {
		if err := r.logger.Start(ctx); err != nil {
			return errors.Wrap(err, "starting logger")
		}
	}

	return r.listener.WatchForChanges(ctx, out, r.doDev)
}
