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

package cloudrun

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/api/option"
	"google.golang.org/api/run/v1"
	"google.golang.org/protobuf/testing/protocmp"
	k8syaml "sigs.k8s.io/yaml"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/label"
	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

const (
	configName = "default"
)

var defaultStatusCheckDeadline = 10 * time.Minute

func TestDeployService(tOuter *testing.T) {
	tests := []struct {
		description         string
		toDeploy            *run.Service
		defaultProject      string
		region              string
		statusCheckDeadline time.Duration
		tolerateFailures    bool
		statusCheck         *bool
		expectedPath        string
		httpErr             int
		errCode             proto.StatusCode
	}{
		{
			description:         "test deploy",
			defaultProject:      "testProject",
			region:              "us-central1",
			expectedPath:        "/v1/projects/testProject/locations/us-central1/services",
			statusCheck:         util.Ptr(true),
			statusCheckDeadline: defaultStatusCheckDeadline,
			toDeploy: &run.Service{
				ApiVersion: "serving.knative.dev/v1",
				Kind:       "Service",
				Metadata: &run.ObjectMeta{
					Name: "test-service",
				},
			},
		},
		{
			description:         "test deploy with status check deadline set to a non default value",
			defaultProject:      "testProject",
			region:              "us-central1",
			expectedPath:        "/v1/projects/testProject/locations/us-central1/services",
			statusCheck:         util.Ptr(true),
			statusCheckDeadline: 15 * time.Minute,
			toDeploy: &run.Service{
				ApiVersion: "serving.knative.dev/v1",
				Kind:       "Service",
				Metadata: &run.ObjectMeta{
					Name: "test-service",
				},
			},
		},
		{
			description:         "test deploy with tolerateFailures set to true",
			defaultProject:      "testProject",
			region:              "us-central1",
			expectedPath:        "/v1/projects/testProject/locations/us-central1/services",
			statusCheckDeadline: 15 * time.Minute,
			statusCheck:         util.Ptr(true),
			tolerateFailures:    true,
			toDeploy: &run.Service{
				ApiVersion: "serving.knative.dev/v1",
				Kind:       "Service",
				Metadata: &run.ObjectMeta{
					Name: "test-service",
				},
			},
		},
		{
			description:    "test deploy with statusCheck set to false",
			defaultProject: "testProject",
			region:         "us-central1",
			expectedPath:   "/v1/projects/testProject/locations/us-central1/services",
			statusCheck:    util.Ptr(false),
			toDeploy: &run.Service{
				ApiVersion: "serving.knative.dev/v1",
				Kind:       "Service",
				Metadata: &run.ObjectMeta{
					Name: "test-service",
				},
			},
		},
		{
			description:         "test deploy with specified project",
			defaultProject:      "testProject",
			region:              "us-central1",
			statusCheckDeadline: defaultStatusCheckDeadline,
			expectedPath:        "/v1/projects/testProject/locations/us-central1/services",
			statusCheck:         util.Ptr(true),
			toDeploy: &run.Service{
				ApiVersion: "serving.knative.dev/v1",
				Kind:       "Service",
				Metadata: &run.ObjectMeta{
					Name:      "test-service",
					Namespace: "my-project",
				},
			},
		},
		{
			description:         "test permission denied on deploy errors",
			defaultProject:      "testProject",
			region:              "us-central1",
			statusCheckDeadline: defaultStatusCheckDeadline,
			httpErr:             http.StatusUnauthorized,
			statusCheck:         util.Ptr(true),
			toDeploy: &run.Service{
				ApiVersion: "serving.knative.dev/v1",
				Kind:       "Service",
				Metadata: &run.ObjectMeta{
					Name:      "test-service",
					Namespace: "my-project",
				},
			},
			errCode: proto.StatusCode_DEPLOY_CLOUD_RUN_GET_SERVICE_ERR,
		},
		{
			description:         "test no project specified",
			region:              "us-central1",
			statusCheckDeadline: defaultStatusCheckDeadline,
			statusCheck:         util.Ptr(true),
			toDeploy: &run.Service{
				ApiVersion: "serving.knative.dev/v1",
				Kind:       "Service",
				Metadata: &run.ObjectMeta{
					Name: "test-service",
				},
			},
			errCode: proto.StatusCode_DEPLOY_READ_MANIFEST_ERR,
		},
	}
	for _, test := range tests {
		testutil.Run(tOuter, test.description, func(t *testutil.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if test.httpErr != 0 {
					http.Error(w, "test expecting error", test.httpErr)
					return
				}
				if r.URL.Path != test.expectedPath {
					http.Error(w, "unexpected path: "+r.URL.Path, http.StatusNotFound)
				}
				var service run.Service
				body, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, "Unable to read body: "+err.Error(), http.StatusInternalServerError)
					return
				}
				if err = json.Unmarshal(body, &service); err != nil {
					http.Error(w, "Unable to parse service: "+err.Error(), http.StatusBadRequest)
					return
				}
				b, err := json.Marshal(service)
				if err != nil {
					http.Error(w, "unable to marshal response: "+err.Error(), http.StatusInternalServerError)
					return
				}
				w.Write(b)
			}))

			deployer, _ := NewDeployer(
				&runcontext.RunContext{},
				&label.DefaultLabeller{},
				&latest.CloudRunDeploy{
					ProjectID: test.defaultProject,
					Region:    test.region},
				configName,
				test.statusCheckDeadline,
				test.tolerateFailures,
				test.statusCheck)
			deployer.clientOptions = append(deployer.clientOptions, option.WithEndpoint(ts.URL), option.WithoutAuthentication())
			deployer.useGcpOptions = false
			manifestList, _ := json.Marshal(test.toDeploy)
			manifestsByConfig := manifest.NewManifestListByConfig()
			manifestsByConfig.Add(configName, manifest.ManifestList{manifestList})
			err := deployer.Deploy(context.Background(), os.Stderr, []graph.Artifact{}, manifestsByConfig)
			if test.errCode == proto.StatusCode_OK && err != nil {
				t.Fatalf("Expected success but got err: %v", err)
			} else if test.errCode != proto.StatusCode_OK {
				if err == nil {
					t.Fatalf("Expected status code %s but got success", test.errCode)
				}
				sErr := err.(sErrors.Error)
				if sErr.StatusCode() != test.errCode {
					t.Fatalf("Expected status code %v but got %v", test.errCode, sErr.StatusCode())
				}
			}
		})
	}
}

