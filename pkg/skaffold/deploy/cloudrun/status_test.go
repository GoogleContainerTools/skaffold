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
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.golang.org/api/option"
	"google.golang.org/api/run/v1"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	proto "github.com/GoogleContainerTools/skaffold/v2/proto/v2"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
	testEvent "github.com/GoogleContainerTools/skaffold/v2/testutil/event"
)

func TestPrintSummaryStatus(t *testing.T) {
	labeller := label.NewLabeller(true, nil, "run-id")

	tests := []struct {
		description string
		pending     int32
		resource    RunResourceName
		ae          *proto.ActionableErr
		expected    string
	}{
		{
			description: "single resource running",
			pending:     int32(1),
			resource:    RunResourceName{Project: "test", Region: "region", Service: "test-service"},
			ae: &proto.ActionableErr{
				ErrCode: proto.StatusCode_STATUSCHECK_SUCCESS,
				Message: "Service started",
			},
			expected: "Cloud Run Service test-service finished: Service started. 1/10 deployment(s) still pending\n",
		},
		{
			description: "single job running",
			pending:     int32(1),
			resource:    RunResourceName{Project: "test", Region: "region", Job: "test-job"},
			ae: &proto.ActionableErr{
				ErrCode: proto.StatusCode_STATUSCHECK_SUCCESS,
				Message: "Job started",
			},
			expected: "Cloud Run Job test-job finished: Job started. 1/10 deployment(s) still pending\n",
		},
		{
			description: "single workerpool running",
			pending:     int32(1),
			resource:    RunResourceName{Project: "test", Region: "region", WorkerPool: "test-wp"},
			ae: &proto.ActionableErr{
				ErrCode: proto.StatusCode_STATUSCHECK_SUCCESS,
				Message: "WorkerPool started",
			},
			expected: "Cloud Run WorkerPool test-wp finished: WorkerPool started. 1/10 deployment(s) still pending\n",
		},
		{
			description: "nothing prints if cancelled",
			pending:     int32(3),
			resource:    RunResourceName{Project: "test", Region: " region", Service: "test-service"},
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
				resource: test.resource,
				status:   Status{ae: test.ae},
				sub:      &runServiceResource{path: test.resource.String()},
			}
			s := NewMonitor(labeller, []option.ClientOption{}, defaultStatusCheckDeadline, false)
			out := new(bytes.Buffer)
			testEvent.InitializeState([]latest.Pipeline{{}})
			c := newCounter(10)
			c.pending = test.pending
			s.printStatusCheckSummary(out, c, res)
			t.CheckDeepEqual(test.expected, out.String())
		})
	}
}
func TestPollServiceStatus(t *testing.T) {
	tests := []struct {
		description      string
		resource         RunResourceName
		responses        []run.Service
		expected         *proto.ActionableErr
		tolerateFailures bool
		fail             bool
	}{
		{
			description: "test basic check with one resource ready",
			resource:    RunResourceName{Project: "tp", Region: "tr", Service: "test-service"},
			responses: []run.Service{
				{
					ApiVersion: "serving.knative.dev/v1",
					Metadata: &run.ObjectMeta{
						Generation: 1,
					},
					Status: &run.ServiceStatus{
						ObservedGeneration: 1,
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
			resource:    RunResourceName{Project: " tp", Region: "tr", Service: "test-service"},
			responses: []run.Service{
				{
					ApiVersion: "serving.knative.dev/v1",
					Metadata: &run.ObjectMeta{
						Generation: 1,
					},
					Status: &run.ServiceStatus{
						ObservedGeneration: 1,
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
					Metadata: &run.ObjectMeta{
						Generation: 1,
					},
					Status: &run.ServiceStatus{
						ObservedGeneration: 1,
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
			description: "test previous deploy failed reports correctly",
			resource:    RunResourceName{Project: "tp", Region: "tr", Service: "test-service"},
			responses: []run.Service{
				{
					ApiVersion: "serving.knative.dev/v1",
					Metadata: &run.ObjectMeta{
						Generation: 2,
					},
					Status: &run.ServiceStatus{
						ObservedGeneration: 1,
						Conditions: []*run.GoogleCloudRunV1Condition{
							{
								Type:    "Ready",
								Status:  "False",
								Message: "Pre-existing failure",
							},
						},
					},
				},
				{
					ApiVersion: "serving.knative.dev/v1",
					Metadata: &run.ObjectMeta{
						Generation: 2,
					},
					Status: &run.ServiceStatus{
						ObservedGeneration: 2,
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
					Metadata: &run.ObjectMeta{
						Generation: 2,
					},
					Status: &run.ServiceStatus{
						ObservedGeneration: 2,
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
			defer ts.Close()
			testEvent.InitializeState([]latest.Pipeline{{}})

			resource := &runResource{resource: test.resource, sub: &runServiceResource{path: test.resource.String()}}
			ctx := context.Background()
			resource.pollResourceStatus(
				ctx,
				5*time.Second,
				1*time.Second,
				[]option.ClientOption{
					option.WithEndpoint(ts.URL),
					option.WithoutAuthentication(),
				},
				false,
				false)
			t.CheckDeepEqual(test.expected, resource.status.ae, protocmp.Transform())
		})
	}
}

func TestPollJobStatus(t *testing.T) {
	tests := []struct {
		description string
		resource    RunResourceName
		responses   []run.Job
		expected    *proto.ActionableErr
		fail        bool
	}{
		{
			description: "test basic check with one resource ready",
			resource:    RunResourceName{Project: "tp", Region: "tr", Job: "test-job"},
			responses: []run.Job{
				{
					ApiVersion: "run.googleapis.com/v1",
					Metadata: &run.ObjectMeta{
						Generation: 1,
					},
					Status: &run.JobStatus{
						ObservedGeneration: 1,
						Conditions: []*run.GoogleCloudRunV1Condition{
							{
								Type:   "Ready",
								Status: "True",
							},
						},
					},
				},
			},
			expected: &proto.ActionableErr{Message: "Job started", ErrCode: proto.StatusCode_STATUSCHECK_SUCCESS},
		},
		{
			description: "test previous deploy failed reports correctly",
			resource:    RunResourceName{Project: "tp", Region: "tr", Job: "test-job"},
			responses: []run.Job{
				{
					ApiVersion: "run.googleapis.com/v1",
					Metadata: &run.ObjectMeta{
						Generation: 2,
					},
					Status: &run.JobStatus{
						ObservedGeneration: 1,
						Conditions: []*run.GoogleCloudRunV1Condition{
							{
								Type:    "Ready",
								Status:  "False",
								Message: "Pre-existing failure",
							},
						},
					},
				},
				{
					ApiVersion: "run.googleapis.com/v1",
					Metadata: &run.ObjectMeta{
						Generation: 2,
					},
					Status: &run.JobStatus{
						ObservedGeneration: 2,
						Conditions: []*run.GoogleCloudRunV1Condition{
							{
								Type:   "Ready",
								Status: "Unknown",
							},
						},
					},
				},
				{
					ApiVersion: "run.googleapis.com/v1",
					Metadata: &run.ObjectMeta{
						Generation: 2,
					},
					Status: &run.JobStatus{
						ObservedGeneration: 2,
						Conditions: []*run.GoogleCloudRunV1Condition{
							{
								Type:   "Ready",
								Status: "True",
							},
						},
					},
				},
			},
			expected: &proto.ActionableErr{Message: "Job started", ErrCode: proto.StatusCode_STATUSCHECK_SUCCESS},
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
			defer ts.Close()
			testEvent.InitializeState([]latest.Pipeline{{}})

			resource := &runResource{resource: test.resource, sub: &runJobResource{path: test.resource.String()}}
			ctx := context.Background()
			resource.pollResourceStatus(
				ctx,
				5*time.Second,
				1*time.Second,
				[]option.ClientOption{
					option.WithEndpoint(ts.URL),
					option.WithoutAuthentication(),
				},
				false,
				false)
			t.CheckDeepEqual(test.expected, resource.status.ae, protocmp.Transform())
		})
	}
}

func TestPollWorkerPoolStatus(t *testing.T) {
	tests := []struct {
		description string
		resource    RunResourceName
		responses   []run.WorkerPool
		expected    *proto.ActionableErr
		fail        bool
	}{
		{
			description: "test basic check with one workerpool ready",
			resource:    RunResourceName{Project: "tp", Region: "tr", WorkerPool: "test-wp"},
			responses: []run.WorkerPool{
				{
					ApiVersion: "run.googleapis.com/v1",
					Metadata: &run.ObjectMeta{
						Generation: 1,
					},
					Status: &run.WorkerPoolStatus{
						ObservedGeneration: 1,
						Conditions: []*run.GoogleCloudRunV1Condition{
							{
								Type:   "Ready",
								Status: "True",
							},
						},
					},
				},
			},
			expected: &proto.ActionableErr{Message: "WorkerPool started", ErrCode: proto.StatusCode_STATUSCHECK_SUCCESS},
		},
		{
			description: "test workerpool going ready after 1 non-ready",
			resource:    RunResourceName{Project: "tp", Region: "tr", WorkerPool: "test-wp"},
			responses: []run.WorkerPool{
				{
					ApiVersion: "run.googleapis.com/v1",
					Metadata: &run.ObjectMeta{
						Generation: 1,
					},
					Status: &run.WorkerPoolStatus{
						ObservedGeneration: 1,
						Conditions: []*run.GoogleCloudRunV1Condition{
							{
								Type:    "Ready",
								Status:  "Unknown",
								Message: "Creating",
							},
						},
					},
				},
				{
					ApiVersion: "run.googleapis.com/v1",
					Metadata: &run.ObjectMeta{
						Generation: 1,
					},
					Status: &run.WorkerPoolStatus{
						ObservedGeneration: 1,
						Conditions: []*run.GoogleCloudRunV1Condition{
							{
								Type:   "Ready",
								Status: "True",
							},
						},
					},
				},
			},
			expected: &proto.ActionableErr{Message: "WorkerPool started", ErrCode: proto.StatusCode_STATUSCHECK_SUCCESS},
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
			defer ts.Close()
			testEvent.InitializeState([]latest.Pipeline{{}})

			resource := &runResource{resource: test.resource, sub: &runWorkerPoolResource{path: test.resource.String()}}
			ctx := context.Background()
			resource.pollResourceStatus(
				ctx,
				5*time.Second,
				1*time.Second,
				[]option.ClientOption{
					option.WithEndpoint(ts.URL),
					option.WithoutAuthentication(),
				},
				false,
				false)
			t.CheckDeepEqual(test.expected, resource.status.ae, protocmp.Transform())
		})
	}
}

func TestMonitorPrintStatus(t *testing.T) {
	labeller := label.NewLabeller(true, nil, "run-id")
	tests := []struct {
		description string
		resources   []*runResource
		expected    string
		done        bool
	}{
		{
			description: "test basic print with one resource getting ready",
			resources: []*runResource{
				{
					resource:  RunResourceName{Project: "tp", Region: "tr", Service: "test-service"},
					completed: false,
					status: Status{
						reported: false,
						ae: &proto.ActionableErr{
							ErrCode: proto.StatusCode_STATUSCHECK_CONTAINER_WAITING_UNKNOWN,
							Message: "Waiting for service to start",
						},
					},
				},
			},
			expected: "test-service: Waiting for service to start\n",
			done:     false,
		},
		{
			description: "test basic print with one resource ready and reported, one not ready",
			resources: []*runResource{
				{
					resource:  RunResourceName{Project: "tp", Region: "tr", Service: "test-service1"},
					completed: true,
					status: Status{
						reported: true,
						ae: &proto.ActionableErr{
							ErrCode: proto.StatusCode_STATUSCHECK_SUCCESS,
							Message: "Service started",
						},
					},
				},
				{
					resource:  RunResourceName{Project: "tp", Region: "tr", Service: "test-service2"},
					completed: false,
					status: Status{
						reported: false,
						ae: &proto.ActionableErr{
							ErrCode: proto.StatusCode_STATUSCHECK_CONTAINER_CREATING,
							Message: "Service starting: Deploying Revision",
						},
					},
				},
			},

			expected: ("test-service2: Service starting: Deploying Revision\n"),
		},
		{
			description: "test basic print with one wp ready and reported, one not ready",
			resources: []*runResource{
				{
					resource:  RunResourceName{Project: "tp", Region: "tr", WorkerPool: "test-wp1"},
					completed: true,
					status: Status{
						reported: true,
						ae: &proto.ActionableErr{
							ErrCode: proto.StatusCode_STATUSCHECK_SUCCESS,
							Message: "WorkerPool started",
						},
					},
				},
				{
					resource:  RunResourceName{Project: "tp", Region: "tr", WorkerPool: "test-wp2"},
					completed: false,

					status: Status{
						reported: false,
						ae: &proto.ActionableErr{
							ErrCode: proto.StatusCode_STATUSCHECK_CONTAINER_CREATING,
							Message: "WorkerPool Creating",
						},
					},
				},
			},
			expected: ("test-wp2: WorkerPool Creating\n"),
		},
		{
			description: "test resources completed",
			resources: []*runResource{
				{
					resource:  RunResourceName{Project: "tp", Region: "tr", Service: "test-service"},
					completed: true,
				},
			},
			done: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			testEvent.InitializeState([]latest.Pipeline{{}})

			monitor := NewMonitor(labeller, []option.ClientOption{}, defaultStatusCheckDeadline, false)
			out := new(bytes.Buffer)
			done := monitor.printStatus(test.resources, out)
			if done != test.done {
				t.Fatalf("Expected finished state to be %v but got %v. Output:\n%s", test.done, done, out.String())
			}
			t.CheckDeepEqual(test.expected, out.String())
		})
	}
}
