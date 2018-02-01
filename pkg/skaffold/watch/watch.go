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
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/afero"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/config"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/constants"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/docker"
	"github.com/rjeczalik/notify"
	"github.com/sirupsen/logrus"
)

// Watcher provides a watch trigger for the skaffold pipeline to begin
type Watcher interface {
	// Watch watches a set of artifacts for changes, and on the first change returns
	// a reference to the changed artifact
	Watch(artifacts []*config.Artifact, ready chan *WatchEvent, cancel chan struct{}) (*WatchEvent, error)
}

type WatchEvent struct {
	EventType       string
	ChangedArtifact *config.Artifact
}

var fs = afero.NewOsFs()

type FSWatcher struct{}

const (
	WatchReady = "WatchReady"
	WatchStop  = "WatchStop"
)

//TODO(@r2d4): Figure out best UX to support configuring this blacklist
var ignoredPrefixes = []string{"vendor", ".git"}

func (f *FSWatcher) Watch(artifacts []*config.Artifact, ready chan *WatchEvent, cancel chan struct{}) (*WatchEvent, error) {
	depsToArtifact := map[string]*config.Artifact{}
	c := make(chan notify.EventInfo, 1)
	errCh := make(chan error, 1)
	defer notify.Stop(c)
	for _, a := range artifacts {
		if a.DockerfilePath == "" {
			a.DockerfilePath = constants.DefaultDockerfilePath
		}
		fullPath := filepath.Join(a.Workspace, a.DockerfilePath)
		r, err := fs.Open(fullPath)
		if err != nil {
			return nil, errors.Wrap(err, "opening file for watch")
		}
		deps, err := docker.GetDockerfileDependencies(a.Workspace, r)
		if err != nil {
			return nil, errors.Wrap(err, "getting dockerfile dependencies")
		}
		for _, dep := range deps {
			// We need to evaluate the symlink if the dependency is a symlink
			// inotify will return the final path of the symlink
			// so we need to match it to know which dockerfile needs rebuilding
			evalPath, err := filepath.EvalSymlinks(dep)
			if err != nil {
				return nil, errors.Wrap(err, "following possible symlink")
			}
			depsToArtifact[evalPath] = a
			if err := watchFile(a.Workspace, dep, c); err != nil {
				return nil, errors.Wrapf(err, "starting watch on file %s", dep)
			}
		}
	}
	if ready != nil {
		ready <- &WatchEvent{EventType: WatchReady}
	}
	for {
		select {
		case ei := <-c:
			logrus.Infof("%s %s", ei.Event().String(), ei.Path())
			artifact := depsToArtifact[ei.Path()]
			return &WatchEvent{
				EventType:       ei.Event().String(),
				ChangedArtifact: artifact,
			}, nil
		case err := <-errCh:
			return nil, err
		case <-cancel:
			logrus.Info("Watch canceled")
			return &WatchEvent{EventType: WatchStop}, nil
		}
	}

}

func watchFile(workspace, path string, c chan notify.EventInfo) error {
	for _, ig := range ignoredPrefixes {
		relPath, err := filepath.Rel(workspace, path)
		if err != nil {
			return errors.Wrap(err, "calculating path relative to workspace")
		}
		if strings.HasPrefix(relPath, ig) {
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
