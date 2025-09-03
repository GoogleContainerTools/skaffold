/*
Copyright 2022 The Skaffold Authors

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
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"google.golang.org/api/option"
	"google.golang.org/api/run/v1"

	"github.com/GoogleContainerTools/skaffold/v2/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/gcp"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestDeployCloudRun(t *testing.T) {
	MarkIntegrationTest(t, NeedsGcp)
	// Other integration tests run with the --default-repo option.
	// This one explicitly specifies the full image name.
	skaffold.Deploy().InDir("testdata/deploy-cloudrun").RunOrFail(t)
	ctx := context.Background()
	svc, err := getRunService(ctx, "k8s-skaffold", "us-central1", "skaffold-test")
	if err != nil {
		t.Fatal(err)
	}
	if err = checkReady(svc); err != nil {
		t.Fatal(err)
	}
}

func TestDeployCloudRunWithHooks(t *testing.T) {
	MarkIntegrationTest(t, NeedsGcp)

	testutil.Run(t, "cloud run deploy with hooks", func(t *testutil.T) {
		expectedOutput := []string{
			"PRE-DEPLOY Cloud Run MODULE1 host hook",
			"POST-DEPLOY Cloud Run MODULE1 host hook",
			"PRE-DEPLOY Cloud Run MODULE2 host hook",
			"POST-DEPLOY Cloud Run MODULE2 host hook",
		}

		out := skaffold.Run().InDir("testdata/deploy-cloudrun-with-hooks").RunOrFailOutput(t.T)

		commandOutput := string(out)
		previousFoundIndex := -1

		for _, expectedOutput := range expectedOutput {
			expectedOutputFoundIndex := strings.Index(commandOutput, expectedOutput)
			isPreviousOutputBeforeThanCurrent := previousFoundIndex < expectedOutputFoundIndex
			t.CheckTrue(isPreviousOutputBeforeThanCurrent)
			previousFoundIndex = expectedOutputFoundIndex
		}
	})
}

func TestDeployCloudRunWorkerPool(t *testing.T) {
	MarkIntegrationTest(t, NeedsGcp)
	// Other integration tests run with the --default-repo option.
	// This one explicitly specifies the full image name.
	skaffold.Deploy().InDir("testdata/deploy-cloudrun-workerpool").RunOrFail(t)
	ctx := context.Background()
	workerpool, err := getWorkerPool(ctx, "k8s-skaffold", "us-central1", "skaffold-test")
	if err != nil {
		t.Fatal(err)
	}
	if err = checkWorkerPoolReadyStatus(workerpool); err != nil {
		t.Fatal(err)
	}
}

func TestDeployJobWithMaxRetries(t *testing.T) {
	MarkIntegrationTest(t, NeedsGcp)

	tests := []struct {
		descrition         string
		jobManifest        string
		skaffoldCfg        string
		args               []string
		expectedMaxRetries int64
	}{
		{
			descrition:         "maxRetries set to specific value",
			expectedMaxRetries: 2,
			jobManifest: `
apiVersion: run.googleapis.com/v1
kind: Job
metadata:
  annotations:
    run.googleapis.com/launch-stage: BETA
  name: %v
spec:
  template:
    spec:
      template:
        spec:
          containers:
            - image: docker.io/library/busybox:latest
              name: job
          maxRetries: 2`,
			skaffoldCfg: `
apiVersion: %v
kind: Config
metadata:
   name: cloud-run-test
manifests:
  rawYaml:
    - job.yaml
deploy:
  cloudrun:
    projectid: %v
    region: %v`,
		},
		{
			descrition:         "maxRetries set to 0",
			expectedMaxRetries: 0,
			jobManifest: `
apiVersion: run.googleapis.com/v1
kind: Job
metadata:
  annotations:
    run.googleapis.com/launch-stage: BETA
  name: %v
spec:
  template:
    spec:
      template:
        spec:
          containers:
            - image: docker.io/library/busybox:latest
              name: job
          maxRetries: 0`,
			skaffoldCfg: `
apiVersion: %v
kind: Config
metadata:
   name: cloud-run-test
manifests:
  rawYaml:
    - job.yaml
deploy:
  cloudrun:
    projectid: %v
    region: %v`,
		},
		{
			descrition:         "maxRetries not specified - default 3",
			expectedMaxRetries: 3,
			jobManifest: `
apiVersion: run.googleapis.com/v1
kind: Job
metadata:
  annotations:
    run.googleapis.com/launch-stage: BETA
  name: %v
spec:
  template:
    spec:
      template:
        spec:
          containers:
            - image: docker.io/library/busybox:latest
              name: job`,
			skaffoldCfg: `
apiVersion: %v
kind: Config
metadata:
   name: cloud-run-test
manifests:
  rawYaml:
    - job.yaml
deploy:
  cloudrun:
    projectid: %v
    region: %v`,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.descrition, func(t *testutil.T) {
			projectID := "k8s-skaffold"
			region := "us-central1"
			jobName := fmt.Sprintf("job-%v", uuid.New().String())
			skaffoldCfg := fmt.Sprintf(test.skaffoldCfg, latest.Version, projectID, region)
			jobManifest := fmt.Sprintf(test.jobManifest, jobName)

			tmpDir := t.NewTempDir()
			tmpDir.Write("skaffold.yaml", skaffoldCfg)
			tmpDir.Write("job.yaml", jobManifest)

			skaffold.Run().InDir(tmpDir.Root()).RunOrFail(t.T)
			t.Cleanup(func() {
				skaffold.Delete(test.args...).InDir(tmpDir.Root()).RunOrFail(t.T)
			})

			job, err := getJob(context.Background(), projectID, region, jobName)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(job.Spec.Template.Spec.Template.Spec.MaxRetries, test.expectedMaxRetries); diff != "" {
				t.Fatalf("Job MaxRetries differ (-got,+want):\n%s", diff)
			}
		})
	}
}

// TODO: remove nolint when test is unskipped
//
//nolint:unused
func getRunService(ctx context.Context, project, region, service string) (*run.Service, error) {
	crclient, err := run.NewService(ctx, gcp.ClientOptions(ctx)...)
	if err != nil {
		return nil, err
	}
	sName := fmt.Sprintf("projects/%s/locations/%s/services/%s", project, region, service)
	call := crclient.Projects.Locations.Services.Get(sName)
	return call.Do()
}

func getJob(ctx context.Context, project, region, job string) (*run.Job, error) {
	cOptions := []option.ClientOption{option.WithEndpoint(fmt.Sprintf("%s-run.googleapis.com", region))}
	cOptions = append(gcp.ClientOptions(ctx), cOptions...)
	crclient, err := run.NewService(ctx, cOptions...)
	if err != nil {
		return nil, err
	}
	jName := fmt.Sprintf("namespaces/%v/jobs/%v", project, job)
	call := crclient.Namespaces.Jobs.Get(jName)
	return call.Do()
}

func getWorkerPool(ctx context.Context, project, region, workerpool string) (*run.WorkerPool, error) {
	cOptions := []option.ClientOption{option.WithEndpoint(fmt.Sprintf("%s-run.googleapis.com", region))}
	cOptions = append(gcp.ClientOptions(ctx), cOptions...)
	crclient, err := run.NewService(ctx, cOptions...)
	if err != nil {
		return nil, err
	}
	wpName := fmt.Sprintf("namespaces/%v/jobs/%v", project, workerpool)
	call := crclient.Namespaces.Workerpools.Get(wpName)
	return call.Do()
}

// TODO: remove nolint when test is unskipped
//
//nolint:unused
func checkReady(svc *run.Service) error {
	var ready *run.GoogleCloudRunV1Condition
	for _, cond := range svc.Status.Conditions {
		if cond.Type == "Ready" {
			ready = cond
		}
	}
	if ready == nil {
		return fmt.Errorf("ready condition not found in service: %v", svc)
	}
	if ready.Status != "True" {
		return fmt.Errorf("expected ready status of true, got %s with reason %s", ready.Status, ready.Message)
	}
	return nil
}

func checkWorkerPoolReadyStatus(svc *run.WorkerPool) error {
	var ready *run.GoogleCloudRunV1Condition
	for _, cond := range svc.Status.Conditions {
		if cond.Type == "Ready" {
			ready = cond
		}
	}
	if ready == nil {
		return fmt.Errorf("ready condition not found in workerpool: %v", svc)
	}
	if ready.Status != "True" {
		return fmt.Errorf("expected ready status of true, got %s with reason %s", ready.Status, ready.Message)
	}
	return nil
}