func TestDeployJob(tOuter *testing.T) {
	tests := []struct {
		description        string
		toDeploy           *run.Job
		defaultProject     string
		region             string
		expectedPath       string
		httpErr            int
		errCode            proto.StatusCode
		expectedMaxRetries *float64
	}{
		{
			description:    "test deploy",
			defaultProject: "testProject",
			region:         "us-central1",
			expectedPath:   "/apis/run.googleapis.com/v1/namespaces/testProject/jobs",
			toDeploy: &run.Job{
				ApiVersion: "run.googleapis.com/v1",
				Kind:       "Job",
				Metadata: &run.ObjectMeta{
					Name: "test-service",
				},
			},
		},
		{
			description:    "test deploy with specified project",
			defaultProject: "testProject",
			region:         "us-central1",
			expectedPath:   "/apis/run.googleapis.com/v1/namespaces/testProject/jobs",
			toDeploy: &run.Job{
				ApiVersion: "run.googleapis.com/v1",
				Kind:       "Job",
				Metadata: &run.ObjectMeta{
					Name:      "test-service",
					Namespace: "my-project",
				},
			},
		},
		{
			description:    "test permission denied on deploy errors",
			defaultProject: "testProject",
			region:         "us-central1",
			httpErr:        http.StatusUnauthorized,
			toDeploy: &run.Job{
				ApiVersion: "run.googleapis.com/v1",
				Kind:       "Job",
				Metadata: &run.ObjectMeta{
					Name:      "test-service",
					Namespace: "my-project",
				},
			},
			errCode: proto.StatusCode_DEPLOY_CLOUD_RUN_GET_SERVICE_ERR,
		},
		{
			description: "test no project specified",
			region:      "us-central1",
			toDeploy: &run.Job{
				ApiVersion: "run.googleapis.com/v1",
				Kind:       "Job",
				Metadata: &run.ObjectMeta{
					Name: "test-service",
				},
			},
			errCode: proto.StatusCode_DEPLOY_READ_MANIFEST_ERR,
		},
		{
			description:        "test deploy with maxRetries field set to 0",
			defaultProject:     "testProject",
			region:             "us-central1",
			expectedPath:       "/apis/run.googleapis.com/v1/namespaces/testProject/jobs",
			expectedMaxRetries: util.Ptr[float64](0),
			toDeploy: &run.Job{
				ApiVersion: "run.googleapis.com/v1",
				Kind:       "Job",
				Metadata: &run.ObjectMeta{
					Name: "test-service",
				},
				Spec: &run.JobSpec{
					Template: &run.ExecutionTemplateSpec{
						Spec: &run.ExecutionSpec{
							Template: &run.TaskTemplateSpec{
								Spec: &run.TaskSpec{
									MaxRetries:      0,
									ForceSendFields: []string{"MaxRetries"},
								},
							},
						},
					},
				},
			},
		},
		{
			description:        "test deploy with maxRetries field set to 5",
			defaultProject:     "testProject",
			region:             "us-central1",
			expectedPath:       "/apis/run.googleapis.com/v1/namespaces/testProject/jobs",
			expectedMaxRetries: util.Ptr[float64](5),
			toDeploy: &run.Job{
				ApiVersion: "run.googleapis.com/v1",
				Kind:       "Job",
				Metadata: &run.ObjectMeta{
					Name: "test-service",
				},
				Spec: &run.JobSpec{
					Template: &run.ExecutionTemplateSpec{
						Spec: &run.ExecutionSpec{
							Template: &run.TaskTemplateSpec{
								Spec: &run.TaskSpec{
									MaxRetries:      5,
									ForceSendFields: []string{"MaxRetries"},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(tOuter, test.description, func(t *testutil.T) {
			var jobReceivedInServer []byte
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if test.httpErr != 0 {
					http.Error(w, "test expecting error", test.httpErr)
					return
				}
				if r.URL.Path != test.expectedPath {
					http.Error(w, "unexpected path: "+r.URL.Path, http.StatusNotFound)
					return
				}
				var job run.Job
				body, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, "Unable to read body: "+err.Error(), http.StatusInternalServerError)
					return
				}
				if err = json.Unmarshal(body, &job); err != nil {
					http.Error(w, "Unable to parse service: "+err.Error(), http.StatusBadRequest)
					return
				}
				b, err := json.Marshal(job)
				if err != nil {
					http.Error(w, "unable to marshal response: "+err.Error(), http.StatusInternalServerError)
					return
				}

				jobReceivedInServer = body
				w.Write(b)
			}))

			deployer, _ := NewDeployer(
				&runcontext.RunContext{},
				&label.DefaultLabeller{},
				&latest.CloudRunDeploy{
					ProjectID: test.defaultProject,
					Region:    test.region,
				},
				configName,
				defaultStatusCheckDeadline,
				false,
				util.Ptr(true))
			deployer.clientOptions = append(deployer.clientOptions, option.WithEndpoint(ts.URL), option.WithoutAuthentication())
			deployer.useGcpOptions = false
			manifestList, _ := k8syaml.Marshal(test.toDeploy)
			manifestsByConfig := manifest.NewManifestListByConfig()
			manifestsByConfig.Add(configName, manifest.ManifestList{manifestList})
			err := deployer.Deploy(context.Background(), os.Stderr, []graph.Artifact{}, manifestsByConfig)
			if test.errCode == proto.StatusCode_OK && err != nil {
				t.Fatalf("Expected success but got err: %v", err)
			} else if test.errCode != proto.StatusCode_OK {
				if err == nil {
					t.Fatalf("Expected status code %s but got success", test.errCode)
				}
				sErr := err.(sErrors.Error)
				if sErr.StatusCode() != test.errCode {
					t.Fatalf("Expected status code %v but got %v", test.errCode, sErr.StatusCode())
				}
			}

			if test.errCode == proto.StatusCode_OK {
				checkMaxRetriesValue(t, jobReceivedInServer, test.expectedMaxRetries)
			}
		})
	}
}

func TestDeployWorkerPool(tOuter *testing.T) {
	tests := []struct {
		description    string
		toDeploy       *run.WorkerPool
		defaultProject string
		region         string
		expectedPath   string
		httpErr        int
		errCode        proto.StatusCode
	}{
		{
			description:    "test deploy workerpool",
			defaultProject: "testProject",
			region:         "us-central1",
			expectedPath:   "/apis/run.googleapis.com/v1/namespaces/testProject/workerpools",
			toDeploy: &run.WorkerPool{
				ApiVersion: "run.googleapis.com/v1",
				Kind:       "WorkerPool",
				Metadata: &run.ObjectMeta{
					Name: "test-wp",
				},
			},
		},
		{
			description:    "test deploy workerpool with specified project",
			defaultProject: "testProject",
			region:         "us-central1",
			expectedPath:   "/apis/run.googleapis.com/v1/namespaces/testProject/workerpools",
			toDeploy: &run.WorkerPool{
				ApiVersion: "run.googleapis.com/v1",
				Kind:       "WorkerPool",
				Metadata: &run.ObjectMeta{
					Name:      "test-wp",
					Namespace: "my-project",
				},
			},
		},
		{
			description:    "test permission denied on deploy workerpool errors",
			defaultProject: "testProject",
			region:         "us-central1",
			httpErr:        http.StatusUnauthorized,
			toDeploy: &run.WorkerPool{
				ApiVersion: "run.googleapis.com/v1",
				Kind:       "WorkerPool",
				Metadata: &run.ObjectMeta{
					Name:      "test-wp",
					Namespace: "my-project",
				},
			},
			errCode: proto.StatusCode_DEPLOY_CLOUD_RUN_GET_WORKER_POOL_ERR,
		},
		{
			description: "test no project specified for workerpool",
			region:      "us-central1",
			toDeploy: &run.WorkerPool{
				ApiVersion: "run.googleapis.com/v1",
				Kind:       "WorkerPool",
				Metadata: &run.ObjectMeta{
					Name: "test-wp",
				},
			},
			errCode: proto.StatusCode_DEPLOY_READ_MANIFEST_ERR,
		},
	}
	for _, test := range tests {
		testutil.Run(tOuter, test.description, func(t *testutil.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if test.httpErr != 0 {
					http.Error(w, "test expecting error", test.httpErr)
					return
				}
				if r.URL.Path != test.expectedPath {
					http.Error(w, "unexpected path: "+r.URL.Path, http.StatusNotFound)
					return
				}
				var wp run.WorkerPool
				body, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, "Unable to read body: "+err.Error(), http.StatusInternalServerError)
					return
				}
				if err = json.Unmarshal(body, &wp); err != nil {
					http.Error(w, "Unable to parse workerpool: "+err.Error(), http.StatusBadRequest)
					return
				}
				b, err := json.Marshal(wp)
				if err != nil {
					http.Error(w, "unable to marshal response: "+err.Error(), http.StatusInternalServerError)
					return
				}
				w.Write(b)
			}))

			deployer, _ := NewDeployer(
				&runcontext.RunContext{},
				&label.DefaultLabeller{},
				&latest.CloudRunDeploy{
					ProjectID: test.defaultProject,
					Region:    test.region,
				},
				configName,
				defaultStatusCheckDeadline,
				false,
				util.Ptr(true))
			deployer.clientOptions = append(deployer.clientOptions, option.WithEndpoint(ts.URL), option.WithoutAuthentication())
			deployer.useGcpOptions = false
			manifestList, _ := json.Marshal(test.toDeploy)
			manifestsByConfig := manifest.NewManifestListByConfig()
			manifestsByConfig.Add(configName, manifest.ManifestList{manifestList})
			err := deployer.Deploy(context.Background(), os.Stderr, []graph.Artifact{}, manifestsByConfig)
			if test.errCode == proto.StatusCode_OK && err != nil {
				t.Fatalf("Expected success but got err: %v", err)
			} else if test.errCode != proto.StatusCode_OK {
				if err == nil {
					t.Fatalf("Expected status code %s but got success", test.errCode)
				}
				sErr := err.(sErrors.Error)
				if sErr.StatusCode() != test.errCode {
					t.Fatalf("Expected status code %v but got %v", test.errCode, sErr.StatusCode())
				}
			}
		})
	}
}

