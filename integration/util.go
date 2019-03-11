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
	"os/exec"
	"testing"
	"time"

	kubernetesutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	apps_v1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func Run(t *testing.T, dir, command string, args ...string) {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	if output, err := cmd.Output(); err != nil {
		t.Fatalf("running command [%s %v]: %s %v", command, args, output, err)
	}
}

// SetupNamespace creates a Kubernetes namespace to run a test.
func SetupNamespace(t *testing.T) (*v1.Namespace, *NSKubernetesClient, func()) {
	client, err := kubernetesutil.GetClientset()
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

	fmt.Println("Namespace:", ns.Name)

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
	for _, podName := range podNames {
		if err := kubernetesutil.WaitForPodReady(context.Background(), k.client.CoreV1().Pods(k.ns), podName); err != nil {
			k.t.Fatalf("Timed out waiting for pod %s ready in namespace %s", podName, k.ns)
		}
	}
}

// WaitForDeploymentsToStabilize waits for a list of deployments to become stable.
func (k *NSKubernetesClient) WaitForDeploymentsToStabilize(depNames ...string) {
	for _, depName := range depNames {
		if err := kubernetesutil.WaitForDeploymentToStabilize(context.Background(), k.client, k.ns, depName, 10*time.Minute); err != nil {
			k.t.Fatalf("Timed out waiting for deployment %s to stabilize in namespace %s", depName, k.ns)
		}
	}
}

// GetDeployment gets a deployment by name.
func (k *NSKubernetesClient) GetDeployment(depName string) *apps_v1.Deployment {
	k.WaitForDeploymentsToStabilize(depName)

	dep, err := k.client.AppsV1().Deployments(k.ns).Get(depName, meta_v1.GetOptions{})
	if err != nil {
		k.t.Fatalf("Could not find deployment: %s in namespace %s", depName, k.ns)
	}
	return dep
}
