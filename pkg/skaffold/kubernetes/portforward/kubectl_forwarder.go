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
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
)

type EntryForwarder interface {
	Forward(parentCtx context.Context, pfe *portForwardEntry, retryFunc func())
	Terminate(p *portForwardEntry)
}

type KubectlForwarder struct {
	kubectl *kubectl.CLI
}

// Forward port-forwards a pod using kubectl port-forward
// It assumes that retryFunc does the synchronization itself
func (k *KubectlForwarder) Forward(parentCtx context.Context, pfe *portForwardEntry, retryFunc func()) {
	ctx, cancel := context.WithCancel(parentCtx)
	// when retrying a portforwarding entry, it might already have a context running
	if pfe.cancel != nil {
		pfe.cancel()
	}
	pfe.cancel = cancel
	cmd := k.kubectl.Command(ctx,
		"port-forward",
		"--pod-running-timeout", "1s",
		fmt.Sprintf("%s/%s", pfe.resource.Type, pfe.resource.Name),
		fmt.Sprintf("%d:%d", pfe.localPort, pfe.resource.Port),
		"--namespace", pfe.resource.Namespace,
	)
	pfe.logBuffer = &bytes.Buffer{}
	cmd.Stdout = pfe.logBuffer
	cmd.Stderr = pfe.logBuffer

	//retry on exit at Start()

	for err := cmd.Start(); err != nil; {
		if errors.Cause(err) == context.Canceled {
			return
		}
		logrus.Debugf("error starting port forwarding %v: %s, output: %s", pfe, err, pfe.logBuffer.String())
		logrus.Debugf("retrying...")
	}

	retryChan := make(chan bool)
	//retry on exit at Wait()
	go func() {
		if err := cmd.Wait(); err != nil {
			if errors.Cause(err) == context.Canceled {
				return
			}
			logrus.Debugf("terminated port forwarding %v: %s, output: %s", pfe, err, pfe.logBuffer.String())
			retryChan <- false
		}
	}()

	//retry on kubectl port-forward error logs
	go k.monitorErrorLogs(pfe, func() {
		retryChan <- true
	})

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case logError := <-retryChan:
				if logError {
					go retryFunc()
					// read off the exit error as well
					<-retryChan
				} else {
					//otherwise just simply retry
					go retryFunc()
				}
				return
			}
		}
	}()
}

// Terminate terminates an existing kubectl port-forward command using SIGTERM
func (*KubectlForwarder) Terminate(p *portForwardEntry) {
	logrus.Debugf("Terminating port-forward %v", p)

	if p.cancel != nil {
		p.cancel()
	}
}

// Monitor monitors the logs for a kubectl port forward command
// If it sees an error, it calls back to the EntryManager to
// retry the entire port forward operation.
func (*KubectlForwarder) monitorErrorLogs(p *portForwardEntry, retryFunc func()) {
	for {
		time.Sleep(1 * time.Second)
		s, _ := p.logBuffer.ReadString(byte('\n'))
		if s != "" {
			logrus.Tracef("[port-forward] %s", s)

			if strings.Contains(s, "error forwarding port") ||
				strings.Contains(s, "unable to forward") ||
				strings.Contains(s, "error upgrading connection") {
				// kubectl is having an error. retry the command
				logrus.Infof("error in kubectl port-forward logs: %s", s)
				go retryFunc()
				return
			}
		}
	}
}
