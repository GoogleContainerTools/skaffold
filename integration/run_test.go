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
	"os"
	"os/exec"
	"testing"
	"time"

	kubernetesutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
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
			dir:         "../examples/getting-started",
		},
		{
			description: "annotated getting-started example",
			args:        []string{"run"},
			filename:    "annotated-skaffold.yaml",
			pods:        []string{"getting-started"},
			dir:         "../examples",
		},
		{
			description: "getting-started envTagger",
			args:        []string{"run"},
			pods:        []string{"getting-started"},
			dir:         "../examples/tagging-with-environment-variables",
			env:         []string{"FOO=foo"},
		},
		{
			description: "gcb builder example",
			args:        []string{"run", "-p", "gcb"},
			pods:        []string{"getting-started"},
			dir:         "../examples/getting-started",
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
			dir: "../examples/kustomize",
		},
		{
			description: "bazel example",
			args:        []string{"run"},
			pods:        []string{"bazel"},
			dir:         "../examples/bazel",
		},
		{
			description: "kaniko example",
			args:        []string{"run"},
			pods:        []string{"getting-started-kaniko"},
			dir:         "../examples/kaniko",
			remoteOnly:  true,
		},
		{
			description: "helm example",
			args:        []string{"run"},
			deployments: []string{"skaffold-helm"},
			dir:         "../examples/helm-deployment",
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
			cleanupTest(t, ns, testCase.dir, testCase.filename)
		})
	}
}

func TestDev(t *testing.T) {
	type testDevCase struct {
		description   string
		dir           string
		args          []string
		setup         func(t *testing.T) func(t *testing.T)
		jobs          []string
		jobValidation func(t *testing.T, ns *v1.Namespace, j *batchv1.Job)
	}

	testCases := []testDevCase{
		{
			description: "delete and redeploy job",
			dir:         "../examples/test-dev-job",
			args:        []string{"dev"},
			setup: func(t *testing.T) func(t *testing.T) {
				// create foo
				cmd := exec.Command("touch", "../examples/test-dev-job/foo")
				if output, err := util.RunCmdOut(cmd); err != nil {
					t.Fatalf("creating foo: %s %v", output, err)
				}
				return func(t *testing.T) {
					// delete foo
					cmd := exec.Command("rm", "../examples/test-dev-job/foo")
					if output, err := util.RunCmdOut(cmd); err != nil {
						t.Fatalf("creating foo: %s %v", output, err)
					}
				}
			},
			jobs: []string{
				"test-dev-job",
			},
			jobValidation: func(t *testing.T, ns *v1.Namespace, j *batchv1.Job) {
				originalUID := j.GetUID()
				// Make a change to foo so that dev is forced to delete the job and redeploy
				cmd := exec.Command("sh", "-c", "echo bar > ../examples/test-dev-job/foo")
				if output, err := util.RunCmdOut(cmd); err != nil {
					t.Fatalf("creating bar: %s %v", output, err)
				}
				// Make sure the UID of the old Job and the UID of the new Job is different
				err := wait.PollImmediate(time.Millisecond*500, 10*time.Minute, func() (bool, error) {
					newJob, err := client.BatchV1().Jobs(ns.Name).Get(j.Name, meta_v1.GetOptions{})
					if err != nil {
						return false, nil
					}
					return originalUID != newJob.GetUID(), nil
				})
				if err != nil {
					t.Fatalf("original UID and new UID are the same, redeploy failed")
				}
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			ns, deleteNs := setupNamespace(t)
			defer deleteNs()

			cleanupTC := testCase.setup(t)
			defer cleanupTC(t)

			args := []string{}
			args = append(args, testCase.args...)
			args = append(args, "--namespace", ns.Name)

			cmd := exec.Command("skaffold", args...)
			cmd.Dir = testCase.dir
			go func() {
				if output, err := util.RunCmdOut(cmd); err != nil {
					t.Fatalf("skaffold: %s %v", output, err)
				}
			}()

			for _, j := range testCase.jobs {
				if err := kubernetesutil.WaitForJobToStabilize(client, ns.Name, j, 10*time.Minute); err != nil {
					t.Fatalf("Timed out waiting for job to stabilize")
				}
				if testCase.jobValidation != nil {
					job, err := client.BatchV1().Jobs(ns.Name).Get(j, meta_v1.GetOptions{})
					if err != nil {
						t.Fatalf("Could not find job: %s %s", ns.Name, j)
					}
					testCase.jobValidation(t, ns, job)
				}
			}

			// Cleanup
			cleanupTest(t, ns, testCase.dir, "")
		})
	}
}

func cleanupTest(t *testing.T, ns *v1.Namespace, dir, filename string) {
	args := []string{"delete", "--namespace", ns.Name}
	if filename != "" {
		args = append(args, "-f", filename)
	}
	cmd := exec.Command("skaffold", args...)
	cmd.Dir = dir
	if output, err := util.RunCmdOut(cmd); err != nil {
		t.Fatalf("skaffold delete: %s %v", output, err)
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
	fixCmd.Dir = "testdata/old-config"
	out, err := util.RunCmdOut(fixCmd)
	if err != nil {
		t.Fatalf("testing error: %v", err)
	}

	runCmd := exec.Command("skaffold", "run", "--namespace", ns.Name, "-f", "-")
	runCmd.Dir = "testdata/old-config"
	runCmd.Stdin = bytes.NewReader(out)
	err = util.RunCmd(runCmd)
	if err != nil {
		t.Fatalf("testing error: %v", err)
	}
}
