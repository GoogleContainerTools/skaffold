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

package portforward

import (
	"context"
	"fmt"
	"strconv"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

var (
	// For testing
	newPodWatcher    = kubernetes.NewPodWatcher
	topLevelOwnerKey = kubernetes.TopLevelOwnerKey
)

// WatchingPodForwarder is responsible for selecting pods satisfying a certain condition and port-forwarding the exposed
// container ports within those pods. It also tracks and manages the port-forward connections.
type WatchingPodForwarder struct {
	entryManager *EntryManager
	podWatcher   kubernetes.PodWatcher
	events       chan kubernetes.PodEvent
}

// NewWatchingPodForwarder returns a struct that tracks and port-forwards pods as they are created and modified
func NewWatchingPodForwarder(entryManager *EntryManager, podSelector kubernetes.PodSelector, namespaces []string) *WatchingPodForwarder {
	return &WatchingPodForwarder{
		entryManager: entryManager,
		podWatcher:   newPodWatcher(podSelector, namespaces),
		events:       make(chan kubernetes.PodEvent),
	}
}

func (p *WatchingPodForwarder) Start(ctx context.Context) error {
	p.podWatcher.Register(p.events)
	stopWatcher, err := p.podWatcher.Start()
	if err != nil {
		return err
	}

	go func() {
		defer stopWatcher()

		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-p.events:
				if !ok {
					return
				}

				// At this point, we know the event's type is "ADDED" or "MODIFIED".
				// We must take both types into account as it is possible for the pod to have become ready for port-forwarding before we established the watch.
				pod := evt.Pod
				if evt.Type != watch.Deleted && pod.Status.Phase == v1.PodRunning && pod.DeletionTimestamp == nil {
					if err := p.portForwardPod(ctx, pod); err != nil {
						logrus.Warnf("port forwarding pod failed: %s", err)
					}
				}
			}
		}
	}()

	return nil
}

func (p *WatchingPodForwarder) Stop() {
	p.entryManager.Stop()
}

func (p *WatchingPodForwarder) portForwardPod(ctx context.Context, pod *v1.Pod) error {
	ownerReference := topLevelOwnerKey(pod, pod.Kind)
	for _, c := range pod.Spec.Containers {
		for _, port := range c.Ports {
			// get current entry for this container
			resource := latest.PortForwardResource{
				Type:      constants.Pod,
				Name:      pod.Name,
				Namespace: pod.Namespace,
				Port:      int(port.ContainerPort),
				Address:   constants.DefaultPortForwardAddress,
				LocalPort: int(port.ContainerPort),
			}

			entry, err := p.podForwardingEntry(pod.ResourceVersion, c.Name, port.Name, ownerReference, resource)
			if err != nil {
				return fmt.Errorf("getting pod forwarding entry: %w", err)
			}
			if entry.resource.Port != entry.localPort {
				color.Yellow.Fprintf(p.entryManager.output, "Forwarding container %s/%s to local port %d.\n", pod.Name, c.Name, entry.localPort)
			}
			if prevEntry, ok := p.entryManager.forwardedResources.Load(entry.key()); ok {
				// Check if this is a new generation of pod
				if entry.resourceVersion > prevEntry.resourceVersion {
					p.entryManager.Terminate(prevEntry)
				}
			}
			p.entryManager.forwardPortForwardEntry(ctx, entry)
		}
	}
	return nil
}

func (p *WatchingPodForwarder) podForwardingEntry(resourceVersion, containerName, portName, ownerReference string, resource latest.PortForwardResource) (*portForwardEntry, error) {
	rv, err := strconv.Atoi(resourceVersion)
	if err != nil {
		return nil, fmt.Errorf("converting resource version to integer: %w", err)
	}
	entry := newPortForwardEntry(rv, resource, resource.Name, containerName, portName, ownerReference, 0, true)

	// If we have, return the current entry
	oldEntry, ok := p.entryManager.forwardedResources.Load(entry.key())

	if ok {
		entry.localPort = oldEntry.localPort
		return entry, nil
	}

	// retrieve an open port on the host
	entry.localPort = retrieveAvailablePort(resource.Address, resource.Port, &p.entryManager.forwardedPorts)

	return entry, nil
}
