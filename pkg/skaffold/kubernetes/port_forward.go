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
	"sync"
	"syscall"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

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
	forwardedPods *sync.Map

	// forwardedPorts is a map of port (int32) -> container name (string)
	forwardedPorts *sync.Map
}

type portForwardEntry struct {
	resourceVersion int
	podName         string
	namespace       string
	containerName   string
	port            int32

	cmd *exec.Cmd
}

// Forwarder is an interface that can modify and manage port-forward processes
type Forwarder interface {
	Forward(*portForwardEntry) error
	Stop(*portForwardEntry) error
}

type kubectlForwarder struct{}

// Forward port-forwards a pod using kubectl port-forward
// It returns an error only if the process fails or was terminated by a signal other than SIGTERM
func (*kubectlForwarder) Forward(pfe *portForwardEntry) error {
	logrus.Debugf("Port forwarding %s", pfe)
	portNumber := fmt.Sprintf("%d", pfe.port)
	cmd := exec.Command("kubectl", "port-forward", pfe.podName, portNumber, portNumber, "--namespace", pfe.namespace)
	pfe.cmd = cmd

	buf := &bytes.Buffer{}
	cmd.Stdout = buf
	cmd.Stderr = buf

	if err := cmd.Run(); err != nil && !util.IsTerminatedError(err) {
		return errors.Wrapf(err, "port forwarding pod: %s/%s, port: %s, err: %s", pfe.namespace, pfe.podName, portNumber, buf.String())
	}
	return nil
}

// Stop terminates an existing kubectl port-forward command using SIGTERM
func (*kubectlForwarder) Stop(p *portForwardEntry) error {
	logrus.Debugf("Terminating port-forward %s", p)
	if p.cmd == nil {
		return fmt.Errorf("no port-forward command found for %s", p)
	}
	if err := p.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return errors.Wrap(err, "terminating port-forward process")
	}
	return nil
}

// NewPortForwarder returns a struct that tracks and port-forwards pods as they are created and modified
func NewPortForwarder(out io.Writer, podSelector PodSelector) *PortForwarder {
	return &PortForwarder{
		Forwarder:      &kubectlForwarder{},
		output:         out,
		podSelector:    podSelector,
		forwardedPods:  &sync.Map{},
		forwardedPorts: &sync.Map{},
	}
}

func (p *PortForwarder) cleanupPorts() {
	p.forwardedPods.Range(func(k, v interface{}) bool {
		entry := v.(*portForwardEntry)
		if err := p.Stop(entry); err != nil {
			logrus.Warnf("cleaning up port forwards: %s", err)
		}
		return false
	})
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
				p.cleanupPorts()
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
				// If the event's type is "DELETED", warn and continue.
				if evt.Type == watch.Deleted {
					logrus.Warnf("got unexpected event of type %s for pod %s/%s", evt.Type, pod.Namespace, pod.Name)
				}
				// At this point, we know the event's type if "ADDED" or "MODIFIED".
				// We must take both types into account as it is possible for the pod to have become ready for port-forwarding before we established the watch.
				if p.podSelector.Select(pod) && pod.Status.Phase == v1.PodRunning && pod.DeletionTimestamp == nil {
					go func() {
						if err := p.portForwardPod(pod); err != nil {
							logrus.Warnf("port forwarding pod failed: %s", err)
						}
					}()
				}
			}
		}
	}()

	return nil
}

func (p *PortForwarder) portForwardPod(pod *v1.Pod) error {
	resourceVersion, err := strconv.Atoi(pod.ResourceVersion)
	if err != nil {
		return errors.Wrap(err, "converting resource version to integer")
	}

	var g errgroup.Group

	for _, c := range pod.Spec.Containers {
		for _, port := range c.Ports {
			// If the port is already port-forwarded by another container,
			// continue without port-forwarding
			currentApp, ok := p.forwardedPorts.Load(port.ContainerPort)
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
			v, ok := p.forwardedPods.Load(entry.key())

			if ok {
				prevEntry := v.(*portForwardEntry)
				// Check if this is a new generation of pod
				if entry.resourceVersion > prevEntry.resourceVersion {
					if err := p.Stop(prevEntry); err != nil {
						return errors.Wrap(err, "terminating port-forward process")
					}
				}
			}

			color.Default.Fprintln(p.output, fmt.Sprintf("Port Forwarding %s %d -> %d", entry.podName, entry.port, entry.port))
			p.forwardedPods.Store(entry.key(), entry)
			p.forwardedPorts.Store(entry.port, entry.containerName)

			g.Go(func() error {
				return p.Forward(entry)
			})
		}
	}

	if err := g.Wait(); err != nil {
		return errors.Wrap(err, "port forwarding")
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
