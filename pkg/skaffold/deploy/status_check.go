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

package deploy

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	kubernetesutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

var (
	// TODO: Move this to a flag or global config.
	// Default deadline set to 10 minutes. This is default value for progressDeadlineInSeconds
	// See: https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/api/apps/v1/types.go#L305
	defaultStatusCheckDeadlineInSeconds int32 = 600
	// Poll period for checking set to 100 milliseconds
	defaultPollPeriodInMilliseconds = 100

	// Default deadline set to 1 minutes for pods.
	defaultPodStatusDeadline = time.Duration(1) * time.Minute

	// For testing
	executeRolloutStatus = getRollOutStatus
)

func StatusCheck(ctx context.Context, defaultLabeller *DefaultLabeller, runCtx *runcontext.RunContext, out io.Writer) error {
	client, err := kubernetesutil.GetClientset()
	if err != nil {
		return err
	}

	wg := &sync.WaitGroup{}

	// Its safe to use sync.Map without locks here as each subroutine adds a different resource/name to the map.
	syncMap := &sync.Map{}

	// Check deployment status
	wg.Add(1)
	go func(syncMap *sync.Map) {
		defer wg.Done()
		StatusCheckDeployments(ctx, client, defaultLabeller, runCtx, syncMap, out)
	}(syncMap)

	// Check pod status
	wg.Add(1)
	go func(syncMap *sync.Map) {
		defer wg.Done()
		StatusCheckPods(ctx, client, defaultLabeller, runCtx, syncMap, out)
	}(syncMap)

	// Wait for all resource status to be fetched
	wg.Wait()
	if isSkaffoldDeployInError(syncMap) {
		return fmt.Errorf("one or more deployed resources were in error")
	}
	return nil
}

func StatusCheckDeployments(ctx context.Context, client kubernetes.Interface, defaultLabeller *DefaultLabeller, runCtx *runcontext.RunContext, syncMap *sync.Map, out io.Writer) {
	dMap, err := getDeployments(client, runCtx.Opts.Namespace, defaultLabeller)
	if err != nil {
		syncMap.Store("could not fetch deployments", err)
		return
	}
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	kubeCtl := &kubectl.CLI{
		Namespace:   runCtx.Opts.Namespace,
		KubeContext: runCtx.KubeContext,
	}
	numDeps := int32(len(dMap))
	var ops int32
	atomic.StoreInt32(&ops, numDeps)

	fmt.Fprintln(out, fmt.Sprintf("Waiting on %d of %d deployments", atomic.LoadInt32(&ops), numDeps))
	for dName, deadline := range dMap {
		deadlineDuration := time.Duration(deadline) * time.Second
		wg.Add(1)
		go func(dName string, deadlineDuration time.Duration) {
			defer func() {
				atomic.AddInt32(&ops, -1)
				printResourceStatus("deployment", dName, syncMap, atomic.LoadInt32(&ops), numDeps, out)
				wg.Done()
			}()
			pollDeploymentRolloutStatus(ctx, kubeCtl, dName, deadlineDuration, syncMap)
		}(dName, deadlineDuration)
	}
}

