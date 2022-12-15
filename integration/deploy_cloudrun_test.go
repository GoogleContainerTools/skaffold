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

	"google.golang.org/api/run/v1"

	"github.com/GoogleContainerTools/skaffold/v2/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/gcp"
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
