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
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
)

type PortForwarder struct {
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
	containerName   string
	port            int32

	cmd *exec.Cmd
}

func NewPortForwarder(out io.Writer, podSelector PodSelector) *PortForwarder {
	return &PortForwarder{
		output:         out,
		podSelector:    podSelector,
		forwardedPods:  &sync.Map{},
		forwardedPorts: &sync.Map{},
	}
}

func (p *PortForwarder) cleanupPorts() {
	p.forwardedPods.Range(func(k, v interface{}) bool {
		entry := v.(*portForwardEntry)
		if err := entry.stop(); err != nil {
			logrus.Warnf("cleaning up port forwards", err)
		}
		return false
	})
}

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

				// Pods will never be "added" in a state that they are ready for port-forwarding
				// so only watch "modified" events
				if evt.Type != watch.Modified {
					continue
				}

				pod, ok := evt.Object.(*v1.Pod)
				if !ok {
					continue
				}
				if p.podSelector.Select(pod) && pod.Status.Phase == v1.PodRunning && pod.DeletionTimestamp == nil {
					go func() {
						if err := p.portForwardPod(ctx, pod); err != nil {
							logrus.Warnf("port forwarding pod failed: %s", err)
						}
					}()
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
			currentApp, ok := p.forwardedPorts.Load(port.ContainerPort)
			if ok && currentApp != c.Name {
				color.LightYellow.Fprintf(p.output, "Port %d for %s is already in use by container %s\n", port.ContainerPort, c.Name, currentApp)
				continue
			}

			entry := &portForwardEntry{
				resourceVersion: resourceVersion,
				podName:         pod.Name,
				containerName:   c.Name,
				port:            port.ContainerPort,
			}
			v, ok := p.forwardedPods.Load(entry.key())

			if ok {
				prevEntry := v.(*portForwardEntry)

				// Check if this is a new generation of pod
				if entry.resourceVersion > prevEntry.resourceVersion {
					if err := prevEntry.stop(); err != nil {
						return errors.Wrap(err, "terminating port-forward process")
					}
				}
			}

			if err := p.forward(entry); err != nil {
				return errors.Wrap(err, "port forwarding")
			}
		}
	}

	return nil
}

func (p *PortForwarder) forward(pfe *portForwardEntry) error {
	portNumber := fmt.Sprintf("%d", pfe.port)
	color.Default.Fprintln(p.output, fmt.Sprintf("Port Forwarding %s %d -> %d", pfe.podName, pfe.port, pfe.port))
	cmd := exec.Command("kubectl", "port-forward", fmt.Sprintf("pod/%s", pfe.podName), portNumber, portNumber)
	pfe.cmd = cmd

	p.forwardedPods.Store(pfe.key(), pfe)
	p.forwardedPorts.Store(pfe.port, pfe.containerName)

	if err := util.RunCmd(cmd); err != nil && !IsTerminatedError(err) {
		return errors.Wrapf(err, "port forwarding pod: %s, port: %s", pfe.podName, portNumber)
	}
	return nil
}

func IsTerminatedError(err error) bool {
	exitError, ok := err.(*exec.ExitError)
	if !ok {
		return false
	}
	ws := exitError.Sys().(syscall.WaitStatus)
	return ws.Signal() == syscall.SIGTERM
}

func (p *portForwardEntry) key() string {
	return fmt.Sprintf("%s-%d", p.containerName, p.port)
}

func (p *portForwardEntry) String() string {
	return fmt.Sprintf("%s/%s:%d", p.podName, p.containerName, p.port)
}

func (p *portForwardEntry) stop() error {
	logrus.Debugf("Terminating port-forward %s", p.String())
	if p.cmd == nil {
		return fmt.Errorf("No port-forward command found for %s", p.String())
	}
	if err := p.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return errors.Wrap(err, "terminating port-forward process")
	}
	return nil
}
