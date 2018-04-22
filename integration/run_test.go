// +build integration

/*
Copyright 2018 Google LLC

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
		if stdout, stderr, err := util.RunCommand(cmd, nil); err != nil {
			logrus.Fatalf("Error authenticating to GKE cluster stdout: %s, stderr: %s, err: %s", stdout, stderr, err)
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
		description string
		dir         string
		extraArgs   []string
		deployments []testObject
		pods        []testObject
		env         map[string]string

		remoteOnly bool
	}

	var testCases = []testRunCase{
		{
			description: "getting-started example",
			pods: []testObject{
				{
					name: "getting-started",
				},
			},
			dir: "../examples/getting-started",
		},
		{
			description: "no manifest example",
			deployments: []testObject{
				{
					name: "skaffold",
				},
			},
			dir: "../examples/no-manifest",
		},
		{
			description: "annotated getting-started example",
			pods: []testObject{
				{
					name: "getting-started",
				},
			},
			dir:       "../examples",
			extraArgs: []string{"-f", "annotated-skaffold.yaml"},
		},
		{
			description: "getting-started envTagger",
			pods: []testObject{
				{
					name: "getting-started",
				},
			},
			dir: "../examples/environment-variables",
			env: map[string]string{"FOO": "foo"},
		},
		// // Don't run this test for now. It takes awhile to download all the
		// // dependencies
		// {
		// 	description: "repository root skaffold.yaml",
		// 	pods: []testObject{
		// 		{
		// 			name:      "skaffold",
		// 			namespace: "default",
		// 		},
		// 	},
		// 	dir: "../",
		// },
		{
			description: "gcb builder example",
			pods: []testObject{
				{
					name: "getting-started",
				},
			},
			dir:        "../examples/getting-started",
			extraArgs:  []string{"-p", "gcb"},
			remoteOnly: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			if !*remote && testCase.remoteOnly {
				t.Skip("skipping remote only test")
			}

			ns, deleteNs := setupNamespace(t)
			defer deleteNs()

			args := []string{"run"}
			args = append(args, testCase.extraArgs...)
			cmd := exec.Command("skaffold", args...)
			env := os.Environ()
			for k, v := range testCase.env {
				env = append(env, fmt.Sprintf("%s=%s", k, v))
			}
			cmd.Env = env
			cmd.Dir = testCase.dir
			out, outerr, err := util.RunCommand(cmd, nil)
			if err != nil {
				t.Fatalf("skaffold run: \nstdout: %s\nstderr: %s\nerror: %s", out, outerr, err)
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
	out, outerr, err := util.RunCommand(kubectlCmd, nil)
	if err != nil {
		t.Fatalf("kubectl config set-context --namespace: %s\nstderr: %s\nerror: %s", out, outerr, err)
	}

	return ns, func() { client.CoreV1().Namespaces().Delete(ns.Name, &meta_v1.DeleteOptions{}); return }
}
func TestFix(t *testing.T) {
	_, deleteNs := setupNamespace(t)
	defer deleteNs()

	fixCmd := exec.Command("skaffold", "fix", "-f", "skaffold.yaml")
	fixCmd.Dir = "testdata/old-config"
	out, _, err := util.RunCommand(fixCmd, nil)
	if err != nil {
		t.Fatalf("testing error: %s", err.Error())
	}
	runCmd := exec.Command("skaffold", "run", "-f", "-")
	runCmd.Dir = "testdata/old-config"
	_, _, err = util.RunCommand(runCmd, bytes.NewReader(out))
	if err != nil {
		t.Fatalf("testing error: %s", err.Error())
	}
}
