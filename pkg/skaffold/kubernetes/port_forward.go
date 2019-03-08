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

package kubernetes

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strconv"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/proto"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// PortForwarder is responsible for selecting pods satisfying a certain condition and port-forwarding the exposed
// container ports within those pods. It also tracks and manages the port-forward connections.
type PortForwarder struct {
	Forwarder

	output      io.Writer
	podSelector PodSelector
	namespaces  []string

	// forwardedPods is a map of portForwardEntry.key() (string) -> portForwardEntry
	forwardedPods map[string]*portForwardEntry
}

type portForwardEntry struct {
	resourceVersion int
	podName         string
	namespace       string
	containerName   string
	address         string
	port            int32
	localPort       int32

	cancel context.CancelFunc
}

// Forwarder is an interface that can modify and manage port-forward processes
type Forwarder interface {
	Forward(context.Context, *portForwardEntry, string) error
	Terminate(*portForwardEntry)
}

type kubectlForwarder struct{}

var (
	// For testing
	retrieveAvailablePort = util.GetAvailablePort
	isPortAvailable       = util.IsPortAvailable
)

// Forward port-forwards a pod using kubectl port-forward
// It returns an error only if the process fails or was terminated by a signal other than SIGTERM
func (*kubectlForwarder) Forward(parentCtx context.Context, pfe *portForwardEntry, address string) error {
	logrus.Debugf("Port forwarding %s", pfe)

	ctx, cancel := context.WithCancel(parentCtx)
	pfe.cancel = cancel

	// Lets create the kubectl port-forward command
	args := []string{"port-forward"}
	if address != "" {
		args = append(args, "--address", address)
	}
	args = append(args, pfe.podName, fmt.Sprintf("%d:%d", pfe.localPort, pfe.port), "--namespace", pfe.namespace)

	cmd := exec.CommandContext(ctx, "kubectl", args...)

	buf := &bytes.Buffer{}
	cmd.Stdout = buf
	cmd.Stderr = buf

	if err := cmd.Start(); err != nil {
		if errors.Cause(err) == context.Canceled {
			return nil
		}
		return errors.Wrapf(err, "port forwarding pod: %s/%s, port: %d to local port: %d, address:%s, err: %s", pfe.namespace, pfe.podName, pfe.port, pfe.localPort, address, buf.String())
	}

	event.Handle(&proto.Event{
		EventType: &proto.Event_PortEvent{
			PortEvent: &proto.PortEvent{
				LocalPort:     pfe.localPort,
				RemotePort:    pfe.port,
				PodName:       pfe.podName,
				ContainerName: pfe.containerName,
				Namespace:     pfe.namespace,
			},
		},
	})

	go cmd.Wait()

	return nil
}

// Terminate terminates an existing kubectl port-forward command using SIGTERM
func (*kubectlForwarder) Terminate(p *portForwardEntry) {
	logrus.Debugf("Terminating port-forward %s", p)

	if p.cancel != nil {
		p.cancel()
	}
}

// NewPortForwarder returns a struct that tracks and port-forwards pods as they are created and modified
func NewPortForwarder(out io.Writer, podSelector PodSelector, namespaces []string) *PortForwarder {
	return &PortForwarder{
		Forwarder:     &kubectlForwarder{},
		output:        out,
		podSelector:   podSelector,
		namespaces:    namespaces,
		forwardedPods: make(map[string]*portForwardEntry),
	}
}

// Stop terminates all kubectl port-forward commands.
func (p *PortForwarder) Stop() {
	for _, entry := range p.forwardedPods {
		p.Terminate(entry)
	}
}

// Start begins a pod watcher that port forwards any pods involving containers with exposed ports.
// TODO(r2d4): merge this event loop with pod watcher from log writer
func (p *PortForwarder) Start(ctx context.Context, address string) error {
	aggregate := make(chan watch.Event)
	stopWatchers, err := AggregatePodWatcher(p.namespaces, aggregate)
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
					if err := p.portForwardPod(ctx, pod, address); err != nil {
						logrus.Warnf("port forwarding pod failed: %s", err)
					}
				}
			}
		}
	}()

	return nil
}

func (p *PortForwarder) portForwardPod(ctx context.Context, pod *v1.Pod, address string) error {
	resourceVersion, err := strconv.Atoi(pod.ResourceVersion)
	if err != nil {
		return errors.Wrap(err, "converting resource version to integer")
	}

	for _, c := range pod.Spec.Containers {
		for _, port := range c.Ports {
			// get current entry for this container
			entry := p.getCurrentEntry(pod, c, port, resourceVersion)
			if entry.port != entry.localPort {
				color.Yellow.Fprintf(p.output, "Forwarding container %s to local port %d.\n", c.Name, entry.localPort)
			}
			if err := p.forward(ctx, entry, address); err != nil {
				return errors.Wrap(err, "failed to forward port")
			}
		}
	}
	return nil
}

func (p *PortForwarder) getCurrentEntry(pod *v1.Pod, c v1.Container, port v1.ContainerPort, resourceVersion int) *portForwardEntry {
	// determine if we have seen this before
	entry := &portForwardEntry{
		resourceVersion: resourceVersion,
		podName:         pod.Name,
		namespace:       pod.Namespace,
		containerName:   c.Name,
		port:            port.ContainerPort,
	}
	// If we have, return the current entry
	oldEntry, ok := p.forwardedPods[entry.key()]
	if ok {
		entry.localPort = oldEntry.localPort
		return entry
	}

	// retrieve an open port on the host
	entry.localPort = int32(retrieveAvailablePort(int(port.ContainerPort)))
	return entry
}

func (p *PortForwarder) forward(ctx context.Context, entry *portForwardEntry, address string) error {
	if prevEntry, ok := p.forwardedPods[entry.key()]; ok {
		// Check if this is a new generation of pod
		if entry.resourceVersion > prevEntry.resourceVersion {
			p.Terminate(prevEntry)
		}
	}

	color.Default.Fprintln(p.output, fmt.Sprintf("Port Forwarding %s/%s %s:%d -> %d", entry.podName, entry.containerName, address, entry.port, entry.localPort))
	p.forwardedPods[entry.key()] = entry

	if err := p.Forward(ctx, entry, address); err != nil {
		return errors.Wrap(err, "port forwarding failed")
	}
	return nil
}

// Key is an identifier for the lock on a port during the skaffold dev cycle.
func (p *portForwardEntry) key() string {
	return fmt.Sprintf("%s-%d", p.containerName, p.port)
}

// String is a utility function that returns the port forward entry as a user-readable string
func (p *portForwardEntry) String() string {
	return fmt.Sprintf("%s/%s:%d", p.podName, p.containerName, p.port)
}
