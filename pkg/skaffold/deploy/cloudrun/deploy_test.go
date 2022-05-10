package cloudrun

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"google.golang.org/api/option"
	"google.golang.org/api/run/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestDeploy(tOuter *testing.T) {
	tests := []struct {
		description    string
		toDeploy       *run.Service
		defaultProject string
		region         string
		expectedPath   string
		httpErr        int
	}{
		{
			description:    "test deploy",
			defaultProject: "testProject",
			region:         "us-central1",
			expectedPath:   "/v1/projects/testProject/locations/us-central1/services",
			toDeploy: &run.Service{
				Metadata: &run.ObjectMeta{
					Name: "test-service",
				},
			},
		},
		{
			description:    "test deploy with specified project",
			defaultProject: "testProject",
			region:         "us-central1",
			expectedPath:   "/v1/projects/my-project/locations/us-central1/services",
			toDeploy: &run.Service{
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
				Metadata: &run.ObjectMeta{
					Name:      "test-service",
					Namespace: "my-project",
				},
			},
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

			deployer, _ := NewDeployer(&label.DefaultLabeller{}, &latest.CloudRunDeploy{DefaultProjectID: test.defaultProject, Region: test.region})
			deployer.clientOptions = append(deployer.clientOptions, option.WithEndpoint(ts.URL))
			manifest, _ := json.Marshal(test.toDeploy)
			manifests := [][]byte{manifest}
			err := deployer.Deploy(context.Background(), os.Stderr, []graph.Artifact{}, manifests)
			if test.httpErr == 0 && err != nil {
				t.Fatalf("Expected success but got err: %v", err)
			} else if test.httpErr != 0 && err == nil {
				t.Fatalf("Expected HTTP Error %s but got success", http.StatusText(test.httpErr))
			}
		})
	}
}

func TestCleanup(tOuter *testing.T) {
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
				Metadata: &run.ObjectMeta{
					Name: "test-service",
				},
			},
		},
		{
			description:    "test cleanup with specified project",
			defaultProject: "testProject",
			region:         "us-central1",
			expectedPath:   "/v1/projects/my-project/locations/us-central1/services/test-service",
			toDelete: &run.Service{
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
			deployer, _ := NewDeployer(&label.DefaultLabeller{}, &latest.CloudRunDeploy{DefaultProjectID: test.defaultProject, Region: test.region})
			deployer.clientOptions = append(deployer.clientOptions, option.WithEndpoint(ts.URL))
			manifest, _ := json.Marshal(test.toDelete)
			manifests := [][]byte{manifest}
			err := deployer.Cleanup(context.Background(), os.Stderr, false, manifests)
			if test.httpErr == 0 && err != nil {
				t.Fatalf("Expected success but got err: %v", err)
			} else if test.httpErr != 0 && err == nil {
				t.Fatalf("Expected HTTP Error %s but got success", http.StatusText(test.httpErr))
			}
		})
	}
}