func checkMaxRetriesValue(t *testutil.T, serverJob []byte, expectedMaxRetries *float64) {
	maxRetriesPath := []string{"spec", "template", "spec", "template", "spec"}
	var foundMaxRetries *float64
	fields := make(map[string]interface{})

	if err := json.Unmarshal(serverJob, &fields); err != nil {
		t.Fatalf("Error unmarshaling job from server: %v", err)
	}

	for _, field := range maxRetriesPath {
		value := fields[field]
		child, ok := value.(map[string]interface{})
		if !ok {
			fields = nil
			break
		}
		fields = child
	}

	mxRetryVal := fields["maxRetries"]
	if val, ok := mxRetryVal.(float64); ok {
		foundMaxRetries = util.Ptr(val)
	}
	if diff := cmp.Diff(expectedMaxRetries, foundMaxRetries); diff != "" {
		t.Fatalf("MaxRetries don't match (+got-want):\n%v", diff)
	}
}

func TestDeployRewrites(tOuter *testing.T) {
	tests := []struct {
		description    string
		toDeploy       *run.Service
		defaultProject string
		region         string
		expected       *run.Service
	}{
		{
			description: "override run-id in service and template",
			toDeploy: &run.Service{
				ApiVersion: "serving.knative.dev/v1",
				Kind:       "Service",
				Metadata: &run.ObjectMeta{
					Labels: map[string]string{
						"skaffold.dev/run-id": "abc123",
					},
					Name: "test-service",
				},
				Spec: &run.ServiceSpec{
					Template: &run.RevisionTemplate{
						Metadata: &run.ObjectMeta{
							Labels: map[string]string{
								"skaffold.dev/run-id": "abc123",
							},
						},
					},
				},
			},
			defaultProject: "test-project",
			region:         "us-central1",
			expected: &run.Service{
				ApiVersion: "serving.knative.dev/v1",
				Kind:       "Service",
				Metadata: &run.ObjectMeta{
					Labels: map[string]string{
						"run-id": "abc123",
					},
					Name:      "test-service",
					Namespace: "test-project",
				},
				Spec: &run.ServiceSpec{
					Template: &run.RevisionTemplate{
						Metadata: &run.ObjectMeta{
							Labels: map[string]string{
								"run-id": "abc123",
							},
						},
					},
				},
			},
		},
		{
			description: "test deploy with overridden project",
			toDeploy: &run.Service{
				ApiVersion: "serving.knative.dev/v1",
				Kind:       "Service",
				Metadata: &run.ObjectMeta{
					Name:      "test-service",
					Namespace: "my-project",
				},
			},
			defaultProject: "test-project",
			region:         "us-central1",
			expected: &run.Service{
				ApiVersion: "serving.knative.dev/v1",
				Kind:       "Service",
				Metadata: &run.ObjectMeta{
					Name:      "test-service",
					Namespace: "test-project",
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(tOuter, test.description, func(t *testutil.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "GET" {
					http.Error(w, "want to return empty default", http.StatusNotFound)
					return
				}
				var service run.Service
				body, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, "Unable to read body: "+err.Error(), http.StatusInternalServerError)
					return
				}
				if err = json.Unmarshal(body, &service); err != nil {
					http.Error(w, "Unable to parse service: "+err.Error(), http.StatusBadRequest)
					return
				}
				if test.expected != nil {
					if diff := cmp.Diff(*test.expected, service, protocmp.Transform()); diff != "" {
						http.Error(w, "Expected equal but got diff "+diff, http.StatusBadRequest)
						return
					}
				}
				b, err := json.Marshal(service)
				if err != nil {
					http.Error(w, "unable to marshal response: "+err.Error(), http.StatusInternalServerError)
					return
				}
				w.Write(b)
			}))
			deployer, _ := NewDeployer(
				&runcontext.RunContext{},
				&label.DefaultLabeller{},
				&latest.CloudRunDeploy{
					ProjectID: test.defaultProject,
					Region:    test.region,
				},
				"",
				defaultStatusCheckDeadline,
				false,
				util.Ptr(true))
			deployer.clientOptions = append(deployer.clientOptions, option.WithEndpoint(ts.URL), option.WithoutAuthentication())
			deployer.useGcpOptions = false
			m, _ := json.Marshal(test.toDeploy)
			manifests := [][]byte{m}
			manifestByConfig := manifest.NewManifestListByConfig()
			manifestByConfig.Add("", manifests)
			err := deployer.Deploy(context.Background(), os.Stderr, []graph.Artifact{}, manifestByConfig)
			if err != nil {
				t.Fatalf("Expected success but got err: %v", err)
			}
		})
	}
}

