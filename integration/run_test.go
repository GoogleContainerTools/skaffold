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
	"flag"
	"os"
	"os/exec"
	"testing"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/util"
	"github.com/sirupsen/logrus"
)

var gkeZone = flag.String("gke-zone", "us-central1-a", "gke zone")
var gkeClusterName = flag.String("gke-cluster-name", "integration-tests", "name of the integration test cluster")
var gcpProject = flag.String("gcp-project", "k8s-skaffold", "the gcp project where the integration test cluster lives")
var remote = flag.Bool("remote", false, "if true, run tests on a remote GKE cluster")

func TestMain(m *testing.M) {
	flag.Parse()
	if *remote {
		cmd := exec.Command("gcloud", "container", "clusters", "get-credentials", *gkeClusterName, "--zone", *gkeZone, "--project", *gcpProject)
		if stdout, stderr, err := util.RunCommand(cmd, nil); err != nil {
			logrus.Fatalf("Error authenticating to GKE cluster stdout: %s, stderr: %s, err: %s", stdout, stderr, err)
		}
	}

	os.Exit(m.Run())
}

func TestRunNoArgs(t *testing.T) {
	client, err := kubernetes.GetClientset()
	if err != nil {
		t.Fatalf("Test setup error: getting kubernetes client: %s", err)
	}

	if err := client.CoreV1().Pods("default").Delete("getting-started", nil); err != nil {
		t.Log(err)
	}

	defer func() {
		if err := client.CoreV1().Pods("default").Delete("getting-started", nil); err != nil {
			t.Fatalf("Error deleting pod %s", err)
		}
	}()

	cmd := exec.Command("skaffold", "run")
	cmd.Dir = "../examples/getting-started"
	out, outerr, err := util.RunCommand(cmd, nil)
	if err != nil {
		t.Fatalf("skaffold run: \nstdout: %s\nstderr: %s\nerror: %s", out, outerr, err)
	}
	t.Logf("%s %s", out, outerr)

	if err := kubernetes.WaitForPodReady(client.CoreV1().Pods("default"), "getting-started"); err != nil {
		t.Fatalf("waiting for pod ready %s", err)
	}
}