func getDeployments(client kubernetes.Interface, ns string, l *DefaultLabeller) (map[string]int32, error) {
	deps, err := client.AppsV1().Deployments(ns).List(metav1.ListOptions{
		LabelSelector: l.K8sManagedByLabelKeyValueString(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not fetch deployments")
	}

	depMap := map[string]int32{}

	for _, d := range deps.Items {
		var deadline int32
		if d.Spec.ProgressDeadlineSeconds == nil {
			logrus.Debugf("no progressDeadlineSeconds config found for deployment %s. Setting deadline to %d seconds", d.Name, defaultStatusCheckDeadlineInSeconds)
			deadline = defaultStatusCheckDeadlineInSeconds
		} else {
			deadline = *d.Spec.ProgressDeadlineSeconds
		}
		depMap[d.Name] = deadline
	}

	return depMap, nil
}

func pollDeploymentRolloutStatus(ctx context.Context, k *kubectl.CLI, dName string, deadline time.Duration, syncMap *sync.Map) {
	pollDuration := time.Duration(defaultPollPeriodInMilliseconds) * time.Millisecond
	// Add poll duration to account for one last attempt after progressDeadlineSeconds.
	timeoutContext, cancel := context.WithTimeout(ctx, deadline+pollDuration)
	logrus.Debugf("checking rollout status %s", dName)
	defer cancel()
	for {
		select {
		case <-timeoutContext.Done():
			syncMap.Store(fmt.Sprintf("deployment/%s", dName), errors.Wrap(timeoutContext.Err(), fmt.Sprintf("deployment rollout status could not be fetched within %v", deadline)))
			return
		case <-time.After(pollDuration):
			status, err := executeRolloutStatus(timeoutContext, k, dName)
			if err != nil {
				syncMap.Store(fmt.Sprintf("deployment/%s", dName), err)
				return
			}
			if strings.Contains(status, "successfully rolled out") {
				syncMap.Store(fmt.Sprintf("deployment/%s", dName), nil)
				return
			}
		}
	}
}

func getRollOutStatus(ctx context.Context, k *kubectl.CLI, dName string) (string, error) {
	b, err := k.RunOut(ctx, nil, "rollout", []string{"status", "deployment", dName},
		"--watch=false")
	return string(b), err
}

func StatusCheckPods(ctx context.Context, client kubernetes.Interface, defaultLabeller *DefaultLabeller, runCtx *runcontext.RunContext, syncMap *sync.Map, out io.Writer) {
	podInterface := client.CoreV1().Pods(runCtx.Opts.Namespace)
	pods, err := getPods(podInterface, defaultLabeller)
	if err != nil {
		syncMap.Store("could not fetch pods", err)
		return
	}
	numPods := int32(len(pods))
	var ops int32
	atomic.StoreInt32(&ops, numPods)

	wg := &sync.WaitGroup{}
	defer wg.Wait()
	fmt.Fprintln(out, fmt.Sprintf("Waiting on %d of %d pods", atomic.LoadInt32(&ops), numPods))
	for _, po := range pods {
		wg.Add(1)
		go func(po v1.Pod) {
			defer func() {
				atomic.AddInt32(&ops, -1)
				printResourceStatus("pod", po.Name, syncMap, atomic.LoadInt32(&ops), numPods, out)
				wg.Done()
			}()
			getPodStatus(ctx, podInterface, client, &po, defaultPodStatusDeadline, syncMap)
		}(po)
	}
}

func getPods(pi corev1.PodInterface, l *DefaultLabeller) ([]v1.Pod, error) {
	pods, err := pi.List(metav1.ListOptions{
		LabelSelector: l.K8sManagedByLabelKeyValueString(),
	})
	return pods.Items, err
}

func getPodStatus(ctx context.Context, pi corev1.PodInterface, client kubernetes.Interface, po *v1.Pod, deadline time.Duration, syncMap *sync.Map) {
	err := kubernetesutil.WaitForPodToStabilize(ctx, pi, po.Name, deadline)
	if err == nil {
		syncMap.Store(fmt.Sprintf("pod/%s", po.Name), nil)
		return
	}
	// Get container statuses.
	podInterface := client.CoreV1().Pods(po.Namespace)
	statusErr := kubernetesutil.GetPodDetails(podInterface, po.Name)
	syncMap.Store(fmt.Sprintf("pod/%s", po.Name), statusErr)
}

func isSkaffoldDeployInError(m *sync.Map) bool {
	isError := false
	m.Range(func(k, v interface{}) bool {
		_, isError = v.(error)
		// Stop range iteration if we see an error by returning false.
		return !isError
	})
	return isError
}

func getResourceStatus(m *sync.Map, resource string) error {
	v, ok := m.Load(resource)
	if !ok {
		return fmt.Errorf("error getting status for %s", resource)
	}
	if t, ok := v.(error); ok {
		return t
	}
	return nil
}

func printResourceStatus(resourcetype string, name string, m *sync.Map, numLeft int32, total int32, out io.Writer) {
	resource := fmt.Sprintf("%s/%s", resourcetype, name)
	waitingMsg := fmt.Sprintf("[%d/%d %s(s) still pending]", numLeft, total, resourcetype)
	if numLeft == 0 {
		waitingMsg = fmt.Sprintf("[Status check complete for all %ss]", resourcetype)
	}
	if err := getResourceStatus(m, resource); err != nil {
		color.Default.Fprintln(out, fmt.Sprintf("%s failed %s. Error: %s.", resource, waitingMsg, err.Error()))
	} else {
		color.Default.Fprintln(out, fmt.Sprintf("%s is ready. %s", resource, waitingMsg))
	}
}
