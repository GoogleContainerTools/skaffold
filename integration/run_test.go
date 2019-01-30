// +build integration

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

package integration

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd/config"
	kubernetesutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

var (
	gkeZone        = flag.String("gke-zone", "us-central1-a", "gke zone")
	gkeClusterName = flag.String("gke-cluster-name", "integration-tests", "name of the integration test cluster")
	gcpProject     = flag.String("gcp-project", "k8s-skaffold", "the gcp project where the integration test cluster lives")
	remote         = flag.Bool("remote", false, "if true, run tests on a remote GKE cluster")

	client kubernetes.Interface
)

func TestMain(m *testing.M) {
	flag.Parse()
	if *remote {
		cmd := exec.Command("gcloud", "container", "clusters", "get-credentials", *gkeClusterName, "--zone", *gkeZone, "--project", *gcpProject)
		if err := util.RunCmd(cmd); err != nil {
			logrus.Fatalf("Error authenticating to GKE cluster stdout: %v", err)
		}
	}

	var err error
	client, err = kubernetesutil.GetClientset()
	if err != nil {
		logrus.Fatalf("Test setup error: getting kubernetes client: %s", err)
	}

	exitCode := m.Run()

	os.Exit(exitCode)
}

func TestRun(t *testing.T) {
	tests := []struct {
		description string
		dir         string
		filename    string
		args        []string
		deployments []string
		pods        []string
		env         []string
		remoteOnly  bool
	}{
		{
			description: "getting-started",
			dir:         "examples/getting-started",
			pods:        []string{"getting-started"},
		}, {
			description: "nodejs",
			dir:         "examples/nodejs",
			pods:        []string{"node"},
		}, {
			description: "structure-tests",
			dir:         "examples/structure-tests",
			pods:        []string{"getting-started"},
		}, {
			description: "microservices",
			dir:         "examples/microservices",
			deployments: []string{"leeroy-app", "leeroy-web"},
		}, {
			description: "annotated-skaffold",
			dir:         "examples",
			filename:    "annotated-skaffold.yaml",
			pods:        []string{"getting-started"},
		}, {
			description: "envTagger",
			dir:         "examples/tagging-with-environment-variables",
			pods:        []string{"getting-started"},
			env:         []string{"FOO=foo"},
		}, {
			description: "bazel",
			dir:         "examples/bazel",
			pods:        []string{"bazel"},
		}, {
			description: "Google Cloud Build",
			dir:         "examples/structure-tests",
			args:        []string{"-p", "gcb"},
			pods:        []string{"getting-started"},
			remoteOnly:  true,
		}, {
			description: "kaniko",
			dir:         "examples/kaniko",
			pods:        []string{"getting-started-kaniko"},
			remoteOnly:  true,
		}, {
			description: "kaniko local",
			dir:         "examples/kaniko-local",
			pods:        []string{"getting-started-kaniko"},
			remoteOnly:  true,
		}, {
			description: "helm",
			dir:         "examples/helm-deployment",
			deployments: []string{"skaffold-helm"},
			remoteOnly:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			if !*remote && test.remoteOnly {
				t.Skip("skipping remote only test")
			}

			ns, deleteNs := setupNamespace(t)
			defer deleteNs()

			runSkaffold(t, "run", test.dir, ns.Name, test.filename, test.env)

			for _, p := range test.pods {
				if err := kubernetesutil.WaitForPodReady(context.Background(), client.CoreV1().Pods(ns.Name), p); err != nil {
					t.Fatalf("Timed out waiting for pod ready")
				}
			}

			for _, d := range test.deployments {
				if err := kubernetesutil.WaitForDeploymentToStabilize(context.Background(), client, ns.Name, d, 10*time.Minute); err != nil {
					t.Fatalf("Timed out waiting for deployment to stabilize")
				}
			}

			runSkaffold(t, "delete", test.dir, ns.Name, test.filename, test.env)
		})
	}
}

func TestDeploy(t *testing.T) {
	ns, deleteNs := setupNamespace(t)
	defer deleteNs()

	runSkaffold(t, "deploy", "examples/kustomize", ns.Name, "", nil, "--images", "index.docker.io/library/busybox:1")

	depName := "kustomize-test"
	if err := kubernetesutil.WaitForDeploymentToStabilize(context.Background(), client, ns.Name, depName, 10*time.Minute); err != nil {
		t.Fatalf("Timed out waiting for deployment to stabilize")
	}

	dep, err := client.AppsV1().Deployments(ns.Name).Get(depName, meta_v1.GetOptions{})
	if err != nil {
		t.Fatalf("Could not find deployment: %s %s", ns.Name, depName)
	}

	if dep.Spec.Template.Spec.Containers[0].Image != "index.docker.io/library/busybox:1" {
		t.Fatalf("Wrong image name in kustomized deployment: %s", dep.Spec.Template.Spec.Containers[0].Image)
	}

	runSkaffold(t, "delete", "examples/kustomize", ns.Name, "", nil)
}

