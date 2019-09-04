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

package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	pkgkubernetes "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func ShouldRunGCPOnlyTests() bool {
	return os.Getenv("GCP_ONLY") == "true"
}

func Run(t *testing.T, dir, command string, args ...string) {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	if output, err := cmd.Output(); err != nil {
		t.Fatalf("running command [%s %v]: %s %v", command, args, output, err)
	}
}

// SetupNamespace creates a Kubernetes namespace to run a test.
func SetupNamespace(t *testing.T) (*v1.Namespace, *NSKubernetesClient, func()) {
	client, err := pkgkubernetes.Client()
	if err != nil {
		t.Fatalf("Test setup error: getting kubernetes client: %s", err)
	}

	ns, err := client.CoreV1().Namespaces().Create(&v1.Namespace{
		ObjectMeta: meta_v1.ObjectMeta{
			GenerateName: "skaffold",
		},
	})
	if err != nil {
		t.Fatalf("creating namespace: %s", err)
	}

	logrus.Infoln("Namespace:", ns.Name)

	nsClient := &NSKubernetesClient{
		t:      t,
		client: client,
		ns:     ns.Name,
	}

	return ns, nsClient, func() {
		client.CoreV1().Namespaces().Delete(ns.Name, &meta_v1.DeleteOptions{})
	}
}

// NSKubernetesClient wraps a Kubernetes Client for a given namespace.
type NSKubernetesClient struct {
	t      *testing.T
	client kubernetes.Interface
	ns     string
}

// WaitForPodsReady waits for a list of pods to become ready.
func (k *NSKubernetesClient) WaitForPodsReady(podNames ...string) {
	k.WaitForPodsInPhase(v1.PodRunning, podNames...)
}

// WaitForPodsReady waits for a list of pods to become ready.
func (k *NSKubernetesClient) WaitForPodsInPhase(expectedPhase v1.PodPhase, podNames ...string) {
	if len(podNames) == 0 {
		return
	}

	logrus.Infoln("Waiting for pods", podNames, "to be ready")

	ctx, cancelTimeout := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancelTimeout()

	pods := k.client.CoreV1().Pods(k.ns)
	w, err := pods.Watch(meta_v1.ListOptions{})
	if err != nil {
		k.t.Fatalf("Unable to watch pods: %v", err)
	}
	defer w.Stop()

	phases := map[string]v1.PodPhase{}

	for {
	waitLoop:
		select {
		case <-ctx.Done():
			k.printDiskFreeSpace()
			//k.debug("nodes")
			k.debug("pods")
			k.t.Fatalf("Timed out waiting for pods %v ready in namespace %s", podNames, k.ns)

		case event := <-w.ResultChan():
			pod := event.Object.(*v1.Pod)
			logrus.Infoln("Pod", pod.Name, "is", pod.Status.Phase)
			if pod.Status.Phase == v1.PodFailed {
				logs, err := pods.GetLogs(pod.Name, &v1.PodLogOptions{}).DoRaw()
				if err != nil {
					k.t.Fatalf("failed to get logs for failed pod %s: %s", pod.Name, err)
				}
				k.t.Fatalf("pod %s failed. Logs:\n %s", pod.Name, logs)
			}
			phases[pod.Name] = pod.Status.Phase

			for _, podName := range podNames {
				if phases[podName] != expectedPhase {
					break waitLoop
				}
			}

			logrus.Infoln("Pods", podNames, "ready")
			return
		}
	}
}

// GetDeployment gets a deployment by name.
func (k *NSKubernetesClient) GetDeployment(depName string) *appsv1.Deployment {
	k.WaitForDeploymentsToStabilize(depName)

	dep, err := k.client.AppsV1().Deployments(k.ns).Get(depName, meta_v1.GetOptions{})
	if err != nil {
		k.t.Fatalf("Could not find deployment: %s in namespace %s", depName, k.ns)
	}
	return dep
}

// WaitForDeploymentsToStabilize waits for a list of deployments to become stable.
func (k *NSKubernetesClient) WaitForDeploymentsToStabilize(depNames ...string) {
	if len(depNames) == 0 {
		return
	}

	logrus.Infoln("Waiting for deployments", depNames, "to stabilize")

	ctx, cancelTimeout := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancelTimeout()

	w, err := k.client.AppsV1().Deployments(k.ns).Watch(meta_v1.ListOptions{})
	if err != nil {
		k.t.Fatalf("Unable to watch deployments: %v", err)
	}
	defer w.Stop()

	deployments := map[string]*appsv1.Deployment{}

	for {
	waitLoop:
		select {
		case <-ctx.Done():
			k.printDiskFreeSpace()
			//k.debug("nodes")
			k.debug("deployments.apps")
			k.debug("pods")
			k.t.Fatalf("Timed out waiting for deployments %v to stabilize in namespace %s", depNames, k.ns)

		case event := <-w.ResultChan():
			dp := event.Object.(*appsv1.Deployment)
			logrus.Infof("Deployment %s: Generation %d/%d, Replicas %d/%d", dp.Name, dp.Status.ObservedGeneration, dp.Generation, dp.Status.Replicas, *(dp.Spec.Replicas))

			deployments[dp.Name] = dp

			for _, depName := range depNames {
				if d, present := deployments[depName]; !present || !isStable(d) {
					break waitLoop
				}
			}

			logrus.Infoln("Deployments", depNames, "are stable")
			return
		}
	}
}

// debug is used to print all the details about pods or deployments
func (k *NSKubernetesClient) debug(entities string) {
	cmd := exec.Command("kubectl", "-n", k.ns, "get", entities, "-oyaml")
	out, _ := cmd.CombinedOutput()

	logrus.Warnln(cmd.Args)
	// Use fmt.Println, not logrus, for prettier output
	fmt.Println(string(out))
}

func (k *NSKubernetesClient) printDiskFreeSpace() {
	cmd := exec.Command("df", "-h")
	out, _ := cmd.CombinedOutput()
	fmt.Println(string(out))
}

func isStable(dp *appsv1.Deployment) bool {
	return dp.Generation <= dp.Status.ObservedGeneration && *(dp.Spec.Replicas) == dp.Status.Replicas
}
