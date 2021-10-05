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
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/GoogleContainerTools/skaffold/integration/binpack"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	kubernetesclient "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/client"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	k8s "github.com/GoogleContainerTools/skaffold/pkg/webhook/kubernetes"
)

type TestType int

const (
	CanRunWithoutGcp TestType = iota
	NeedsGcp
)

func MarkIntegrationTest(t *testing.T, testType TestType) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	runOnGCP := os.Getenv("GCP_ONLY") == "true"

	if testType == NeedsGcp && !runOnGCP {
		t.Skip("skipping GCP integration test")
	}

	if testType == CanRunWithoutGcp && runOnGCP {
		t.Skip("skipping non-GCP integration test")
	}

	if partition() && testType == CanRunWithoutGcp && !matchesPartition(t.Name()) {
		t.Skip(fmt.Sprintf("skipping non-GCP integration test that doesn't match partition %s", getPartition()))
	}
}

func partition() bool {
	return getPartition() != ""
}

func getPartition() string {
	return os.Getenv("IT_PARTITION")
}

func matchesPartition(testName string) bool {
	var partition int
	m, lastPartition := binpack.Partitions()
	if p, ok := m[testName]; ok {
		partition = p
	} else {
		partition = lastPartition
	}
	return strconv.Itoa(partition) == getPartition()
}

func Run(t *testing.T, dir, command string, args ...string) {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	if output, err := cmd.Output(); err != nil {
		t.Fatalf("running command [%s %v]: %s %v", command, args, output, err)
	}
}

// SetupNamespace creates a Kubernetes namespace to run a test.
func SetupNamespace(t *testing.T) (*v1.Namespace, *NSKubernetesClient) {
	client, err := kubernetesclient.DefaultClient()
	if err != nil {
		t.Fatalf("Test setup error: getting Kubernetes client: %s", err)
	}

	ns, err := client.CoreV1().Namespaces().Create(context.Background(), &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "skaffold",
		},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("creating namespace: %s", err)
	}

	logrus.Infoln("Namespace:", ns.Name)

	nsClient := &NSKubernetesClient{
		t:      t,
		client: client,
		ns:     ns.Name,
	}

	t.Cleanup(func() {
		client.CoreV1().Namespaces().Delete(context.Background(), ns.Name, metav1.DeleteOptions{})
	})

	return ns, nsClient
}

func DefaultNamespace(t *testing.T) (*v1.Namespace, *NSKubernetesClient) {
	client, err := kubernetesclient.DefaultClient()
	if err != nil {
		t.Fatalf("Test setup error: getting Kubernetes client: %s", err)
	}
	ns, err := client.CoreV1().Namespaces().Get(context.Background(), "default", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("getting default namespace: %s", err)
	}
	return ns, &NSKubernetesClient{
		t:      t,
		client: client,
		ns:     ns.Name,
	}
}

// NSKubernetesClient wraps a Kubernetes Client for a given namespace.
type NSKubernetesClient struct {
	t      *testing.T
	client kubernetes.Interface
	ns     string
}

func (k *NSKubernetesClient) Pods() corev1.PodInterface {
	return k.client.CoreV1().Pods(k.ns)
}

func (k *NSKubernetesClient) Secrets() corev1.SecretInterface {
	return k.client.CoreV1().Secrets(k.ns)
}

func (k *NSKubernetesClient) Services() corev1.ServiceInterface {
	return k.client.CoreV1().Services(k.ns)
}

func (k *NSKubernetesClient) Deployments() typedappsv1.DeploymentInterface {
	return k.client.AppsV1().Deployments(k.ns)
}

func (k *NSKubernetesClient) DefaultSecrets() corev1.SecretInterface {
	return k.client.CoreV1().Secrets("default")
}

func (k *NSKubernetesClient) CreateSecretFrom(ns, name string) {
	secret, err := k.client.CoreV1().Secrets(ns).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		k.t.Fatalf("failed reading default/e2esecret: %s", err)
	}

	secret.Namespace = k.ns
	secret.ResourceVersion = ""
	if _, err = k.Secrets().Create(context.Background(), secret, metav1.CreateOptions{}); err != nil {
		k.t.Fatalf("failed creating %s/e2esecret: %s", k.ns, err)
	}
}