func TestCleanupService(tOuter *testing.T) {
	tests := []struct {
		description    string
		toDelete       *run.Service
		defaultProject string
		region         string
		expectedPath   string
		httpErr        int
	}{
		{
			description:    "test cleanup",
			defaultProject: "testProject",
			region:         "us-central1",
			expectedPath:   "/v1/projects/testProject/locations/us-central1/services/test-service",
			toDelete: &run.Service{
				ApiVersion: "serving.knative.dev/v1",
				Kind:       "Service",
				Metadata: &run.ObjectMeta{
					Name: "test-service",
				},
			},
		},
		{
			description:    "test cleanup with specified project",
			defaultProject: "testProject",
			region:         "us-central1",
			expectedPath:   "/v1/projects/testProject/locations/us-central1/services/test-service",
			toDelete: &run.Service{
				ApiVersion: "serving.knative.dev/v1",
				Kind:       "Service",
				Metadata: &run.ObjectMeta{
					Name:      "test-service",
					Namespace: "my-project",
				},
			},
		},
		{
			description:    "test cleanup fails",
			defaultProject: "testProject",
			region:         "us-central1",
			expectedPath:   "/v1/projects/testProject/locations/us-central1/services/test-service",
			toDelete: &run.Service{
				ApiVersion: "serving.knative.dev/v1",
				Kind:       "Service",
				Metadata: &run.ObjectMeta{
					Name: "test-service",
				},
			},
			httpErr: http.StatusUnauthorized,
		},
	}
	for _, test := range tests {
		testutil.Run(tOuter, test.description, func(t *testutil.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if test.httpErr != 0 {
					http.Error(w, "Expected http error", test.httpErr)
					return
				}
				if r.URL.Path != test.expectedPath {
					http.Error(w, "unexpected path: "+r.URL.Path, http.StatusNotFound)
					return
				}
				response := &run.Status{}
				b, err := json.Marshal(response)
				if err != nil {
					http.Error(w, "unable to marshal response: "+err.Error(), http.StatusInternalServerError)
					return
				}
				w.Write(b)
			}))
			defer ts.Close()
			deployer, _ := NewDeployer(
				&runcontext.RunContext{},
				&label.DefaultLabeller{},
				&latest.CloudRunDeploy{
					ProjectID: test.defaultProject,
					Region:    test.region,
				},
				configName,
				defaultStatusCheckDeadline,
				false,
				util.Ptr(true))
			deployer.clientOptions = append(deployer.clientOptions, option.WithEndpoint(ts.URL), option.WithoutAuthentication())
			deployer.useGcpOptions = false
			manifestListByConfig := manifest.NewManifestListByConfig()
			manifest, _ := json.Marshal(test.toDelete)
			manifests := [][]byte{manifest}
			manifestListByConfig.Add(configName, manifests)
			err := deployer.Cleanup(context.Background(), os.Stderr, false, manifestListByConfig)
			if test.httpErr == 0 && err != nil {
				t.Fatalf("Expected success but got err: %v", err)
			} else if test.httpErr != 0 && err == nil {
				t.Fatalf("Expected HTTP Error %s but got success", http.StatusText(test.httpErr))
			}
		})
	}
}

