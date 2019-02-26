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
	"os"
	"os/exec"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var Client kubernetes.Interface

func RunSkaffold(t *testing.T, command, dir, namespace, filename string, env []string, additionalArgs ...string) {
	if err := RunSkaffoldNoFail(make(chan bool), command, dir, namespace, filename, env, additionalArgs...); err != nil {
		t.Fatalf("skaffold delete: %v", err)
	}
}

func RunSkaffoldNoFail(cancel chan bool, command, dir, namespace, filename string, env []string, additionalArgs ...string) error {
	args := []string{command, "--namespace", namespace}
	if filename != "" {
		args = append(args, "-f", filename)
	}
	args = append(args, additionalArgs...)

	cmd := exec.Command("skaffold", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	cmd.Start()

	result := make(chan error)
	go func() {
		err := cmd.Wait()
		result <- err
	}()

	select {
	case err := <-result:
		return err
	case <-cancel:
		return cmd.Process.Kill()
	}
}

func Run(t *testing.T, dir, command string, args ...string) {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	if output, err := util.RunCmdOut(cmd); err != nil {
		t.Fatalf("running command [%s %v]: %s %v", command, args, output, err)
	}
}

func SetupNamespace(t *testing.T) (*v1.Namespace, func()) {
	ns, err := Client.CoreV1().Namespaces().Create(&v1.Namespace{
		ObjectMeta: meta_v1.ObjectMeta{
			GenerateName: "skaffold",
		},
	})
	if err != nil {
		t.Fatalf("creating namespace: %s", err)
	}

	return ns, func() {
		Client.CoreV1().Namespaces().Delete(ns.Name, &meta_v1.DeleteOptions{})
	}
}
