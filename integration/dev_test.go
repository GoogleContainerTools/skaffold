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
	"context"
	"testing"
	"time"

	kubernetesutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func TestDev(t *testing.T) {
	ns, deleteNs := SetupNamespace(t)
	defer deleteNs()

	Run(t, "examples/test-dev-job", "touch", "foo")
	defer Run(t, "examples/test-dev-job", "rm", "foo")

	cancel := make(chan bool)
	go RunSkaffoldNoFail(cancel, "dev", "examples/test-dev-job", ns.Name, "", nil)
	defer func() { cancel <- true }()

	jobName := "test-dev-job"
	if err := kubernetesutil.WaitForJobToStabilize(context.Background(), Client, ns.Name, jobName, 10*time.Minute); err != nil {
		t.Fatalf("Timed out waiting for job to stabilize")
	}

	job, err := Client.BatchV1().Jobs(ns.Name).Get(jobName, meta_v1.GetOptions{})
	if err != nil {
		t.Fatalf("Could not find job: %s %s", ns.Name, jobName)
	}

	// Make a change to foo so that dev is forced to delete the job and redeploy
	Run(t, "examples/test-dev-job", "sh", "-c", "echo bar > foo")

	// Make sure the UID of the old Job and the UID of the new Job is different
	err = wait.PollImmediate(time.Millisecond*500, 10*time.Minute, func() (bool, error) {
		newJob, err := Client.BatchV1().Jobs(ns.Name).Get(job.Name, meta_v1.GetOptions{})
		if err != nil {
			return false, nil
		}
		return job.GetUID() != newJob.GetUID(), nil
	})
	if err != nil {
		t.Fatalf("redeploy failed: %v", err)
	}
}