func TestCleanupJob(tOuter *testing.T) {
	tests := []struct {
		description    string
		toDelete       *run.Job
		defaultProject string
		region         string
		expectedPath   string
		httpErr        int
	}{
		{
			description:    "test cleanup",
			defaultProject: "testProject",
			region:         "us-central1",
			expectedPath:   "/apis/run.googleapis.com/v1/namespaces/testProject/jobs/test-job",
			toDelete: &run.Job{
				ApiVersion: "run.googleapis.com/v1",
				Kind:       "Job",
				Metadata: &run.ObjectMeta{
					Name: "test-job",
				},
			},
		},
		{
			description:    "test cleanup fails",
			defaultProject: "testProject",
			region:         "us-central1",
			expectedPath:   "/apis/run.googleapis.com/v1/namespaces/testProject/jobs/test-job",
			toDelete: &run.Job{
				ApiVersion: "run.googleapis.com/v1",
				Kind:       "Job",
				Metadata: &run.ObjectMeta{
					Name: "test-job",
				},
			},
			httpErr: http.StatusUnauthorized,
		},
	}
	for _, test := range tests {
		testutil.Run(tOuter, test.description, func(t *testutil.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if test.httpErr != 0 {
					http.Error(w, "Expected http error", test.httpErr)
					return
				}
				if r.URL.Path != test.expectedPath {
					http.Error(w, "unexpected path: "+r.URL.Path, http.StatusNotFound)
					return
				}
				response := &run.Status{}
				b, err := json.Marshal(response)
				if err != nil {
					http.Error(w, "unable to marshal response: "+err.Error(), http.StatusInternalServerError)
					return
				}
				w.Write(b)
			}))
			defer ts.Close()
			deployer, _ := NewDeployer(
				&runcontext.RunContext{},
				&label.DefaultLabeller{},
				&latest.CloudRunDeploy{
					ProjectID: test.defaultProject,
					Region:    test.region,
				},
				configName,
				defaultStatusCheckDeadline,
				false,
				util.Ptr(true))
			deployer.clientOptions = append(deployer.clientOptions, option.WithEndpoint(ts.URL), option.WithoutAuthentication())
			deployer.useGcpOptions = false
			manifestListByConfig := manifest.NewManifestListByConfig()
			manifest, _ := json.Marshal(test.toDelete)
			manifests := [][]byte{manifest}
			manifestListByConfig.Add(configName, manifests)
			err := deployer.Cleanup(context.Background(), os.Stderr, false, manifestListByConfig)
			if test.httpErr == 0 && err != nil {
				t.Fatalf("Expected success but got err: %v", err)
			} else if test.httpErr != 0 && err == nil {
				t.Fatalf("Expected HTTP Error %s but got success", http.StatusText(test.httpErr))
			}
		})
	}
}

