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
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

type EntryForwarder interface {
	Forward(parentCtx context.Context, pfe *portForwardEntry)
	Terminate(p *portForwardEntry)
}

type KubectlForwarder struct {
	out     io.Writer
	kubectl *kubectl.CLI
}

// NewKubectlForwarder returns a new KubectlForwarder
func NewKubectlForwarder(out io.Writer, cli *kubectl.CLI) *KubectlForwarder {
	return &KubectlForwarder{
		out:     out,
		kubectl: cli,
	}
}

// For testing
var (
	isPortFree      = util.IsPortFree
	portForwardCmd  = portForwardCommand
	deferFunc       = func() {}
	waitPortNotFree = 5 * time.Second
	waitErrorLogs   = 1 * time.Second
)

// Forward port-forwards a pod using kubectl port-forward in the background
// It kills the command on errors in the kubectl port-forward log
// It restarts the command if it was not cancelled by skaffold
// It retries in case the port is taken
func (k *KubectlForwarder) Forward(parentCtx context.Context, pfe *portForwardEntry) {
	go k.forward(parentCtx, pfe)
}

func (k *KubectlForwarder) forward(parentCtx context.Context, pfe *portForwardEntry) {
	var notifiedUser bool
	defer deferFunc()

	for {
		pfe.terminationLock.Lock()
		if pfe.terminated {
			logrus.Debugf("port forwarding %v was cancelled...", pfe)
			pfe.terminationLock.Unlock()
			return
		}
		pfe.terminationLock.Unlock()

		if !isPortFree(util.Loopback, pfe.localPort) {
			//assuming that Skaffold brokered ports don't overlap, this has to be an external process that started
			//since the dev loop kicked off. We are notifying the user in the hope that they can fix it
			color.Red.Fprintf(k.out, "failed to port forward %v, port %d is taken, retrying...\n", pfe, pfe.localPort)
			notifiedUser = true
			time.Sleep(waitPortNotFree)
			continue
		}

		if notifiedUser {
			color.Green.Fprintf(k.out, "port forwarding %v recovered on port %d\n", pfe, pfe.localPort)
			notifiedUser = false
		}

		ctx, cancel := context.WithCancel(parentCtx)
		pfe.cancel = cancel

		var buf bytes.Buffer
		cmd := portForwardCmd(ctx, k.kubectl, pfe, &buf)

		logrus.Debugf("Running command: %s", cmd.Args)
		if err := cmd.Start(); err != nil {
			if ctx.Err() == context.Canceled {
				logrus.Debugf("couldn't start %v due to context cancellation", pfe)
				return
			}
			//retry on exit at Start()
			logrus.Debugf("error starting port forwarding %v: %s, output: %s", pfe, err, buf.String())
			time.Sleep(500 * time.Millisecond)
			continue
		}

		//kill kubectl on port forwarding error logs
		go k.monitorErrorLogs(ctx, &buf, cmd, pfe)
		if err := cmd.Wait(); err != nil {
			if ctx.Err() == context.Canceled {
				logrus.Debugf("terminated %v due to context cancellation", pfe)
				return
			}
			//to make sure that the log monitor gets cleared up
			cancel()
			logrus.Debugf("port forwarding %v got terminated: %s, output: %s", pfe, err, buf.String())
			time.Sleep(500 * time.Millisecond)
		}
	}
}

func portForwardCommand(ctx context.Context, k *kubectl.CLI, pfe *portForwardEntry, buf io.Writer) *kubectl.Cmd {
	args := []string{
		"--pod-running-timeout", "1s",
		fmt.Sprintf("%s/%s", pfe.resource.Type, pfe.resource.Name),
		fmt.Sprintf("%d:%d", pfe.localPort, pfe.resource.Port),
		"--namespace", pfe.resource.Namespace,
	}
	if pfe.resource.Address != "" && pfe.resource.Address != util.Loopback {
		args = append(args, []string{"--address", pfe.resource.Address}...)
	}
	cmd := k.CommandWithStrictCancellation(ctx, "port-forward", args...)
	cmd.Stdout = buf
	cmd.Stderr = buf
	return cmd
}

// Terminate terminates an existing kubectl port-forward command using SIGTERM
func (*KubectlForwarder) Terminate(p *portForwardEntry) {
	logrus.Debugf("Terminating port-forward %v", p)

	p.terminationLock.Lock()
	defer p.terminationLock.Unlock()

	if p.cancel != nil {
		p.cancel()
	}
	p.terminated = true
}

// Monitor monitors the logs for a kubectl port forward command
// If it sees an error, it calls back to the EntryManager to
// retry the entire port forward operation.
func (*KubectlForwarder) monitorErrorLogs(ctx context.Context, logs io.Reader, cmd *kubectl.Cmd, p *portForwardEntry) {
	ticker := time.NewTicker(waitErrorLogs)
	defer ticker.Stop()

	r := bufio.NewReader(logs)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s, _ := r.ReadString('\n')
			if s == "" {
				continue
			}

			logrus.Tracef("[port-forward] %s", s)

			if strings.Contains(s, "error forwarding port") ||
				strings.Contains(s, "unable to forward") ||
				strings.Contains(s, "error upgrading connection") {
				// kubectl is having an error. retry the command
				logrus.Tracef("killing port forwarding %v", p)
				if err := cmd.Terminate(); err != nil {
					logrus.Tracef("failed to kill port forwarding %v, err: %s", p, err)
				}
				return
			}
		}
	}
}
