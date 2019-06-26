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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/portforward"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/watch"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ErrorConfigurationChanged is a special error that's returned when the skaffold configuration was changed.
var ErrorConfigurationChanged = errors.New("configuration changed")

// Dev watches for changes and runs the skaffold build and deploy
// config until interrupted by the user.
func (r *SkaffoldRunner) Dev(ctx context.Context, out io.Writer, artifacts []*latest.Artifact) error {
	logger := r.newLogger(out, artifacts)
	defer logger.Stop()

	forwarderManager := portforward.NewForwarderManager(out, r.imageList, r.runCtx.Namespaces, r.defaultLabeller.K8sManagedByLabelKeyValueString(), r.runCtx.Opts.PortForward)
	defer forwarderManager.Stop()

	// Create watcher and register artifacts to build current state of files.
	changed := changes{}
	onChange := func() error {
		defer changed.reset()

		logger.Mute()

		for _, a := range changed.dirtyArtifacts {
			s, err := sync.NewItem(a.artifact, a.events, r.builds, r.runCtx.InsecureRegistries)
			if err != nil {
				return errors.Wrap(err, "sync")
			}
			if s != nil {
				changed.AddResync(s)
			} else {
				changed.AddRebuild(a.artifact)
			}
		}

		switch {
		case changed.needsReload:
			return ErrorConfigurationChanged
		case len(changed.needsResync) > 0:
			for _, s := range changed.needsResync {
				color.Default.Fprintf(out, "Syncing %d files for %s\n", len(s.Copy)+len(s.Delete), s.Image)

				if err := r.Syncer.Sync(ctx, s); err != nil {
					logrus.Warnln("Skipping deploy due to sync error:", err)
					return nil
				}
			}
		case len(changed.needsRebuild) > 0:
			if _, err := r.BuildAndTest(ctx, out, changed.needsRebuild); err != nil {
				logrus.Warnln("Skipping deploy due to error:", err)
				return nil
			}
			fallthrough
		case changed.needsRedeploy:
			if err := r.Deploy(ctx, out, r.builds); err != nil {
				logrus.Warnln("Skipping deploy due to error:", err)
				return nil
			}
		}

		logger.Unmute()
		return nil
	}

	// Watch artifacts
	for i := range artifacts {
		artifact := artifacts[i]
		if !r.runCtx.Opts.IsTargetImage(artifact) {
			continue
		}

		if err := r.Watcher.Register(
			func() ([]string, error) { return r.Builder.DependenciesForArtifact(ctx, artifact) },
			func(e watch.Events) { changed.AddDirtyArtifact(artifact, e) },
		); err != nil {
			return errors.Wrapf(err, "watching files for artifact %s", artifact.ImageName)
		}
	}

	// Watch test configuration
	if err := r.Watcher.Register(
		r.Tester.TestDependencies,
		func(watch.Events) { changed.needsRedeploy = true },
	); err != nil {
		return errors.Wrap(err, "watching test files")
	}

	// Watch deployment configuration
	if err := r.Watcher.Register(
		r.Deployer.Dependencies,
		func(watch.Events) { changed.needsRedeploy = true },
	); err != nil {
		return errors.Wrap(err, "watching files for deployer")
	}

	// Watch Skaffold configuration
	if err := r.Watcher.Register(
		func() ([]string, error) { return []string{r.runCtx.Opts.ConfigurationFile}, nil },
		func(watch.Events) { changed.needsReload = true },
	); err != nil {
		return errors.Wrapf(err, "watching skaffold configuration %s", r.runCtx.Opts.ConfigurationFile)
	}

	// First build
	if _, err := r.BuildAndTest(ctx, out, artifacts); err != nil {
		return errors.Wrap(err, "exiting dev mode because first build failed")
	}

	// Start logs
	if r.runCtx.Opts.TailDev {
		if err := logger.Start(ctx); err != nil {
			return errors.Wrap(err, "starting logger")
		}
	}

	// First deploy
	if err := r.Deploy(ctx, out, r.builds); err != nil {
		return errors.Wrap(err, "exiting dev mode because first deploy failed")
	}

	// Forward ports
	if err := forwarderManager.Start(ctx); err != nil {
		return errors.Wrap(err, "starting forwarder manager")
	}

	return r.Watcher.Run(ctx, out, onChange)
}