func TestCleanupWorkerPool(tOuter *testing.T) {
	tests := []struct {
		description    string
		toDelete       *run.WorkerPool
		defaultProject string
		region         string
		expectedPath   string
		httpErr        int
	}{
		{
			description:    "test workerpool cleanup",
			defaultProject: "testProject",
			region:         "us-central1",
			expectedPath:   "/apis/run.googleapis.com/v1/namespaces/testProject/workerpools/test-wp",
			toDelete: &run.WorkerPool{
				ApiVersion: "run.googleapis.com/v1",
				Kind:       "WorkerPool",
				Metadata: &run.ObjectMeta{
					Name: "test-wp",
				},
			},
		},
		{
			description:    "test workerpool cleanup fails",
			defaultProject: "testProject",
			region:         "us-central1",
			expectedPath:   "/apis/run.googleapis.com/v1/namespaces/testProject/workerpools/test-wp",
			toDelete: &run.WorkerPool{
				ApiVersion: "run.googleapis.com/v1",
				Kind:       "WorkerPool",
				Metadata: &run.ObjectMeta{
					Name: "test-wp",
				},
			},
			httpErr: http.StatusUnauthorized,
		},
	}
	for _, test := range tests {
		testutil.Run(tOuter, test.description, func(t *testutil.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if test.httpErr != 0 {
					http.Error(w, "Expected http error", test.httpErr)
					return
				}
				if r.URL.Path != test.expectedPath {
					http.Error(w, "unexpected path: "+r.URL.Path, http.StatusNotFound)
					return
				}
				response := &run.Status{}
				b, err := json.Marshal(response)
				if err != nil {
					http.Error(w, "unable to marshal response: "+err.Error(), http.StatusInternalServerError)
					return
				}
				w.Write(b)
			}))
			defer ts.Close()
			deployer, _ := NewDeployer(
				&runcontext.RunContext{},
				&label.DefaultLabeller{},
				&latest.CloudRunDeploy{
					ProjectID: test.defaultProject,
					Region:    test.region,
				},
				configName,
				defaultStatusCheckDeadline,
				false,
				util.Ptr(true))
			deployer.clientOptions = append(deployer.clientOptions, option.WithEndpoint(ts.URL), option.WithoutAuthentication())
			deployer.useGcpOptions = false
			manifestListByConfig := manifest.NewManifestListByConfig()
			manifest, _ := json.Marshal(test.toDelete)
			manifests := [][]byte{manifest}
			manifestListByConfig.Add(configName, manifests)
			err := deployer.Cleanup(context.Background(), os.Stderr, false, manifestListByConfig)
			if test.httpErr == 0 && err != nil {
				t.Fatalf("Expected success but got err: %v", err)
			} else if test.httpErr != 0 && err == nil {
				t.Fatalf("Expected HTTP Error %s but got success", http.StatusText(test.httpErr))
			}
		})
	}
}

