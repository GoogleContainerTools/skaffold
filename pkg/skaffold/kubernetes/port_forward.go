/*
Copyright 2018 The Skaffold Authors

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

	// forwardedPods is a map of portForwardEntry.key() (string) -> portForwardEntry
	forwardedPods map[string]*portForwardEntry

	// forwardedPorts is a map of port (int32) -> container name (string)
	forwardedPorts map[int32]string
}

type portForwardEntry struct {
	resourceVersion int
	podName         string
	namespace       string
	containerName   string
	port            int32

	cancel context.CancelFunc
}

// Forwarder is an interface that can modify and manage port-forward processes
type Forwarder interface {
	Forward(context.Context, *portForwardEntry) error
	Terminate(*portForwardEntry)
}

type kubectlForwarder struct{}

// Forward port-forwards a pod using kubectl port-forward
// It returns an error only if the process fails or was terminated by a signal other than SIGTERM
func (*kubectlForwarder) Forward(parentCtx context.Context, pfe *portForwardEntry) error {
	logrus.Debugf("Port forwarding %s", pfe)

	ctx, cancel := context.WithCancel(parentCtx)
	pfe.cancel = cancel

	portNumber := fmt.Sprintf("%d", pfe.port)

	cmd := exec.CommandContext(ctx, "kubectl", "port-forward", pfe.podName, portNumber, portNumber, "--namespace", pfe.namespace)
	buf := &bytes.Buffer{}
	cmd.Stdout = buf
	cmd.Stderr = buf

	if err := cmd.Start(); err != nil {
		if errors.Cause(err) == context.Canceled {
			return nil
		}
		return errors.Wrapf(err, "port forwarding pod: %s/%s, port: %s, err: %s", pfe.namespace, pfe.podName, portNumber, buf.String())
	}

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
func NewPortForwarder(out io.Writer, podSelector PodSelector) *PortForwarder {
	return &PortForwarder{
		Forwarder:      &kubectlForwarder{},
		output:         out,
		podSelector:    podSelector,
		forwardedPods:  make(map[string]*portForwardEntry),
		forwardedPorts: make(map[int32]string),
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
func (p *PortForwarder) Start(ctx context.Context) error {
	watcher, err := PodWatcher()
	if err != nil {
		return errors.Wrap(err, "initializing pod watcher")
	}

	go func() {
		defer watcher.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-watcher.ResultChan():
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

func (p *PortForwarder) portForwardPod(ctx context.Context, pod *v1.Pod) error {
	resourceVersion, err := strconv.Atoi(pod.ResourceVersion)
	if err != nil {
		return errors.Wrap(err, "converting resource version to integer")
	}

	for _, c := range pod.Spec.Containers {
		for _, port := range c.Ports {
			// If the port is already port-forwarded by another container,
			// continue without port-forwarding
			currentApp, ok := p.forwardedPorts[port.ContainerPort]
			if ok && currentApp != c.Name {
				color.LightYellow.Fprintf(p.output, "Port %d for %s is already in use by container %s\n", port.ContainerPort, c.Name, currentApp)
				continue
			}

			entry := &portForwardEntry{
				resourceVersion: resourceVersion,
				podName:         pod.Name,
				namespace:       pod.Namespace,
				containerName:   c.Name,
				port:            port.ContainerPort,
			}

			if prevEntry, ok := p.forwardedPods[entry.key()]; ok {
				// Check if this is a new generation of pod
				if entry.resourceVersion > prevEntry.resourceVersion {
					p.Terminate(prevEntry)
				}
			}

			color.Default.Fprintln(p.output, fmt.Sprintf("Port Forwarding %s %d -> %d", entry.podName, entry.port, entry.port))
			p.forwardedPods[entry.key()] = entry
			p.forwardedPorts[entry.port] = entry.containerName

			if err := p.Forward(ctx, entry); err != nil {
				return errors.Wrap(err, "port forwarding failed")
			}
		}
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
