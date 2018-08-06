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
	"os"
	"os/exec"
	"testing"
	"time"

	kubernetesutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

var gkeZone = flag.String("gke-zone", "us-central1-a", "gke zone")
var gkeClusterName = flag.String("gke-cluster-name", "integration-tests", "name of the integration test cluster")
var gcpProject = flag.String("gcp-project", "k8s-skaffold", "the gcp project where the integration test cluster lives")
var remote = flag.Bool("remote", false, "if true, run tests on a remote GKE cluster")

var client kubernetes.Interface

var context *api.Context

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

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})

	cfg, err := kubeConfig.RawConfig()
	if err != nil {
		logrus.Fatalf("loading kubeconfig: %s", err)
	}

	context = cfg.Contexts[cfg.CurrentContext]

	exitCode := m.Run()

	// Reset default context and namespace
	if err := exec.Command("kubectl", "config", "set-context", context.Cluster, "--namespace", context.Namespace).Run(); err != nil {
		logrus.Warn(err)
	}

	os.Exit(exitCode)
}

func TestRun(t *testing.T) {
	type testObject struct {
		name string
	}

	type testRunCase struct {
		description          string
		dir                  string
		args                 []string
		deployments          []testObject
		pods                 []testObject
		deploymentValidation func(t *testing.T, d *appsv1.Deployment)
		env                  map[string]string

		remoteOnly bool
		cleanup    func(t *testing.T)
	}

	var testCases = []testRunCase{
		{
			description: "getting-started example",
			args:        []string{"run"},
			pods: []testObject{
				{
					name: "getting-started",
				},
			},
			dir: "../examples/getting-started",
		},
		{
			description: "annotated getting-started example",
			args:        []string{"run", "-f", "annotated-skaffold.yaml"},
			pods: []testObject{
				{
					name: "getting-started",
				},
			},
			dir: "../examples",
		},
		{
			description: "getting-started envTagger",
			args:        []string{"run"},
			pods: []testObject{
				{
					name: "getting-started",
				},
			},
			dir: "../examples/tagging-with-environment-variables",
			env: map[string]string{"FOO": "foo"},
		},
		{
			description: "gcb builder example",
			args:        []string{"run", "-p", "gcb"},
			pods: []testObject{
				{
					name: "getting-started",
				},
			},
			dir:        "../examples/getting-started",
			remoteOnly: true,
		},
		{
			description: "deploy kustomize",
			args:        []string{"deploy", "--images", "index.docker.io/library/busybox:1"},
			deployments: []testObject{
				{
					name: "kustomize-test",
				},
			},
			deploymentValidation: func(t *testing.T, d *appsv1.Deployment) {
				if d == nil {
					t.Fatalf("Could not find deployment")
				}
				if d.Spec.Template.Spec.Containers[0].Image != "index.docker.io/library/busybox:1" {
					t.Fatalf("Wrong image name in kustomized deployment: %s", d.Spec.Template.Spec.Containers[0].Image)
				}
			},
			dir: "../examples/kustomize",
		},
		{
			description: "bazel example",
			args:        []string{"run"},
			pods: []testObject{
				{
					name: "bazel",
				},
			},
			dir: "../examples/bazel",
		},
		{
			description: "kaniko example",
			args:        []string{"run"},
			pods: []testObject{
				{
					name: "getting-started-kaniko",
				},
			},
			dir:        "../examples/kaniko",
			remoteOnly: true,
		},
		{
			description: "helm example",
			args:        []string{"run"},
			deployments: []testObject{
				{
					name: "skaffold-helm",
				},
			},
			dir:        "../examples/helm-deployment",
			remoteOnly: true,
			cleanup: func(t *testing.T) {
				cmd := exec.Command("helm", "delete", "--purge", "skaffold-helm")
				output, err := util.RunCmdOut(cmd)
				if err != nil {
					t.Fatalf("skaffold: %s %v", output, err)
				}
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			if !*remote && testCase.remoteOnly {
				t.Skip("skipping remote only test")
			}

			ns, deleteNs := setupNamespace(t)
			defer deleteNs()

			cmd := exec.Command("skaffold", testCase.args...)
			env := os.Environ()
			for k, v := range testCase.env {
				env = append(env, fmt.Sprintf("%s=%s", k, v))
			}
			cmd.Env = env
			cmd.Dir = testCase.dir
			output, err := util.RunCmdOut(cmd)
			if err != nil {
				t.Fatalf("skaffold: %s %v", output, err)
			}

			for _, p := range testCase.pods {
				if err := kubernetesutil.WaitForPodReady(client.CoreV1().Pods(ns.Name), p.name); err != nil {
					t.Fatalf("Timed out waiting for pod ready")
				}
			}

			for _, d := range testCase.deployments {
				if err := kubernetesutil.WaitForDeploymentToStabilize(client, ns.Name, d.name, 10*time.Minute); err != nil {
					t.Fatalf("Timed out waiting for deployment to stabilize")
				}
				if testCase.deploymentValidation != nil {
					deployment, err := client.AppsV1().Deployments(ns.Name).Get(d.name, meta_v1.GetOptions{})
					if err != nil {
						t.Fatalf("Could not find deployment: %s %s", ns.Name, d)
					}
					testCase.deploymentValidation(t, deployment)
				}

				if testCase.cleanup != nil {
					testCase.cleanup(t)
				}
			}
		})
	}
}

func setupNamespace(t *testing.T) (*v1.Namespace, func()) {
	namespaceName := util.RandomID()
	ns, err := client.CoreV1().Namespaces().Create(&v1.Namespace{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      namespaceName,
			Namespace: namespaceName,
		},
	})
	if err != nil {
		t.Fatalf("creating namespace: %s", err)
	}

	kubectlCmd := exec.Command("kubectl", "config", "set-context", context.Cluster, "--namespace", ns.Name)
	if err := util.RunCmd(kubectlCmd); err != nil {
		t.Fatalf("kubectl config set-context --namespace: %v", err)
	}

	os.Setenv("SKAFFOLD_DEPLOY_NAMESPACE", namespaceName)

	return ns, func() {
		client.CoreV1().Namespaces().Delete(ns.Name, &meta_v1.DeleteOptions{})
		os.Setenv("SKAFFOLD_DEPLOY_NAMESPACE", "")
	}
}
func TestFix(t *testing.T) {
	_, deleteNs := setupNamespace(t)
	defer deleteNs()

	fixCmd := exec.Command("skaffold", "fix", "-f", "skaffold.yaml")
	fixCmd.Dir = "testdata/old-config"
	out, err := util.RunCmdOut(fixCmd)
	if err != nil {
		t.Fatalf("testing error: %v", err)
	}

	runCmd := exec.Command("skaffold", "run", "-f", "-")
	runCmd.Dir = "testdata/old-config"
	runCmd.Stdin = bytes.NewReader(out)
	err = util.RunCmd(runCmd)
	if err != nil {
		t.Fatalf("testing error: %v", err)
	}
}
