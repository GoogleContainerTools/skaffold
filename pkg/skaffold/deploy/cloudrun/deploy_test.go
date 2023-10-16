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

	"github.com/google/go-cmp/cmp"
	"google.golang.org/api/option"
	"google.golang.org/api/run/v1"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/label"
	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

const (
	configName = "default"
)

func TestDeployService(tOuter *testing.T) {
	tests := []struct {
		description    string
		toDeploy       *run.Service
		defaultProject string
		region         string
		expectedPath   string
		httpErr        int
		errCode        proto.StatusCode
	}{
		{
			description:    "test deploy",
			defaultProject: "testProject",
			region:         "us-central1",
			expectedPath:   "/v1/projects/testProject/locations/us-central1/services",
			toDeploy: &run.Service{
				ApiVersion: "serving.knative.dev/v1",
				Kind:       "Service",
				Metadata: &run.ObjectMeta{
					Name: "test-service",
				},
			},
		},
		{
			description:    "test deploy with specified project",
			defaultProject: "testProject",
			region:         "us-central1",
			expectedPath:   "/v1/projects/testProject/locations/us-central1/services",
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
			description:    "test permission denied on deploy errors",
			defaultProject: "testProject",
			region:         "us-central1",
			httpErr:        http.StatusUnauthorized,
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
			description: "test no project specified",
			region:      "us-central1",
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

			deployer, _ := NewDeployer(&runcontext.RunContext{}, &label.DefaultLabeller{}, &latest.CloudRunDeploy{ProjectID: test.defaultProject, Region: test.region}, configName)
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
		description    string
		toDeploy       *run.Job
		defaultProject string
		region         string
		expectedPath   string
		httpErr        int
		errCode        proto.StatusCode
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

			deployer, _ := NewDeployer(&runcontext.RunContext{}, &label.DefaultLabeller{}, &latest.CloudRunDeploy{ProjectID: test.defaultProject, Region: test.region}, configName)
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
			deployer, _ := NewDeployer(&runcontext.RunContext{}, &label.DefaultLabeller{}, &latest.CloudRunDeploy{ProjectID: test.defaultProject, Region: test.region}, "")
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
			deployer, _ := NewDeployer(&runcontext.RunContext{}, &label.DefaultLabeller{}, &latest.CloudRunDeploy{ProjectID: test.defaultProject, Region: test.region}, configName)
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
			deployer, _ := NewDeployer(&runcontext.RunContext{}, &label.DefaultLabeller{}, &latest.CloudRunDeploy{ProjectID: test.defaultProject, Region: test.region}, configName)
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
			expectedPath:   map[string]int{"/apis/run.googleapis.com/v1/namespaces/testProject/jobs/test-job": 1, "/v1/projects/testProject/locations/us-central1/services/test-service": 1},
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
			deployer, _ := NewDeployer(&runcontext.RunContext{}, &label.DefaultLabeller{}, &latest.CloudRunDeploy{ProjectID: test.defaultProject, Region: test.region}, configName)
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
					t.Fatalf("Missing expected call for path " + key)
				}
			}
		})
	}
}