// WaitForPodsReady waits for a list of pods to become ready.
func (k *NSKubernetesClient) WaitForPodsReady(podNames ...string) {
	f := func(pod *v1.Pod) bool {
		for _, cond := range pod.Status.Conditions {
			if cond.Type == v1.PodReady && cond.Status == v1.ConditionTrue {
				return true
			}
		}
		return false
	}
	result := k.waitForPods(f, podNames...)
	logrus.Infof("Pods marked as ready: %v", result)
}

// WaitForPodsInPhase waits for a list of pods to reach the given phase.
func (k *NSKubernetesClient) WaitForPodsInPhase(expectedPhase v1.PodPhase, podNames ...string) {
	f := func(pod *v1.Pod) bool {
		return pod.Status.Phase == expectedPhase
	}
	result := k.waitForPods(f, podNames...)
	logrus.Infof("Pods in phase %q: %v", expectedPhase, result)
}

// waitForPods waits for a list of pods to become ready.
func (k *NSKubernetesClient) waitForPods(podReady func(*v1.Pod) bool, podNames ...string) (podsReady map[string]bool) {
	ctx, cancelTimeout := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancelTimeout()

	pods := k.Pods()
	w, err := pods.Watch(ctx, metav1.ListOptions{})
	if err != nil {
		k.t.Fatalf("Unable to watch pods: %v", err)
	}
	defer w.Stop()

	waitForAllPods := len(podNames) == 0
	if waitForAllPods {
		logrus.Infof("Waiting for all pods in namespace %q to be ready", k.ns)
	} else {
		logrus.Infoln("Waiting for pods", podNames, "to be ready")
	}

	podsReady = map[string]bool{}

	for {
	waitLoop:
		select {
		case <-ctx.Done():
			k.printDiskFreeSpace()
			k.debug("pods")
			k.logs("pod", podNames)
			k.t.Fatalf("Timed out waiting for pods %v in namespace %q", podNames, k.ns)

		case event := <-w.ResultChan():
			if event.Object == nil {
				return
			}
			pod := event.Object.(*v1.Pod)
			if pod.Status.Phase == v1.PodFailed {
				logs, err := pods.GetLogs(pod.Name, &v1.PodLogOptions{}).DoRaw(ctx)
				if err != nil {
					k.t.Fatalf("failed to get logs for failed pod %s: %s", pod.Name, err)
				}
				k.t.Fatalf("pod %s failed. Logs:\n %s", pod.Name, logs)
			}

			if _, found := podsReady[pod.Name]; !found && waitForAllPods {
				podNames = append(podNames, pod.Name)
			}
			podsReady[pod.Name] = podReady(pod)

			var waiting []string
			for _, podName := range podNames {
				if !podsReady[podName] {
					waiting = append(waiting, podName)
				}
			}
			if len(waiting) > 0 {
				logrus.Infof("Still waiting for pods %v", waiting)
				break waitLoop
			} else if l := len(w.ResultChan()); l > 0 {
				// carry on when there are pending messages in case a new pod has been created
				logrus.Infof("%d pending pod update messages", l)
				break waitLoop
			}
			return
		}
	}
}

// GetDeployment gets a deployment by name.
func (k *NSKubernetesClient) GetPod(podName string) *v1.Pod {
	k.t.Helper()
	k.WaitForPodsReady(podName)

	pod, err := k.Pods().Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil {
		k.t.Fatalf("Could not find pod: %s in namespace %s", podName, k.ns)
	}
	return pod
}

// GetDeployment gets a deployment by name.
func (k *NSKubernetesClient) GetDeployment(depName string) *appsv1.Deployment {
	k.t.Helper()
	k.WaitForDeploymentsToStabilize(depName)

	dep, err := k.Deployments().Get(context.Background(), depName, metav1.GetOptions{})
	if err != nil {
		k.t.Fatalf("Could not find deployment: %s in namespace %s", depName, k.ns)
	}
	return dep
}

// WaitForDeploymentsToStabilize waits for a list of deployments to become stable.
func (k *NSKubernetesClient) WaitForDeploymentsToStabilize(depNames ...string) {
	k.t.Helper()
	k.waitForDeploymentsToStabilizeWithTimeout(2*time.Minute, depNames...)
}

