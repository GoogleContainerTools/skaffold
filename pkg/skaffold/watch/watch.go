/*
Copyright 2018 Google LLC

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

package watch

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/config"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/constants"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/docker"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/rjeczalik/notify"
	"github.com/sirupsen/logrus"
)

const quietPeriod = 500 * time.Millisecond

//TODO(@r2d4): Figure out best UX to support configuring this blacklist
var ignoredPrefixes = []string{"vendor", ".git"}

// WatcherFactory can build Watchers from a list of artifacts to be watched for changes
type WatcherFactory func(artifacts []*config.Artifact) (Watcher, error)

// Watcher provides a watch trigger for the skaffold pipeline to begin
type Watcher interface {
	// Start watches a set of artifacts for changes, and on the first change
	// returns a reference to the changed artifact
	Start(ctx context.Context, onChange func([]*config.Artifact))
}

// fsWatcher uses inotify to watch for changes and implements
// the Watcher interface
type fsWatcher struct {
	fsEvents       chan notify.EventInfo
	depsToArtifact map[string][]*config.Artifact
}

// NewWatcher creates a new Watcher on a list of artifacts.
func NewWatcher(artifacts []*config.Artifact) (Watcher, error) {
	// TODO(@dgageot): If file changes happen too quickly, events might be lost
	fsEvents := make(chan notify.EventInfo, 100)

	depsToArtifact := map[string][]*config.Artifact{}
	for _, a := range artifacts {
		if err := addDepsForArtifact(a, depsToArtifact); err != nil {
			notify.Stop(fsEvents)
			return nil, err
		}
		if err := addWatchForDeps(depsToArtifact, fsEvents); err != nil {
			notify.Stop(fsEvents)
			return nil, err
		}
	}

	logrus.Info("Watch is ready")

	return &fsWatcher{
		fsEvents:       fsEvents,
		depsToArtifact: depsToArtifact,
	}, nil
}

// Start watches a set of artifacts for changes with inotify, and on the first change
// returns a reference to the changed artifact
func (f *fsWatcher) Start(ctx context.Context, onChange func([]*config.Artifact)) {
	var changedPaths []string

	timer := time.NewTimer(1<<63 - 1) // Forever
	defer timer.Stop()

	for {
		select {
		case ei := <-f.fsEvents:
			logrus.Infof("%s %s", ei.Event().String(), ei.Path())
			changedPaths = append(changedPaths, ei.Path())
			timer.Reset(quietPeriod)
		case <-timer.C:
			onChange(depsToArtifacts(changedPaths, f.depsToArtifact))
			changedPaths = nil
		case <-ctx.Done():
			notify.Stop(f.fsEvents)
			return
		}
	}
}

func depsToArtifacts(changedPaths []string, depsToArtifact map[string][]*config.Artifact) []*config.Artifact {
	changedArtifacts := map[*config.Artifact]bool{}
	for _, changedPath := range changedPaths {
		for _, changedArtifact := range depsToArtifact[changedPath] {
			changedArtifacts[changedArtifact] = true
		}
	}

	var artifacts []*config.Artifact
	for changedArtifact := range changedArtifacts {
		artifacts = append(artifacts, changedArtifact)
	}

	return artifacts
}

func addDepsForArtifact(a *config.Artifact, depsToArtifact map[string][]*config.Artifact) error {
	dockerfilePath := a.DockerfilePath
	if a.DockerfilePath == "" {
		dockerfilePath = constants.DefaultDockerfilePath
	}
	fullPath := filepath.Join(a.Workspace, dockerfilePath)
	r, err := util.Fs.Open(fullPath)
	if err != nil {
		return errors.Wrap(err, "opening file for watch")
	}
	deps, err := docker.GetDockerfileDependencies(a.Workspace, r)
	if err != nil {
		return errors.Wrap(err, "getting dockerfile dependencies")
	}
	// Add the dockerfile itself as a dependency too
	deps = append(deps, fullPath)
	for _, dep := range deps {
		fi, err := os.Lstat(dep)
		if err != nil {
			return errors.Wrapf(err, "stat %s", dep)
		}
		if fi.Mode() == os.ModeSymlink {
			logrus.Debugf("%s is a symlink", dep)
			// nothing to do for symlinks
			continue
		}
		dep, err = filepath.Abs(dep)
		if err != nil {
			return errors.Wrapf(err, "getting absolute path of %s", dep)
		}
		artifacts, ok := depsToArtifact[dep]
		if !ok {
			depsToArtifact[dep] = []*config.Artifact{a}
			continue
		}
		depsToArtifact[dep] = append(artifacts, a)
	}
	return nil
}

func addWatchForDeps(depsToArtifact map[string][]*config.Artifact, c chan notify.EventInfo) error {
	// It is a purely aesthetic choice to start the watches in sorted order
	sortedDeps := getKeySlice(depsToArtifact)
	for _, dep := range sortedDeps {
		a := depsToArtifact[dep]
		if err := watchFile(a[0].Workspace, dep, c); err != nil {
			return errors.Wrapf(err, "starting watch on file %s", dep)
		}
	}
	return nil
}

func getKeySlice(m map[string][]*config.Artifact) []string {
	r := []string{}
	for k := range m {
		r = append(r, k)
	}
	sort.Strings(r)
	return r
}

func watchFile(workspace, path string, c chan notify.EventInfo) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return errors.Wrapf(err, "calculating absolute path of file %s", path)
	}

	for _, ig := range ignoredPrefixes {
		ignoredPrefix, err := filepath.Abs(filepath.Join(workspace, ig))
		if err != nil {
			return errors.Wrapf(err, "calculating absolute path of ignored dep %s", ig)
		}

		if strings.HasPrefix(absPath, ignoredPrefix) {
			logrus.Debugf("Ignoring watch on %s", path)
			return nil
		}
	}
	logrus.Infof("Added watch for %s", path)
	if err := notify.Watch(path, c, notify.All); err != nil {
		return errors.Wrapf(err, "adding watch for %s", path)
	}
	return nil
}
