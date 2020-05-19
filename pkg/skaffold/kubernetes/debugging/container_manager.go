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
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
)

var (
	// For testing
	aggregatePodWatcher = kubernetes.AggregatePodWatcher

	notifyDebuggingContainerStarted    = event.DebuggingContainerStarted
	notifyDebuggingContainerTerminated = event.DebuggingContainerTerminated
)

type ContainerManager struct {
	cli         *kubectl.CLI
	podSelector kubernetes.PodSelector
	namespaces  []string
	active      map[string]string // set of containers that have been notified
	aggregate   chan watch.Event
}

func NewContainerManager(cli *kubectl.CLI, podSelector kubernetes.PodSelector, namespaces []string) *ContainerManager {
	// Create the channel here as Stop() may be called before Start() when a build fails, thus
	// avoiding the possibility of closing a nil channel. Channels are cheap.
	return &ContainerManager{
		cli:         cli,
		podSelector: podSelector,
		namespaces:  namespaces,
		active:      map[string]string{},
		aggregate:   make(chan watch.Event),
	}
}

func (d *ContainerManager) Start(ctx context.Context) error {
	if d == nil {
		// debug mode probably not enabled
		return nil
	}
	stopWatchers, err := aggregatePodWatcher(d.namespaces, d.aggregate)
	if err != nil {
		stopWatchers()
		return fmt.Errorf("initializing debugging container watcher: %w", err)
	}

	go func() {
		defer stopWatchers()

		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-d.aggregate:
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
				// Unlike other event watchers, we ignore event types as checkPod() uses only container status
				if d.podSelector.Select(pod) {
					d.checkPod(ctx, pod)
				}
			}
		}
	}()
	return nil
}

func (d *ContainerManager) Stop() {
	// if nil then debug mode probably not enabled
	if d != nil {
		close(d.aggregate)
	}
}

func (d *ContainerManager) checkPod(_ context.Context, pod *v1.Pod) {
	debugConfigString, found := pod.Annotations[debug.DebugConfigAnnotation]
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
				notifyDebuggingContainerStarted(
					pod.Name,
					c.Name,
					pod.Namespace,
					config.Artifact,
					config.Runtime,
					config.WorkingDir,
					config.Ports)

			case c.State.Terminated != nil && seen:
				delete(d.active, key)
				notifyDebuggingContainerTerminated(pod.Name, c.Name, pod.Namespace,
					config.Artifact,
					config.Runtime,
					config.WorkingDir,
					config.Ports)
			}
		}
	}
}
