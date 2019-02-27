// +build integration

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
	"flag"
	"os"
	"os/exec"
	"testing"

	kubernetesutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/sirupsen/logrus"
)

var (
	gkeZone        = flag.String("gke-zone", "us-central1-a", "gke zone")
	gkeClusterName = flag.String("gke-cluster-name", "integration-tests", "name of the integration test cluster")
	gcpProject     = flag.String("gcp-project", "k8s-skaffold", "the gcp project where the integration test cluster lives")
	remote         = flag.Bool("remote", false, "if true, run tests on a remote GKE cluster")
)

func TestMain(m *testing.M) {
	flag.Parse()
	if *remote {
		cmd := exec.Command("gcloud", "container", "clusters", "get-credentials", *gkeClusterName, "--zone", *gkeZone, "--project", *gcpProject)
		logrus.Infoln(cmd)

		if out, err := util.RunCmdOut(cmd); err != nil {
			logrus.Fatalf("Error authenticating to GKE cluster: %v, stdout: %v", err, out)
		}
	}

	var err error
	Client, err = kubernetesutil.GetClientset()
	if err != nil {
		logrus.Fatalf("Test setup error: getting kubernetes client: %s", err)
	}

	exitCode := m.Run()

	os.Exit(exitCode)
}