func TestCleanupMultipleResources(tOuter *testing.T) {
	tests := []struct {
		description    string
		toDelete       []interface{}
		defaultProject string
		region         string
		expectedPath   map[string]int
	}{
		{
			description:    "test cleanup",
			defaultProject: "testProject",
			region:         "us-central1",
			expectedPath:   map[string]int{"/apis/run.googleapis.com/v1/namespaces/testProject/jobs/test-job": 1, "/v1/projects/testProject/locations/us-central1/services/test-service": 1, "/apis/run.googleapis.com/v1/namespaces/testProject/workerpools/test-wp": 1},
			toDelete: []interface{}{&run.Job{
				ApiVersion: "run.googleapis.com/v1",
				Kind:       "Job",
				Metadata: &run.ObjectMeta{
					Name: "test-job",
				},
			},
				&run.Service{
					ApiVersion: "serving.knative.dev/v1",
					Kind:       "Service",
					Metadata: &run.ObjectMeta{
						Name: "test-service",
					},
				},
				&run.WorkerPool{
					ApiVersion: "run.googleapis.com/v1",
					Kind:       "WorkerPool",
					Metadata: &run.ObjectMeta{
						Name: "test-wp",
					},
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(tOuter, test.description, func(t *testutil.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if count, ok := test.expectedPath[r.URL.Path]; !ok || count <= 0 {
					http.Error(w, "unexpected path: "+r.URL.Path, http.StatusNotFound)
					return
				}
				test.expectedPath[r.URL.Path]--
				response := &run.Status{}
				b, err := json.Marshal(response)
				if err != nil {
					http.Error(w, "unable to marshal response: "+err.Error(), http.StatusInternalServerError)
					return
				}
				w.Write(b)
			}))
			defer ts.Close()
			deployer, _ := NewDeployer(
				&runcontext.RunContext{},
				&label.DefaultLabeller{},
				&latest.CloudRunDeploy{
					ProjectID: test.defaultProject,
					Region:    test.region,
				},
				configName,
				defaultStatusCheckDeadline,
				false,
				util.Ptr(true))
			deployer.clientOptions = append(deployer.clientOptions, option.WithEndpoint(ts.URL), option.WithoutAuthentication())
			deployer.useGcpOptions = false
			manifestListByConfig := manifest.NewManifestListByConfig()
			var manifests [][]byte
			for _, res := range test.toDelete {
				manifest, err := json.Marshal(res)
				if err != nil {
					t.Fatalf("error marshaling manifest: %v", err)
				}
				manifests = append(manifests, manifest)
			}
			manifestListByConfig.Add(configName, manifests)
			err := deployer.Cleanup(context.Background(), os.Stderr, false, manifestListByConfig)
			if err != nil {
				t.Fatalf("Expected success but got err: %v", err)
			}
			for key, val := range test.expectedPath {
				if val > 0 {
					t.Fatalf("Missing expected call for path %s", key)
				}
			}
		})
	}
}
