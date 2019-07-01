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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/portforward"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ErrorConfigurationChanged is a special error that's returned when the skaffold configuration was changed.
var ErrorConfigurationChanged = errors.New("configuration changed")

func (r *SkaffoldRunner) separateBuildAndSync() error {
	// TODO(nkubala): can this be moved into callback registered in monitor?
	for _, a := range r.changeSet.dirtyArtifacts {
		s, err := sync.NewItem(a.artifact, a.events, r.builds, r.runCtx.InsecureRegistries)
		if err != nil {
			return errors.Wrap(err, "sync")
		}
		if s != nil {
			r.changeSet.AddResync(s)
		} else {
			r.changeSet.AddRebuild(a.artifact)
		}
	}
	return nil
}

func (r *SkaffoldRunner) doDev(ctx context.Context, out io.Writer) error {
	defer r.changeSet.reset()

	r.logger.Mute()

	if err := r.separateBuildAndSync(); err != nil {
		return err
	}

	if r.changeSet.needsAction() {
		// if any action is going to be performed, reset the monitor's changed component tracker for debouncing
		defer r.monitor.Reset()
		defer r.listener.LogWatchToUser(out)
	}

	switch {
	case r.changeSet.needsReload:
		return ErrorConfigurationChanged
	case len(r.changeSet.needsResync) > 0:
		for _, s := range r.changeSet.needsResync {
			color.Default.Fprintf(out, "Syncing %d files for %s\n", len(s.Copy)+len(s.Delete), s.Image)

			if err := r.Syncer.Sync(ctx, s); err != nil {
				logrus.Warnln("Skipping deploy due to sync error:", err)
				return nil
			}
		}
	case len(r.changeSet.needsRebuild) > 0:
		if _, err := r.BuildAndTest(ctx, out, r.changeSet.needsRebuild); err != nil {
			logrus.Warnln("Skipping deploy due to error:", err)
			return nil
		}
		fallthrough
	case r.changeSet.needsRedeploy:
		if err := r.Deploy(ctx, out, r.builds); err != nil {
			logrus.Warnln("Skipping deploy due to error:", err)
			return nil
		}
	}

	r.logger.Unmute()
	return nil
}

// Dev watches for changes and runs the skaffold build and deploy
// config until interrupted by the user.
func (r *SkaffoldRunner) Dev(ctx context.Context, out io.Writer, artifacts []*latest.Artifact) error {
	r.createLogger(out, artifacts)
	defer r.logger.Stop()

	forwarderManager := portforward.NewForwarderManager(out, r.imageList, r.runCtx.Namespaces, r.defaultLabeller.K8sManagedByLabelKeyValueString(), r.runCtx.Opts.PortForward, r.portForwardResources)
	defer forwarderManager.Stop()

	// Watch artifacts
	for i := range artifacts {
		artifact := artifacts[i]
		if !r.runCtx.Opts.IsTargetImage(artifact) {
			continue
		}

		if err := r.monitor.Register(
			func() ([]string, error) { return r.Builder.DependenciesForArtifact(ctx, artifact) },
			func(e filemon.Events) { r.changeSet.AddDirtyArtifact(artifact, e) },
		); err != nil {
			return errors.Wrapf(err, "watching files for artifact %s", artifact.ImageName)
		}
	}

	// Watch test configuration
	if err := r.monitor.Register(
		r.Tester.TestDependencies,
		func(filemon.Events) { r.changeSet.needsRedeploy = true },
	); err != nil {
		return errors.Wrap(err, "watching test files")
	}

	// Watch deployment configuration
	if err := r.monitor.Register(
		r.Deployer.Dependencies,
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

	// Start printing the logs after deploy is finished
	if r.runCtx.Opts.TailDev {
		if err := r.logger.Start(ctx); err != nil {
			return errors.Wrap(err, "starting logger")
		}
	}

	// Forward ports
	if err := forwarderManager.Start(ctx); err != nil {
		return errors.Wrap(err, "starting forwarder manager")
	}

	return r.listener.WatchForChanges(ctx, out, r.doDev)
}