func TestDev(t *testing.T) {
	ns, deleteNs := setupNamespace(t)
	defer deleteNs()

	run(t, "examples/test-dev-job", "touch", "foo")
	defer run(t, "examples/test-dev-job", "rm", "foo")

	go runSkaffoldNoFail("dev", "examples/test-dev-job", ns.Name, "", nil)

	jobName := "test-dev-job"
	if err := kubernetesutil.WaitForJobToStabilize(context.Background(), client, ns.Name, jobName, 10*time.Minute); err != nil {
		t.Fatalf("Timed out waiting for job to stabilize")
	}

	job, err := client.BatchV1().Jobs(ns.Name).Get(jobName, meta_v1.GetOptions{})
	if err != nil {
		t.Fatalf("Could not find job: %s %s", ns.Name, jobName)
	}

	// Make a change to foo so that dev is forced to delete the job and redeploy
	run(t, "examples/test-dev-job", "sh", "-c", "echo bar > foo")

	// Make sure the UID of the old Job and the UID of the new Job is different
	err = wait.PollImmediate(time.Millisecond*500, 10*time.Minute, func() (bool, error) {
		newJob, err := client.BatchV1().Jobs(ns.Name).Get(job.Name, meta_v1.GetOptions{})
		if err != nil {
			return false, nil
		}
		return job.GetUID() != newJob.GetUID(), nil
	})
	if err != nil {
		t.Fatalf("redeploy failed: %v", err)
	}
}

func TestDevSync(t *testing.T) {
	ns, deleteNs := setupNamespace(t)
	defer deleteNs()

	go runSkaffoldNoFail("dev", "examples/test-file-sync", ns.Name, "", nil)

	if err := kubernetesutil.WaitForPodReady(context.Background(), client.CoreV1().Pods(ns.Name), "test-file-sync"); err != nil {
		t.Fatalf("Timed out waiting for pod ready")
	}

	run(t, "examples/test-file-sync", "mkdir", "-p", "test")
	run(t, "examples/test-file-sync", "touch", "test/foobar")
	defer run(t, "examples/test-file-sync", "rm", "-rf", "test")

	err := wait.PollImmediate(time.Millisecond*500, 1*time.Minute, func() (bool, error) {
		cmd := exec.Command("kubectl", "exec", "test-file-sync", "-n", ns.Name, "--", "ls", "/test")
		_, err := util.RunCmdOut(cmd)
		return err == nil, nil
	})
	if err != nil {
		t.Fatalf("checking if /test dir exists in container: %v", err)
	}
}

func runSkaffold(t *testing.T, command, dir, namespace, filename string, env []string, additionalArgs ...string) {
	if output, err := runSkaffoldNoFail(command, dir, namespace, filename, env, additionalArgs...); err != nil {
		t.Fatalf("skaffold delete: %s %v", output, err)
	}
}

