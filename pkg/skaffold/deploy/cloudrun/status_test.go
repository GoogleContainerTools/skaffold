package cloudrun

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	proto "github.com/GoogleContainerTools/skaffold/proto/v2"
	"github.com/GoogleContainerTools/skaffold/testutil"
	testEvent "github.com/GoogleContainerTools/skaffold/testutil/event"
	"google.golang.org/api/option"
	"google.golang.org/api/run/v1"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestPrintSummaryStatus(t *testing.T) {
	labeller := label.NewLabeller(true, nil, "run-id")

	tests := []struct {
		description string
		pending     int32
		path        string
		name        string
		ae          *proto.ActionableErr
		expected    string
	}{
		{
			description: "single resource running",
			pending:     int32(1),
			path:        "/projects/test/locations/region/services/test-service",
			name:        "test-service",
			ae: &proto.ActionableErr{
				ErrCode: proto.StatusCode_STATUSCHECK_SUCCESS,
				Message: "Service started",
			},
			expected: "Cloud Run Service test-service finished: Service started. 1/10 deployment(s) still pending\n",
		},
		{
			description: "nothing prints if cancelled",
			pending:     int32(3),
			path:        "/projects/test/locations/region/services/test-service",
			name:        "test-service",
			ae: &proto.ActionableErr{
				ErrCode: proto.StatusCode_STATUSCHECK_USER_CANCELLED,
				Message: "Deploy cancelled",
			},
			expected: "",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			res := &runResource{
				path: test.path,
				name: test.name,
				ae:   test.ae,
			}
			s := NewMonitor(labeller, []option.ClientOption{})
			out := new(bytes.Buffer)
			testEvent.InitializeState([]latest.Pipeline{{}})
			c := newCounter(10)
			c.pending = test.pending
			s.printStatusCheckSummary(out, c, res)
			t.CheckDeepEqual(test.expected, out.String())
		})
	}
}
func TestPollResourceStatus(t *testing.T) {
	labeller := label.NewLabeller(true, nil, "run-id")
	tests := []struct {
		description string
		resource    ResourceName
		responses   []run.Service
		expected    *proto.ActionableErr
		fail        bool
	}{
		{
			description: "test basic check with one resource ready",
			resource:    ResourceName{name: "test-service", path: "projects/tp/locations/tr/services/test-service"},
			responses: []run.Service{
				{
					ApiVersion: "serving.knative.dev/v1",
					Status: &run.ServiceStatus{
						Conditions: []*run.GoogleCloudRunV1Condition{
							{
								Type:    "Ready",
								Status:  "True",
								Message: "Revision Ready",
							},
						},
					},
				},
			},
			expected: &proto.ActionableErr{Message: "Service started", ErrCode: proto.StatusCode_STATUSCHECK_SUCCESS},
		},
		{
			description: "test basic check with one resource going ready after 1 non-ready",
			resource:    ResourceName{name: "test-service", path: "projects/tp/locations/tr/services/test-service"},
			responses: []run.Service{
				{
					ApiVersion: "serving.knative.dev/v1",
					Status: &run.ServiceStatus{
						Conditions: []*run.GoogleCloudRunV1Condition{
							{
								Type:    "Ready",
								Status:  "Unknown",
								Message: "Deploying Revision",
							},
						},
					},
				},
				{
					ApiVersion: "serving.knative.dev/v1",
					Status: &run.ServiceStatus{
						Conditions: []*run.GoogleCloudRunV1Condition{
							{
								Type:    "Ready",
								Status:  "True",
								Message: "Revision Ready",
							},
						},
					},
				},
			},
			expected: &proto.ActionableErr{Message: "Service started", ErrCode: proto.StatusCode_STATUSCHECK_SUCCESS},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			checkTimes := 0
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if checkTimes >= len(test.responses) {
					checkTimes = len(test.responses) - 1
				}
				resp := test.responses[checkTimes]
				checkTimes++
				b, err := json.Marshal(resp)
				if err != nil {
					http.Error(w, "unable to marshal response: "+err.Error(), http.StatusInternalServerError)
					return
				}
				w.Write(b)
			}))
			testEvent.InitializeState([]latest.Pipeline{{}})

			monitor := NewMonitor(labeller, []option.ClientOption{option.WithEndpoint(ts.URL)})
			// speed up checks for tests
			monitor.pollPeriod = 1 * time.Second
			monitor.reportStatusTime = 500 * time.Millisecond
			monitor.statusCheckDeadline = 5 * time.Second
			resource := &runResource{path: test.resource.path, name: test.resource.name}
			ctx := context.Background()
			resource.pollResourceStatus(ctx, 5*time.Second, 1*time.Second, []option.ClientOption{option.WithEndpoint(ts.URL)})
			t.CheckDeepEqual(test.expected, resource.ae, protocmp.Transform())
		})
	}
}

