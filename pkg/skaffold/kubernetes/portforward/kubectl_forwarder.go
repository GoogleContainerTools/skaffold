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
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
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
	isPortFree          = util.IsPortFree
	findNewestPodForSvc = findNewestPodForService
	deferFunc           = func() {}
	waitPortNotFree     = 5 * time.Second
	waitErrorLogs       = 1 * time.Second
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

		args, err := portForwardArgs(ctx, pfe)
		if err != nil {
			logrus.Warnf("couldn't port-forward to %v: %v", pfe, err)
			return
		}
		var buf bytes.Buffer
		cmd := k.kubectl.CommandWithStrictCancellation(ctx, "port-forward", args...)
		cmd.Stdout = &buf
		cmd.Stderr = &buf

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

func portForwardArgs(ctx context.Context, pfe *portForwardEntry) ([]string, error) {
	args := []string{"--pod-running-timeout", "1s"}

	switch pfe.resource.Type {
	// Services need special handling: https://github.com/GoogleContainerTools/skaffold/issues/4522
	case "service":
		podName, remotePort, err := findNewestPodForSvc(ctx, pfe.resource.Namespace, pfe.resource.Name, pfe.resource.Port)
		if err != nil {
			return nil, err
		}
		args = append(args, fmt.Sprintf("pod/%s", podName), fmt.Sprintf("%d:%d", pfe.localPort, remotePort))

	default:
		args = append(args, fmt.Sprintf("%s/%s", pfe.resource.Type, pfe.resource.Name), fmt.Sprintf("%d:%d", pfe.localPort, pfe.resource.Port))
	}

	args = append(args, "--namespace", pfe.resource.Namespace)

	if pfe.resource.Address != "" && pfe.resource.Address != util.Loopback {
		args = append(args, []string{"--address", pfe.resource.Address}...)
	}
	return args, nil
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

// findNewestPodForService queries the cluster to find a pod that fulfills the given service, giving
// preference to pods that were most recently created.  This is in contrast to the selection algorithm
// used by kubectl (see https://github.com/GoogleContainerTools/skaffold/issues/4522 for details).
func findNewestPodForService(ctx context.Context, ns, serviceName string, servicePort int) (string, int, error) {
	client, err := kubernetes.Client()
	if err != nil {
		return "", -1, fmt.Errorf("getting Kubernetes client: %w", err)
	}
	svc, err := client.CoreV1().Services(ns).Get(serviceName, metav1.GetOptions{})
	if err != nil {
		return "", -1, fmt.Errorf("getting service: %w", err)
	}
	svcPort, err := findServicePort(*svc, servicePort)
	if err != nil {
		return "", -1, err
	}

	set := labels.Set(svc.Spec.Selector)
	listOptions := metav1.ListOptions{LabelSelector: set.AsSelector().String()}
	pods, err := client.CoreV1().Pods(ns).List(listOptions)
	if err != nil {
		return "", -1, fmt.Errorf("listing pods: %w", err)
	}
	sort.Slice(pods.Items, newestPodsFirst(pods.Items))

	if logrus.IsLevelEnabled((logrus.TraceLevel)) {
		var names []string
		for _, p := range pods.Items {
			names = append(names, fmt.Sprintf("(pod:%q phase:%v created:%v)", p.Name, p.Status.Phase, p.CreationTimestamp))
		}
		logrus.Tracef("service %s/%d maps to %d pods: %v", serviceName, servicePort, len(pods.Items), names)
	}

	for _, p := range pods.Items {
		if targetPort := findTargetPort(svcPort, p); targetPort > 0 {
			logrus.Debugf("Forwarding service %s/%d to pod %s/%d", serviceName, servicePort, p.Name, targetPort)
			return p.Name, targetPort, nil
		}
	}

	return "", -1, fmt.Errorf("no pods match service %s/%d", serviceName, servicePort)
}

// newestPodsFirst sorts pods by their running state and then by creation time.
func newestPodsFirst(pods []corev1.Pod) func(int, int) bool {
	// prefer Running then Pending
	phases := map[corev1.PodPhase]int{
		corev1.PodRunning:   0,
		corev1.PodPending:   1,
		corev1.PodUnknown:   2,
		corev1.PodSucceeded: 3,
		corev1.PodFailed:    4,
	}
	return func(i, j int) bool {
		if pods[i].Status.Phase != pods[j].Status.Phase {
			// sort running pods ahead of non-running pods
			return phases[pods[i].Status.Phase] < phases[pods[j].Status.Phase]
		}
		ti := pods[i].CreationTimestamp.Time
		tj := pods[j].CreationTimestamp.Time
		return ti.After(tj)
	}
}

func findServicePort(svc corev1.Service, servicePort int) (corev1.ServicePort, error) {
	for _, s := range svc.Spec.Ports {
		if int(s.Port) == servicePort {
			return s, nil
		}
	}
	return corev1.ServicePort{}, fmt.Errorf("service %q does not expose port %d", svc.Name, servicePort)
}

func findTargetPort(svcPort corev1.ServicePort, pod corev1.Pod) int {
	if svcPort.TargetPort.Type == intstr.Int {
		return svcPort.TargetPort.IntValue()
	}
	for _, c := range pod.Spec.Containers {
		for _, p := range c.Ports {
			if svcPort.TargetPort.StrVal == p.Name {
				return int(p.ContainerPort)
			}
		}
	}
	return -1
}
