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

	v1 "k8s.io/api/core/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug/types"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	eventV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
)

var (
	// For testing
	notifyDebuggingContainerStarted    = event.DebuggingContainerStarted
	notifyDebuggingContainerTerminated = event.DebuggingContainerTerminated
	debuggingContainerStartedV2        = eventV2.DebuggingContainerStarted
	debuggingContainerTerminatedV2     = eventV2.DebuggingContainerTerminated
)

type ContainerManager struct {
	podWatcher  kubernetes.PodWatcher
	active      map[string]string // set of containers that have been notified
	events      chan kubernetes.PodEvent
	stopWatcher func()
	namespaces  *[]string
	kubeContext string
}

func NewContainerManager(podSelector kubernetes.PodSelector, namespaces *[]string, kubeContext string) *ContainerManager {
	// Create the channel here as Stop() may be called before Start() when a build fails, thus
	// avoiding the possibility of closing a nil channel. Channels are cheap.
	return &ContainerManager{
		podWatcher:  kubernetes.NewPodWatcher(podSelector),
		active:      map[string]string{},
		events:      make(chan kubernetes.PodEvent),
		stopWatcher: func() {},
		namespaces:  namespaces,
		kubeContext: kubeContext,
	}
}

func (d *ContainerManager) Start(ctx context.Context) error {
	if d == nil {
		// debug mode probably not enabled
		return nil
	}

	d.podWatcher.Register(d.events)
	stopWatcher, err := d.podWatcher.Start(d.kubeContext, *d.namespaces)
	if err != nil {
		return err
	}
	d.stopWatcher = stopWatcher

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
	if d == nil {
		return
	}
	d.stopWatcher()
}

func (d *ContainerManager) Name() string {
	return "Debug Manager"
}

func (d *ContainerManager) checkPod(pod *v1.Pod) {
	debugConfigString, found := pod.Annotations[types.DebugConfig]
	if !found {
		return
	}
	var configurations map[string]types.ContainerDebugConfiguration
	if err := json.Unmarshal([]byte(debugConfigString), &configurations); err != nil {
		log.Entry(context.TODO()).Warnf("Unable to parse debug-config for pod %s/%s: '%s'", pod.Namespace, pod.Name, debugConfigString)
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
				debuggingContainerStartedV2(pod.Name, c.Name, pod.Namespace, config.Artifact, config.Runtime, config.WorkingDir, config.Ports)

			case c.State.Terminated != nil && seen:
				delete(d.active, key)
				notifyDebuggingContainerTerminated(pod.Name, c.Name, pod.Namespace,
					config.Artifact,
					config.Runtime,
					config.WorkingDir,
					config.Ports)
				debuggingContainerTerminatedV2(pod.Name, c.Name, pod.Namespace, config.Artifact, config.Runtime, config.WorkingDir, config.Ports)
			}
		}
	}
}
