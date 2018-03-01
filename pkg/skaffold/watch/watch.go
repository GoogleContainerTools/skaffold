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
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/config"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/constants"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/docker"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/util"
	"github.com/rjeczalik/notify"
	"github.com/sirupsen/logrus"
)

// Watcher provides a watch trigger for the skaffold pipeline to begin
type Watcher interface {
	// Watch watches a set of artifacts for changes, and on the first change
	// returns a reference to the changed artifact
	Watch(artifacts []*config.Artifact, ready chan *Event, cancel chan struct{}) (*Event, error)
}

// Event is sent on any inotify event and returns all artifacts which
// reference the changed dependency
type Event struct {
	EventType        string
	ChangedArtifacts []*config.Artifact
}

// FSWatcher uses inotify to watch for changes and implements
// the Watcher interface
type FSWatcher struct{}

const (
	// WatchReady is EventType sent when the watcher is ready to watch all files
	WatchReady = "WatchReady"
	// WatchStop is the EventType sent when the watcher is stopped by a cancel
	WatchStop = "WatchStop"
)

var (
	// WatchStopEvent is sent when the watcher is stopped by a message on the
	// cancel channel
	WatchStopEvent = &Event{EventType: WatchStop}
	//WatchStartEvent is sent when the watcher is ready to watch all files
	WatchStartEvent = &Event{EventType: WatchReady}
)

//TODO(@r2d4): Figure out best UX to support configuring this blacklist
var ignoredPrefixes = []string{"vendor", ".git"}

// Watch watches a set of artifacts for changes with inotify, and on the first change
// returns a reference to the changed artifact
func (f *FSWatcher) Watch(artifacts []*config.Artifact, ready chan *Event, cancel chan struct{}) (*Event, error) {
	depsToArtifact := map[string][]*config.Artifact{}
	c := make(chan notify.EventInfo, 1)
	defer notify.Stop(c)
	for _, a := range artifacts {
		if err := addDepsForArtifact(a, depsToArtifact); err != nil {
			return nil, errors.Wrap(err, "adding deps for artifact")
		}
		if err := addWatchForDeps(depsToArtifact, c); err != nil {
			return nil, errors.Wrap(err, "adding watching for deps")
		}
	}
	if ready != nil {
		logrus.Info("Watch is ready")
		ready <- WatchStartEvent
	}
	select {
	case ei := <-c:
		logrus.Infof("%s %s", ei.Event().String(), ei.Path())
		artifacts := depsToArtifact[ei.Path()]
		return &Event{
			EventType:        ei.Event().String(),
			ChangedArtifacts: artifacts,
		}, nil
	case <-cancel:
		logrus.Info("Watch canceled")
		return WatchStopEvent, nil
	}
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
	for _, ig := range ignoredPrefixes {
		absPath, err := filepath.Abs(filepath.Join(workspace, ig))
		if err != nil {
			return errors.Wrapf(err, "calculating absolute path of ignored dep %s", ig)
		}
		if strings.HasPrefix(path, absPath) {
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
