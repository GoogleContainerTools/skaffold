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

package debugging

import (
	"context"
	"io"

	"encoding/json"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
)

var (
	// For testing
	aggregatePodWatcher = kubernetes.AggregatePodWatcher

	notifyDebugContainerStarted    = event.DebugContainerStarted
	notifyDebugContainerTerminated = event.DebugContainerTerminated
)

type DebuggableContainerManager struct {
	output      io.Writer
	cli         *kubectl.CLI
	podSelector kubernetes.PodSelector
	namespaces  []string
	active      map[string]string // set of containers that have been notified
}

func NewDebuggableContainerManager(out io.Writer, cli *kubectl.CLI, podSelector kubernetes.PodSelector, namespaces []string) *DebuggableContainerManager {
	return &DebuggableContainerManager{output: out, cli: cli, podSelector: podSelector, namespaces: namespaces, active: map[string]string{}}
}

func (d *DebuggableContainerManager) Start(ctx context.Context) error {
	if d == nil {
		// debug mode probably not enabled
		return nil
	}
	aggregate := make(chan watch.Event)
	stopWatchers, err := aggregatePodWatcher(d.namespaces, aggregate)
	if err != nil {
		stopWatchers()
		return errors.Wrap(err, "initializing debugging container watcher")
	}

	go func() {
		defer stopWatchers()

		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-aggregate:
				if !ok {
					return
				}
				// If the event's type is "ERROR", warn and continue.
				if evt.Type == watch.Error {
					logrus.Warnf("got unexpected event of type %s", evt.Type)
					continue
				}
				// Grab the pod from the event.
				pod, ok := evt.Object.(*v1.Pod)
				if !ok {
					continue
				}
				// Notify onluy when a container is Running or Terminated (not Waiting)
				// "ADDED" is never interesting since no containers are Running yet
				// "MODIFIED" may now have containers in Running or Terminated
				// "DELETED" if the pod is deleted
				if evt.Type != watch.Modified && evt.Type != watch.Deleted {
					continue
				}
				if d.podSelector.Select(pod) {
					d.checkPod(ctx, pod)
				}
			}
		}
	}()
	return nil
}

func (d *DebuggableContainerManager) checkPod(_ context.Context, pod *v1.Pod) {
	debugConfigString, found := pod.Annotations["debug.cloud.google.com/config"]
	if !found {
		return
	}
	var configurations map[string]debug.ContainerDebugConfiguration
	if err := json.Unmarshal([]byte(debugConfigString), &configurations); err != nil {
		logrus.Warnf("Unable to parse debug-config for pod %s/%s: '%s'", pod.Namespace, pod.Name, debugConfigString)
		return
	}
	for _, c := range pod.Status.ContainerStatuses {
		// only examine debuggable containers
		if config, found := configurations[c.Name]; found {
			key := pod.Namespace + "/" + pod.Name + "/" + c.Name
			// only notify of first appearance or disappearance
			_, seen := d.active[key]
			switch {
			case c.State.Running != nil && !seen:
				d.active[key] = key
				color.Yellow.Fprintf(d.output, "Debuggable container: %s runtime=%s\n", key, config.Runtime)
				notifyDebugContainerStarted(
					pod.Name,
					c.Name,
					pod.Namespace,
					config.ArtifactImage,
					config.Runtime,
					config.WorkingDir,
					config.Ports)

			case c.State.Terminated != nil && seen:
				delete(d.active, key)
				color.Yellow.Fprintf(d.output, "Debuggable container %s terminated\n", key)
				notifyDebugContainerTerminated(pod.Name, c.Name, pod.Namespace,
					config.ArtifactImage,
					config.Runtime,
					config.WorkingDir,
					config.Ports)
			}
		}
	}
}