func TestMontiorCheck(t *testing.T) {
	labeller := label.NewLabeller(true, nil, "run-id")
	tests := []struct {
		description string
		resources   []ResourceName
		responses   map[string]([]run.Service)
		expected    string
		fail        bool
	}{
		{
			description: "test basic check with one resource ready",
			resources: []ResourceName{
				{name: "test-service", path: "projects/tp/locations/tr/services/test-service"},
			},
			responses: map[string][]run.Service{
				"/v1/projects/tp/locations/tr/services/test-service": {
					{
						ApiVersion: "serving.knative.dev/v1",
						Status: &run.ServiceStatus{
							Conditions: []*run.GoogleCloudRunV1Condition{
								{
									Type:    "Ready",
									Status:  "True",
									Message: "Revision Ready",
								},
							},
						},
					},
				},
			},
			expected: "Cloud Run Service test-service finished: Service started. 0/1 deployment(s) still pending\n",
		},
		{
			description: "test basic check with one resource going ready after 1 non-ready",
			resources: []ResourceName{
				{name: "test-service", path: "projects/tp/locations/tr/services/test-service"},
			},
			responses: map[string][]run.Service{
				"/v1/projects/tp/locations/tr/services/test-service": {
					{
						ApiVersion: "serving.knative.dev/v1",
						Status: &run.ServiceStatus{
							Conditions: []*run.GoogleCloudRunV1Condition{
								{
									Type:    "Ready",
									Status:  "Unknown",
									Message: "Deploying Revision",
								},
							},
						},
					},
					{
						ApiVersion: "serving.knative.dev/v1",
						Status: &run.ServiceStatus{
							Conditions: []*run.GoogleCloudRunV1Condition{
								{
									Type:    "Ready",
									Status:  "True",
									Message: "Revision Ready",
								},
							},
						},
					},
				},
			},
			expected: ("test-service: Service starting: Deploying Revision\n" +
				"Cloud Run Service test-service finished: Service started. 0/1 deployment(s) still pending\n"),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			checkTimes := make(map[string]int)
			for resource := range test.responses {
				checkTimes[resource] = 0
			}
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if count, ok := checkTimes[r.URL.Path]; ok {
					checkTimes[r.URL.Path]++
					// strip version from the path
					responses := test.responses[r.URL.Path]
					if count >= len(responses) {
						count = len(responses) - 1
					}
					resp := responses[count]
					b, err := json.Marshal(resp)
					if err != nil {
						http.Error(w, "unable to marshal response: "+err.Error(), http.StatusInternalServerError)
						return
					}
					w.Write(b)
				} else {
					http.Error(w, "Resource not found "+r.URL.Path, http.StatusNotFound)
				}
			}))
			testEvent.InitializeState([]latest.Pipeline{{}})

			monitor := NewMonitor(labeller, []option.ClientOption{option.WithEndpoint(ts.URL)})
			// speed up checks for tests
			monitor.pollPeriod = 1 * time.Second
			monitor.reportStatusTime = 500 * time.Millisecond
			monitor.statusCheckDeadline = 5 * time.Second
			monitor.Resources = append(monitor.Resources, test.resources...)
			ctx := context.Background()
			out := new(bytes.Buffer)
			err := monitor.Check(ctx, out)
			if err != nil && !test.fail {
				t.Fatalf("expected success, got failure %v", err)
			} else if err == nil && test.fail {
				t.Fatalf("expected failure, got success. Output:\n%s", out.String())
			}
			t.CheckDeepEqual(test.expected, out.String())
		})
	}
}
