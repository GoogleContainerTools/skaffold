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
	"fmt"
	"os/exec"
	"testing"

	kubernetesutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
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
func SetupNamespace(t *testing.T) (*v1.Namespace, kubernetes.Interface, func()) {
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

	return ns, client, func() {
		client.CoreV1().Namespaces().Delete(ns.Name, &meta_v1.DeleteOptions{})
	}
}
