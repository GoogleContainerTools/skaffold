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

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
)

var (
	// For testing
	notifyDebuggingContainerStarted    = event.DebuggingContainerStarted
	notifyDebuggingContainerTerminated = event.DebuggingContainerTerminated
)

type ContainerManager struct {
	podWatcher kubernetes.PodWatcher
	active     map[string]string // set of containers that have been notified
	events     chan kubernetes.PodEvent
}

func NewContainerManager(podSelector kubernetes.PodSelector, namespaces []string) *ContainerManager {
	// Create the channel here as Stop() may be called before Start() when a build fails, thus
	// avoiding the possibility of closing a nil channel. Channels are cheap.
	return &ContainerManager{
		podWatcher: kubernetes.NewPodWatcher(podSelector, namespaces),
		active:     map[string]string{},
		events:     make(chan kubernetes.PodEvent),
	}
}

func (d *ContainerManager) Start(ctx context.Context) error {
	if d == nil {
		// debug mode probably not enabled
		return nil
	}

	d.podWatcher.Register(d.events)
	stopWatcher, err := d.podWatcher.Start()
	if err != nil {
		return err
	}

	go func() {
		defer stopWatcher()

		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-d.events:
				if !ok {
					return
				}

				d.checkPod(evt.Pod)
			}
		}
	}()

	return nil
}

func (d *ContainerManager) Stop() {
	// if nil then debug mode probably not enabled
	if d != nil {
		close(d.events)
	}
}

func (d *ContainerManager) checkPod(pod *v1.Pod) {
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
