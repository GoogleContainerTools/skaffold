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
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	type testRunCase struct {
		description          string
		dir                  string
		filename             string
		args                 []string
		deployments          []string
		pods                 []string
		deploymentValidation func(t *testing.T, d *appsv1.Deployment)
		env                  []string

		remoteOnly bool
	}

	var testCases = []testRunCase{
		{
			description: "getting-started example",
			args:        []string{"run"},
			pods:        []string{"getting-started"},
			dir:         "examples/getting-started",
		},
		{
			description: "annotated getting-started example",
			args:        []string{"run"},
			filename:    "annotated-skaffold.yaml",
			pods:        []string{"getting-started"},
			dir:         "examples",
		},
		{
			description: "getting-started envTagger",
			args:        []string{"run"},
			pods:        []string{"getting-started"},
			dir:         "examples/tagging-with-environment-variables",
			env:         []string{"FOO=foo"},
		},
		{
			description: "gcb builder example",
			args:        []string{"run", "-p", "gcb"},
			pods:        []string{"getting-started"},
			dir:         "examples/getting-started",
			remoteOnly:  true,
		},
		{
			description: "deploy kustomize",
			args:        []string{"deploy", "--images", "index.docker.io/library/busybox:1"},
			deployments: []string{"kustomize-test"},
			deploymentValidation: func(t *testing.T, d *appsv1.Deployment) {
				if d == nil {
					t.Fatalf("Could not find deployment")
				}
				if d.Spec.Template.Spec.Containers[0].Image != "index.docker.io/library/busybox:1" {
					t.Fatalf("Wrong image name in kustomized deployment: %s", d.Spec.Template.Spec.Containers[0].Image)
				}
			},
			dir: "examples/kustomize",
		},
		{
			description: "bazel example",
			args:        []string{"run"},
			pods:        []string{"bazel"},
			dir:         "examples/bazel",
		},
		{
			description: "kaniko example",
			args:        []string{"run"},
			pods:        []string{"getting-started-kaniko"},
			dir:         "examples/kaniko",
			remoteOnly:  true,
		},
		{
			description: "helm example",
			args:        []string{"run"},
			deployments: []string{"skaffold-helm"},
			dir:         "examples/helm-deployment",
			remoteOnly:  true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			if !*remote && testCase.remoteOnly {
				t.Skip("skipping remote only test")
			}

			ns, deleteNs := setupNamespace(t)
			defer deleteNs()

			args := []string{}
			args = append(args, testCase.args...)
			args = append(args, "--namespace", ns.Name)
			if testCase.filename != "" {
				args = append(args, "-f", testCase.filename)
			}

			cmd := exec.Command("skaffold", args...)
			cmd.Env = append(os.Environ(), testCase.env...)
			cmd.Dir = testCase.dir
			if output, err := util.RunCmdOut(cmd); err != nil {
				t.Fatalf("skaffold: %s %v", output, err)
			}

			for _, p := range testCase.pods {
				if err := kubernetesutil.WaitForPodReady(client.CoreV1().Pods(ns.Name), p); err != nil {
					t.Fatalf("Timed out waiting for pod ready")
				}
			}

			for _, d := range testCase.deployments {
				if err := kubernetesutil.WaitForDeploymentToStabilize(client, ns.Name, d, 10*time.Minute); err != nil {
					t.Fatalf("Timed out waiting for deployment to stabilize")
				}
				if testCase.deploymentValidation != nil {
					deployment, err := client.AppsV1().Deployments(ns.Name).Get(d, meta_v1.GetOptions{})
					if err != nil {
						t.Fatalf("Could not find deployment: %s %s", ns.Name, d)
					}
					testCase.deploymentValidation(t, deployment)
				}
			}

			// Cleanup
			args = []string{"delete", "--namespace", ns.Name}
			if testCase.filename != "" {
				args = append(args, "-f", testCase.filename)
			}
			cmd = exec.Command("skaffold", args...)
			cmd.Dir = testCase.dir
			if output, err := util.RunCmdOut(cmd); err != nil {
				t.Fatalf("skaffold delete: %s %v", output, err)
			}
		})
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
		name string
		dir  string
		args []string
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			oldYamlPath := filepath.Join(test.dir, "skaffold.yaml")
			oldYaml, err := removeOldSkaffoldYaml(oldYamlPath)
			if err != nil {
				t.Fatalf("removing original skaffold.yaml: %s", err)
			}
			defer restoreOldSkaffoldYaml(oldYaml, oldYamlPath)

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
