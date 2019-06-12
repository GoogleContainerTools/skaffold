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
	"strconv"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// AutomaticPodForwarder is responsible for selecting pods satisfying a certain condition and port-forwarding the exposed
// container ports within those pods. It also tracks and manages the port-forward connections.
type AutomaticPodForwarder struct {
	BaseForwarder
	podSelector kubernetes.PodSelector
}

// NewAutomaticPodForwarder returns a struct that tracks and port-forwards pods as they are created and modified
func NewAutomaticPodForwarder(baseForwarder BaseForwarder, podSelector kubernetes.PodSelector) *AutomaticPodForwarder {
	return &AutomaticPodForwarder{
		BaseForwarder: baseForwarder,
		podSelector:   podSelector,
	}
}

func (p *AutomaticPodForwarder) Start(ctx context.Context) error {
	aggregate := make(chan watch.Event)
	stopWatchers, err := kubernetes.AggregatePodWatcher(p.namespaces, aggregate)
	if err != nil {
		stopWatchers()
		return errors.Wrap(err, "initializing pod watcher")
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
				// If the event's type is "DELETED", continue.
				if evt.Type == watch.Deleted {
					continue
				}

				// At this point, we know the event's type is "ADDED" or "MODIFIED".
				// We must take both types into account as it is possible for the pod to have become ready for port-forwarding before we established the watch.
				if p.podSelector.Select(pod) && pod.Status.Phase == v1.PodRunning && pod.DeletionTimestamp == nil {
					if err := p.portForwardPod(ctx, pod); err != nil {
						logrus.Warnf("port forwarding pod failed: %s", err)
					}
				}
			}
		}
	}()

	return nil
}

func (p *AutomaticPodForwarder) portForwardPod(ctx context.Context, pod *v1.Pod) error {
	for _, c := range pod.Spec.Containers {
		for _, port := range c.Ports {
			// get current entry for this container
			resource := latest.PortForwardResource{
				Type:      constants.PodResourceType,
				Name:      pod.Name,
				Namespace: pod.Namespace,
				Port:      port.ContainerPort,
			}

			entry, err := p.getAutomaticPodForwardingEntry(pod.ResourceVersion, c.Name, port.Name, resource)
			if err != nil {
				return errors.Wrap(err, "getting automatic pod forwarding entry")
			}
			if entry.resource.Port != entry.localPort {
				color.Yellow.Fprintf(p.output, "Forwarding container %s to local port %d.\n", c.Name, entry.localPort)
			}
			if prevEntry, ok := p.forwardedResources[entry.key()]; ok {
				// Check if this is a new generation of pod
				if entry.resourceVersion > prevEntry.resourceVersion {
					p.Terminate(prevEntry)
				}
			}
			return p.forwardPortForwardEntry(ctx, entry)
		}
	}
	return nil
}

func (p *AutomaticPodForwarder) getAutomaticPodForwardingEntry(resourceVersion, containerName, portName string, resource latest.PortForwardResource) (*portForwardEntry, error) {
	rv, err := strconv.Atoi(resourceVersion)
	if err != nil {
		return nil, errors.Wrap(err, "converting resource version to integer")
	}
	entry := &portForwardEntry{
		resource:               resource,
		resourceVersion:        rv,
		podName:                resource.Name,
		containerName:          containerName,
		portName:               portName,
		automaticPodForwarding: true,
	}

	// If we have, return the current entry
	oldEntry, ok := p.forwardedResources[entry.key()]
	if ok {
		entry.localPort = oldEntry.localPort
		return entry, nil
	}

	// retrieve an open port on the host
	entry.localPort = int32(retrieveAvailablePort(int(resource.Port), p.forwardedPorts))

	return entry, nil
}