func runSkaffoldNoFail(command, dir, namespace, filename string, env []string, additionalArgs ...string) ([]byte, error) {
	args := []string{command, "--namespace", namespace}
	if filename != "" {
		args = append(args, "-f", filename)
	}
	args = append(args, additionalArgs...)

	cmd := exec.Command("skaffold", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	return util.RunCmdOut(cmd)
}

func run(t *testing.T, dir, command string, args ...string) {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	if output, err := util.RunCmdOut(cmd); err != nil {
		t.Fatalf("running command [%s %v]: %s %v", command, args, output, err)
	}
}

func setupNamespace(t *testing.T) (*v1.Namespace, func()) {
	ns, err := client.CoreV1().Namespaces().Create(&v1.Namespace{
		ObjectMeta: meta_v1.ObjectMeta{
			GenerateName: "skaffold",
		},
	})
	if err != nil {
		t.Fatalf("creating namespace: %s", err)
	}

	return ns, func() {
		client.CoreV1().Namespaces().Delete(ns.Name, &meta_v1.DeleteOptions{})
	}
}

func TestFix(t *testing.T) {
	ns, deleteNs := setupNamespace(t)
	defer deleteNs()

	fixCmd := exec.Command("skaffold", "fix", "-f", "skaffold.yaml")
	fixCmd.Dir = "testdata"
	out, err := util.RunCmdOut(fixCmd)
	if err != nil {
		t.Fatalf("testing error: %v", err)
	}

	runCmd := exec.Command("skaffold", "run", "--namespace", ns.Name, "-f", "-")
	runCmd.Dir = "testdata"
	runCmd.Stdin = bytes.NewReader(out)
	err = util.RunCmd(runCmd)
	if err != nil {
		t.Fatalf("testing error: %v", err)
	}
}

func TestListConfig(t *testing.T) {
	baseConfig := &config.Config{
		Global: &config.ContextConfig{
			DefaultRepo: "global-repository",
		},
		ContextConfigs: []*config.ContextConfig{
			{
				Kubecontext: "test-context",
				DefaultRepo: "context-local-repository",
			},
		},
	}

	c, _ := yaml.Marshal(*baseConfig)
	cfg, teardown := testutil.TempFile(t, "config", c)
	defer teardown()

	type testListCase struct {
		description    string
		kubectx        string
		expectedOutput []string
	}

	var tests = []testListCase{
		{
			description:    "list for test-context",
			kubectx:        "test-context",
			expectedOutput: []string{"default-repo: context-local-repository"},
		},
		{
			description: "list all",
			expectedOutput: []string{
				"global:",
				"default-repo: global-repository",
				"kube-context: test-context",
				"default-repo: context-local-repository",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			args := []string{"config", "list", "-c", cfg}
			if test.kubectx != "" {
				args = append(args, "-k", test.kubectx)
			} else {
				args = append(args, "--all")
			}
			cmd := exec.Command("skaffold", args...)
			rawOut, err := util.RunCmdOut(cmd)
			if err != nil {
				t.Error(err)
			}
			out := string(rawOut)
			for _, output := range test.expectedOutput {
				if !strings.Contains(out, output) {
					t.Errorf("expected output %s not found in output: %s", output, out)
				}
			}
		})
	}
}

func TestInit(t *testing.T) {
	type testCase struct {
		name             string
		dir              string
		args             []string
		skipSkaffoldYaml bool
	}

	tests := []testCase{
		{
			name: "getting-started",
			dir:  "../examples/getting-started",
		},
		{
			name: "microservices",
			dir:  "../examples/microservices",
			args: []string{
				"-a", "leeroy-app/Dockerfile=gcr.io/k8s-skaffold/leeroy-app",
				"-a", "leeroy-web/Dockerfile=gcr.io/k8s-skaffold/leeroy-web",
			},
		},
		{
			name:             "compose",
			dir:              "../examples/compose",
			args:             []string{"--compose-file", "docker-compose.yaml"},
			skipSkaffoldYaml: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if !test.skipSkaffoldYaml {
				oldYamlPath := filepath.Join(test.dir, "skaffold.yaml")
				oldYaml, err := removeOldSkaffoldYaml(oldYamlPath)
				if err != nil {
					t.Fatalf("removing original skaffold.yaml: %s", err)
				}
				defer restoreOldSkaffoldYaml(oldYaml, oldYamlPath)
			}

			generatedYaml := "skaffold.yaml.out"
			defer func() {
				err := os.Remove(filepath.Join(test.dir, generatedYaml))
				if err != nil {
					t.Errorf("error removing generated skaffold yaml: %v", err)
				}
			}()
			initArgs := []string{"init", "--force", "-f", generatedYaml}
			initArgs = append(initArgs, test.args...)
			initCmd := exec.Command("skaffold", initArgs...)
			initCmd.Dir = test.dir

			out, err := util.RunCmdOut(initCmd)
			if err != nil {
				t.Fatalf("running init: %v, output: %s", err, out)
			}

			runCmd := exec.Command("skaffold", "run", "-f", generatedYaml)
			runCmd.Dir = test.dir
			out, err = util.RunCmdOut(runCmd)
			if err != nil {
				t.Fatalf("running skaffold on generated yaml: %v, output: %s", err, out)
			}
		})
	}
}

func TestSetConfig(t *testing.T) {
	baseConfig := &config.Config{
		Global: &config.ContextConfig{
			DefaultRepo: "global-repository",
		},
		ContextConfigs: []*config.ContextConfig{
			{
				Kubecontext: "test-context",
				DefaultRepo: "context-local-repository",
			},
		},
	}

	c, _ := yaml.Marshal(*baseConfig)
	cfg, teardown := testutil.TempFile(t, "config", c)
	defer teardown()

	type testSetCase struct {
		description string
		kubectx     string
		key         string
		shouldErr   bool
	}

	var tests = []testSetCase{
		{
			description: "set default-repo for context",
			kubectx:     "test-context",
			key:         "default-repo",
		},
		{
			description: "set global default-repo",
			key:         "default-repo",
		},
		{
			description: "fail to set unrecognized value",
			key:         "doubt-this-will-ever-be-a-config-value",
			shouldErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			value := util.RandomID()
			args := []string{"config", "set", test.key, value}
			args = append(args, "-c", cfg)
			if test.kubectx != "" {
				args = append(args, "-k", test.kubectx)
			} else {
				args = append(args, "--global")
			}
			cmd := exec.Command("skaffold", args...)
			if err := util.RunCmd(cmd); err != nil {
				if test.shouldErr {
					return
				}
				t.Error(err)
			}

			listArgs := []string{"config", "list", "-c", cfg}
			if test.kubectx != "" {
				listArgs = append(listArgs, "-k", test.kubectx)
			} else {
				listArgs = append(listArgs, "--all")
			}
			listCmd := exec.Command("skaffold", listArgs...)
			out, err := util.RunCmdOut(listCmd)
			if err != nil {
				t.Error(err)
			}
			t.Log(string(out))
			if !strings.Contains(string(out), fmt.Sprintf("%s: %s", test.key, value)) {
				t.Errorf("value %s not set correctly", test.key)
			}
		})
	}
}

func removeOldSkaffoldYaml(path string) ([]byte, error) {
	skaffoldYaml, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err = os.Remove(path); err != nil {
		return nil, err
	}
	return skaffoldYaml, nil
}

func restoreOldSkaffoldYaml(contents []byte, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if _, err := f.Write(contents); err != nil {
		return err
	}
	return nil
}