func (k *NSKubernetesClient) waitForDeploymentsToStabilizeWithTimeout(timeout time.Duration, depNames ...string) {
	k.t.Helper()
	if len(depNames) == 0 {
		return
	}

	logrus.Infoln("Waiting for deployments", depNames, "to stabilize")

	ctx, cancelTimeout := context.WithTimeout(context.Background(), timeout)
	defer cancelTimeout()

	w, err := k.Deployments().Watch(ctx, metav1.ListOptions{})
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
			k.debug("deployments.apps")
			k.debug("pods")
			k.logs("deployment.app", depNames)
			k.t.Fatalf("Timed out waiting for deployments %v to stabilize in namespace %s", depNames, k.ns)

		case event := <-w.ResultChan():
			dp := event.Object.(*appsv1.Deployment)
			desiredReplicas := *(dp.Spec.Replicas)
			logrus.Infof("Deployment %s: Generation %d/%d, Replicas %d/%d, Available %d/%d",
				dp.Name,
				dp.Status.ObservedGeneration, dp.Generation,
				dp.Status.Replicas, desiredReplicas,
				dp.Status.AvailableReplicas, desiredReplicas)

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
	logrus.Warnln(cmd.Args)
	out, _ := cmd.CombinedOutput()
	fmt.Println(string(out)) // Use fmt.Println, not logrus, for prettier output
}

func (k *NSKubernetesClient) printDiskFreeSpace() {
	cmd := exec.Command("df", "-h")
	logrus.Warnln(cmd.Args)
	out, _ := cmd.CombinedOutput()
	fmt.Println(string(out))
}

// logs is used to print the logs of a resource
func (k *NSKubernetesClient) logs(entity string, names []string) {
	for _, n := range names {
		cmd := exec.Command("kubectl", "-n", k.ns, "logs", entity+"/"+n)
		logrus.Warnln(cmd.Args)
		out, _ := cmd.CombinedOutput()
		fmt.Println(string(out)) // Use fmt.Println, not logrus, for prettier output
	}
}

// ExternalIP waits for the external IP aof a given service.
func (k *NSKubernetesClient) ExternalIP(serviceName string) string {
	svc, err := k.Services().Get(context.Background(), serviceName, metav1.GetOptions{})
	if err != nil {
		k.t.Fatalf("error getting registry service: %v", err)
	}

	ip, err := k8s.GetExternalIP(svc)
	if err != nil {
		k.t.Fatalf("error getting external ip: %v", err)
	}

	return ip
}

func isStable(dp *appsv1.Deployment) bool {
	return dp.Generation <= dp.Status.ObservedGeneration && *(dp.Spec.Replicas) == dp.Status.Replicas && *(dp.Spec.Replicas) == dp.Status.AvailableReplicas
}

func WaitForLogs(t *testing.T, out io.Reader, firstMessage string, moreMessages ...string) {
	lines := make(chan string)
	go func() {
		scanner := bufio.NewScanner(out)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
	}()

	current := 0
	message := firstMessage

	timer := time.NewTimer(90 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			t.Fatal("timeout")
		case line := <-lines:
			if strings.Contains(line, message) {
				if current >= len(moreMessages) {
					return
				}

				message = moreMessages[current]
				current++
			}
		}
	}
}

// SetupDockerClient creates a client against the local docker daemon
func SetupDockerClient(t *testing.T) docker.LocalDaemon {
	kubeConfig, err := kubectx.CurrentConfig()
	if err != nil {
		t.Log("unable to get current cluster context: %w", err)
		t.Logf("test might not be running against the right docker daemon")
	}
	kubeContext := kubeConfig.CurrentContext

	client, err := docker.NewAPIClient(context.Background(), fakeDockerConfig{kubeContext: kubeContext})
	if err != nil {
		t.Fail()
	}
	return client
}

type fakeDockerConfig struct {
	kubeContext string
}

func (d fakeDockerConfig) GetKubeContext() string                 { return d.kubeContext }
func (d fakeDockerConfig) MinikubeProfile() string                { return "" }
func (d fakeDockerConfig) GlobalConfig() string                   { return "" }
func (d fakeDockerConfig) Prune() bool                            { return false }
func (d fakeDockerConfig) ContainerDebugging() bool               { return false }
func (d fakeDockerConfig) GetInsecureRegistries() map[string]bool { return nil }
func (d fakeDockerConfig) Mode() config.RunMode                   { return "" }
