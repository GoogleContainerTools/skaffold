/*
Copyright 2026 The Skaffold Authors

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
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/api/run/v1"

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

func fastTestBackOff(maxRetries uint64) func(context.Context) backoff.BackOff {
	return func(ctx context.Context) backoff.BackOff {
		b := backoff.NewConstantBackOff(1 * time.Millisecond)
		return backoff.WithContext(backoff.WithMaxRetries(b, maxRetries), ctx)
	}
}

type mockResponse struct {
	method string
	path   string
	status int
	body   interface{}
}

func newSequenceServer(sequence []mockResponse, reqCount *atomic.Int32) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idx := int(reqCount.Add(1)) - 1
		if idx >= len(sequence) {
			http.Error(w, "unexpected extra request: "+r.Method+" "+r.URL.Path, http.StatusTeapot)
			return
		}
		step := sequence[idx]
		if r.Method != step.method || r.URL.Path != step.path {
			http.Error(w, "unexpected method/path: "+r.Method+" "+r.URL.Path, http.StatusTeapot)
			return
		}
		if step.status != http.StatusOK {
			http.Error(w, http.StatusText(step.status), step.status)
			return
		}
		b, _ := json.Marshal(step.body)
		w.Write(b)
	}))
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil error", err: nil, want: false},
		{name: "context canceled", err: context.Canceled, want: false},
		{name: "context deadline exceeded", err: context.DeadlineExceeded, want: false},
		{name: "HTTP 429 Too Many Requests", err: &googleapi.Error{Code: http.StatusTooManyRequests}, want: true},
		{name: "HTTP 500 Internal Server Error", err: &googleapi.Error{Code: http.StatusInternalServerError}, want: true},
		{name: "HTTP 502 Bad Gateway", err: &googleapi.Error{Code: http.StatusBadGateway}, want: true},
		{name: "HTTP 503 Service Unavailable", err: &googleapi.Error{Code: http.StatusServiceUnavailable}, want: true},
		{name: "HTTP 504 Gateway Timeout", err: &googleapi.Error{Code: http.StatusGatewayTimeout}, want: true},
		{name: "HTTP 400 Bad Request", err: &googleapi.Error{Code: http.StatusBadRequest}, want: false},
		{name: "HTTP 401 Unauthorized", err: &googleapi.Error{Code: http.StatusUnauthorized}, want: false},
		{name: "HTTP 403 Forbidden", err: &googleapi.Error{Code: http.StatusForbidden}, want: false},
		{name: "HTTP 404 Not Found", err: &googleapi.Error{Code: http.StatusNotFound}, want: false},
		{name: "HTTP 409 Conflict", err: &googleapi.Error{Code: http.StatusConflict}, want: false},
		{name: "HTTP 412 Precondition Failed", err: &googleapi.Error{Code: http.StatusPreconditionFailed}, want: false},
		{name: "unexpected EOF", err: io.ErrUnexpectedEOF, want: true},
		{name: "ECONNRESET", err: syscall.ECONNRESET, want: true},
		{name: "generic error", err: errors.New("boom"), want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isRetryableError(tc.err); got != tc.want {
				t.Fatalf("isRetryableError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestDeployServiceRetries(tOuter *testing.T) {
	svc := &run.Service{
		ApiVersion: "serving.knative.dev/v1",
		Kind:       "Service",
		Metadata: &run.ObjectMeta{
			Name: "test-service",
		},
	}
	getPath := "/v1/projects/testProject/locations/us-central1/services/test-service"
	createPath := "/v1/projects/testProject/locations/us-central1/services"

	tests := []struct {
		description  string
		maxRetries   uint64
		sequence     []mockResponse
		wantErrCode  proto.StatusCode
		wantRequests int32
	}{
		{
			description: "retries transient 503 on Services.Get then creates service",
			maxRetries:  3,
			sequence: []mockResponse{
				{method: http.MethodGet, path: getPath, status: http.StatusServiceUnavailable},
				{method: http.MethodGet, path: getPath, status: http.StatusServiceUnavailable},
				{method: http.MethodGet, path: getPath, status: http.StatusNotFound},
				{method: http.MethodPost, path: createPath, status: http.StatusOK, body: svc},
			},
			wantErrCode:  proto.StatusCode_OK,
			wantRequests: 4,
		},
		{
			description: "retries transient 503 on ReplaceService and succeeds",
			maxRetries:  3,
			sequence: []mockResponse{
				{method: http.MethodGet, path: getPath, status: http.StatusOK, body: svc},
				{method: http.MethodPut, path: getPath, status: http.StatusServiceUnavailable},
				{method: http.MethodGet, path: getPath, status: http.StatusOK, body: svc},
				{method: http.MethodPut, path: getPath, status: http.StatusOK, body: svc},
			},
			wantErrCode:  proto.StatusCode_OK,
			wantRequests: 4,
		},
		{
			description: "idempotent recovery when Create returns 503 after backend creation",
			maxRetries:  3,
			sequence: []mockResponse{
				{method: http.MethodGet, path: getPath, status: http.StatusNotFound},
				{method: http.MethodPost, path: createPath, status: http.StatusServiceUnavailable},
				// On retry, Get sees 200 OK (created by attempt 1) and switches to ReplaceService.
				{method: http.MethodGet, path: getPath, status: http.StatusOK, body: svc},
				{method: http.MethodPut, path: getPath, status: http.StatusOK, body: svc},
			},
			wantErrCode:  proto.StatusCode_OK,
			wantRequests: 4,
		},
		{
			description: "exhausts retries on persistent 503 during Services.Get",
			maxRetries:  2,
			sequence: []mockResponse{
				{method: http.MethodGet, path: getPath, status: http.StatusServiceUnavailable},
				{method: http.MethodGet, path: getPath, status: http.StatusServiceUnavailable},
				{method: http.MethodGet, path: getPath, status: http.StatusServiceUnavailable},
			},
			wantErrCode:  proto.StatusCode_DEPLOY_CLOUD_RUN_GET_SERVICE_ERR,
			wantRequests: 3,
		},
		{
			description: "exhausts retries on persistent 503 during ReplaceService",
			maxRetries:  2,
			sequence: []mockResponse{
				{method: http.MethodGet, path: getPath, status: http.StatusOK, body: svc},
				{method: http.MethodPut, path: getPath, status: http.StatusServiceUnavailable},
				{method: http.MethodGet, path: getPath, status: http.StatusOK, body: svc},
				{method: http.MethodPut, path: getPath, status: http.StatusServiceUnavailable},
				{method: http.MethodGet, path: getPath, status: http.StatusOK, body: svc},
				{method: http.MethodPut, path: getPath, status: http.StatusServiceUnavailable},
			},
			wantErrCode:  proto.StatusCode_DEPLOY_CLOUD_RUN_UPDATE_SERVICE_ERR,
			wantRequests: 6,
		},
		{
			description: "fails immediately without retry on 403 Forbidden",
			maxRetries:  5,
			sequence: []mockResponse{
				{method: http.MethodGet, path: getPath, status: http.StatusForbidden},
			},
			wantErrCode:  proto.StatusCode_DEPLOY_CLOUD_RUN_GET_SERVICE_ERR,
			wantRequests: 1,
		},
		{
			description: "fails immediately without retry on 400 Bad Request during ReplaceService",
			maxRetries:  5,
			sequence: []mockResponse{
				{method: http.MethodGet, path: getPath, status: http.StatusOK, body: svc},
				{method: http.MethodPut, path: getPath, status: http.StatusBadRequest},
			},
			wantErrCode:  proto.StatusCode_DEPLOY_CLOUD_RUN_UPDATE_SERVICE_ERR,
			wantRequests: 2,
		},
	}

	for _, test := range tests {
		testutil.Run(tOuter, test.description, func(t *testutil.T) {
			t.Override(&newRetryBackOff, fastTestBackOff(test.maxRetries))

			var reqCount atomic.Int32
			ts := newSequenceServer(test.sequence, &reqCount)
			defer ts.Close()

			deployer, _ := NewDeployer(
				&runcontext.RunContext{},
				&label.DefaultLabeller{},
				&latest.CloudRunDeploy{ProjectID: "testProject", Region: "us-central1"},
				configName,
				defaultStatusCheckDeadline,
				true,
				util.Ptr(false),
			)
			deployer.clientOptions = append(deployer.clientOptions, option.WithEndpoint(ts.URL), option.WithoutAuthentication())
			deployer.useGcpOptions = false

			rawManifest, _ := json.Marshal(svc)
			manifestsByConfig := manifest.NewManifestListByConfig()
			manifestsByConfig.Add(configName, manifest.ManifestList{rawManifest})

			err := deployer.Deploy(context.Background(), new(bytes.Buffer), []graph.Artifact{}, manifestsByConfig)
			if test.wantErrCode == proto.StatusCode_OK {
				if err != nil {
					t.Fatalf("expected success, got error: %v", err)
				}
			} else {
				if err == nil {
					t.Fatalf("expected error code %v, got nil", test.wantErrCode)
				}
				var sErr sErrors.Error
				if !errors.As(err, &sErr) {
					t.Fatalf("expected sErrors.Error, got %T: %v", err, err)
				}
				if sErr.StatusCode() != test.wantErrCode {
					t.Fatalf("expected status code %v, got %v", test.wantErrCode, sErr.StatusCode())
				}
				var gErr *googleapi.Error
				if !errors.As(err, &gErr) {
					t.Fatalf("expected wrapped *googleapi.Error, got %T: %v", errors.Unwrap(err), err)
				}
			}
			if got := reqCount.Load(); got != test.wantRequests {
				t.Fatalf("expected %d HTTP requests, got %d", test.wantRequests, got)
			}
		})
	}
}

func TestDeployJobAndWorkerPoolRetries(tOuter *testing.T) {
	testutil.Run(tOuter, "Job retries 503 on Get and ReplaceJob", func(t *testutil.T) {
		t.Override(&newRetryBackOff, fastTestBackOff(3))
		job := &run.Job{
			ApiVersion: "run.googleapis.com/v1",
			Kind:       "Job",
			Metadata:   &run.ObjectMeta{Name: "test-job"},
		}
		getPath := "/apis/run.googleapis.com/v1/namespaces/testProject/jobs/test-job"

		var reqCount atomic.Int32
		sequence := []mockResponse{
			{method: http.MethodGet, path: getPath, status: http.StatusServiceUnavailable},
			{method: http.MethodGet, path: getPath, status: http.StatusOK, body: job},
			{method: http.MethodPut, path: getPath, status: http.StatusTooManyRequests},
			{method: http.MethodGet, path: getPath, status: http.StatusOK, body: job},
			{method: http.MethodPut, path: getPath, status: http.StatusOK, body: job},
		}
		ts := newSequenceServer(sequence, &reqCount)
		defer ts.Close()

		deployer, _ := NewDeployer(
			&runcontext.RunContext{},
			&label.DefaultLabeller{},
			&latest.CloudRunDeploy{ProjectID: "testProject", Region: "us-central1"},
			configName,
			defaultStatusCheckDeadline,
			false,
			util.Ptr(false),
		)
		deployer.clientOptions = append(deployer.clientOptions, option.WithEndpoint(ts.URL), option.WithoutAuthentication())
		deployer.useGcpOptions = false

		rawManifest, _ := json.Marshal(job)
		manifestsByConfig := manifest.NewManifestListByConfig()
		manifestsByConfig.Add(configName, manifest.ManifestList{rawManifest})

		if err := deployer.Deploy(context.Background(), new(bytes.Buffer), []graph.Artifact{}, manifestsByConfig); err != nil {
			t.Fatalf("expected Job deploy retry to succeed, got: %v", err)
		}
		if got := reqCount.Load(); got != 5 {
			t.Fatalf("expected 5 requests, got %d", got)
		}
	})

	testutil.Run(tOuter, "WorkerPool retries 503 on Get and Create", func(t *testutil.T) {
		t.Override(&newRetryBackOff, fastTestBackOff(3))
		wp := &run.WorkerPool{
			ApiVersion: "run.googleapis.com/v1",
			Kind:       "WorkerPool",
			Metadata:   &run.ObjectMeta{Name: "test-wp"},
		}
		getPath := "/apis/run.googleapis.com/v1/namespaces/testProject/workerpools/test-wp"
		createPath := "/apis/run.googleapis.com/v1/namespaces/testProject/workerpools"

		var reqCount atomic.Int32
		sequence := []mockResponse{
			{method: http.MethodGet, path: getPath, status: http.StatusServiceUnavailable},
			{method: http.MethodGet, path: getPath, status: http.StatusNotFound},
			{method: http.MethodPost, path: createPath, status: http.StatusBadGateway},
			{method: http.MethodGet, path: getPath, status: http.StatusNotFound},
			{method: http.MethodPost, path: createPath, status: http.StatusOK, body: wp},
		}
		ts := newSequenceServer(sequence, &reqCount)
		defer ts.Close()

		deployer, _ := NewDeployer(
			&runcontext.RunContext{},
			&label.DefaultLabeller{},
			&latest.CloudRunDeploy{ProjectID: "testProject", Region: "us-central1"},
			configName,
			defaultStatusCheckDeadline,
			false,
			util.Ptr(false),
		)
		deployer.clientOptions = append(deployer.clientOptions, option.WithEndpoint(ts.URL), option.WithoutAuthentication())
		deployer.useGcpOptions = false

		rawManifest, _ := json.Marshal(wp)
		manifestsByConfig := manifest.NewManifestListByConfig()
		manifestsByConfig.Add(configName, manifest.ManifestList{rawManifest})

		if err := deployer.Deploy(context.Background(), new(bytes.Buffer), []graph.Artifact{}, manifestsByConfig); err != nil {
			t.Fatalf("expected WorkerPool deploy retry to succeed, got: %v", err)
		}
		if got := reqCount.Load(); got != 5 {
			t.Fatalf("expected 5 requests, got %d", got)
		}
	})
}

func TestCleanupServiceRetries(tOuter *testing.T) {
	testutil.Run(tOuter, "Cleanup retries 503 and treats subsequent 404 as already deleted", func(t *testutil.T) {
		t.Override(&newRetryBackOff, fastTestBackOff(3))
		svc := &run.Service{
			ApiVersion: "serving.knative.dev/v1",
			Kind:       "Service",
			Metadata:   &run.ObjectMeta{Name: "test-service"},
		}
		delPath := "/v1/projects/testProject/locations/us-central1/services/test-service"

		var reqCount atomic.Int32
		sequence := []mockResponse{
			{method: http.MethodDelete, path: delPath, status: http.StatusServiceUnavailable},
			{method: http.MethodDelete, path: delPath, status: http.StatusNotFound},
		}
		ts := newSequenceServer(sequence, &reqCount)
		defer ts.Close()

		deployer, _ := NewDeployer(
			&runcontext.RunContext{},
			&label.DefaultLabeller{},
			&latest.CloudRunDeploy{ProjectID: "testProject", Region: "us-central1"},
			configName,
			defaultStatusCheckDeadline,
			false,
			util.Ptr(false),
		)
		deployer.clientOptions = append(deployer.clientOptions, option.WithEndpoint(ts.URL), option.WithoutAuthentication())
		deployer.useGcpOptions = false

		rawManifest, _ := json.Marshal(svc)
		manifestsByConfig := manifest.NewManifestListByConfig()
		manifestsByConfig.Add(configName, manifest.ManifestList{rawManifest})

		if err := deployer.Cleanup(context.Background(), new(bytes.Buffer), false, manifestsByConfig); err != nil {
			t.Fatalf("expected idempotent Cleanup retry to succeed, got: %v", err)
		}
		if got := reqCount.Load(); got != 2 {
			t.Fatalf("expected 2 requests, got %d", got)
		}
	})
}

type cancelOnSleepBackOff struct {
	cancel context.CancelFunc
}

func (b *cancelOnSleepBackOff) NextBackOff() time.Duration {
	b.cancel()
	return time.Hour
}

func (b *cancelOnSleepBackOff) Reset() {}

func TestMidSleepContextCancellation(tOuter *testing.T) {
	testutil.Run(tOuter, "deploy preserves Get error wrapper when cancelled during retry sleep", func(t *testutil.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		t.Override(&newRetryBackOff, func(ctx context.Context) backoff.BackOff {
			return backoff.WithContext(&cancelOnSleepBackOff{cancel: cancel}, ctx)
		})

		err := deployResourceWithRetry(
			ctx,
			"Service",
			"projects/testProject/locations/us-central1/services/test-service",
			proto.StatusCode_DEPLOY_CLOUD_RUN_GET_SERVICE_ERR,
			proto.StatusCode_DEPLOY_CLOUD_RUN_UPDATE_SERVICE_ERR,
			func(context.Context) error {
				return &googleapi.Error{Code: http.StatusServiceUnavailable, Message: "unavailable"}
			},
			func(context.Context) error { return nil },
			func(context.Context) error { return nil },
		)
		if err == nil {
			t.Fatalf("expected error on context cancellation, got nil")
		}
		var sErr sErrors.Error
		if !errors.As(err, &sErr) {
			t.Fatalf("expected sErrors.Error, got %T: %v", err, err)
		}
		if sErr.StatusCode() != proto.StatusCode_DEPLOY_CLOUD_RUN_GET_SERVICE_ERR {
			t.Fatalf("expected status code %v, got %v", proto.StatusCode_DEPLOY_CLOUD_RUN_GET_SERVICE_ERR, sErr.StatusCode())
		}
		if want := "error checking Cloud Run State: context canceled"; errors.Unwrap(err).Error() != want {
			t.Fatalf("expected unwrapped error %q, got %q", want, errors.Unwrap(err).Error())
		}
	})

	testutil.Run(tOuter, "deploy preserves Update error wrapper when cancelled during retry sleep", func(t *testutil.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		t.Override(&newRetryBackOff, func(ctx context.Context) backoff.BackOff {
			return backoff.WithContext(&cancelOnSleepBackOff{cancel: cancel}, ctx)
		})

		err := deployResourceWithRetry(
			ctx,
			"Service",
			"projects/testProject/locations/us-central1/services/test-service",
			proto.StatusCode_DEPLOY_CLOUD_RUN_GET_SERVICE_ERR,
			proto.StatusCode_DEPLOY_CLOUD_RUN_UPDATE_SERVICE_ERR,
			func(context.Context) error { return nil },
			func(context.Context) error { return nil },
			func(context.Context) error {
				return &googleapi.Error{Code: http.StatusServiceUnavailable, Message: "unavailable"}
			},
		)
		if err == nil {
			t.Fatalf("expected error on context cancellation, got nil")
		}
		var sErr sErrors.Error
		if !errors.As(err, &sErr) {
			t.Fatalf("expected sErrors.Error, got %T: %v", err, err)
		}
		if sErr.StatusCode() != proto.StatusCode_DEPLOY_CLOUD_RUN_UPDATE_SERVICE_ERR {
			t.Fatalf("expected status code %v, got %v", proto.StatusCode_DEPLOY_CLOUD_RUN_UPDATE_SERVICE_ERR, sErr.StatusCode())
		}
		if want := "error deploying Cloud Run Service: context canceled"; errors.Unwrap(err).Error() != want {
			t.Fatalf("expected unwrapped error %q, got %q", want, errors.Unwrap(err).Error())
		}
	})

	testutil.Run(tOuter, "delete aligns fallback error message when cancelled during retry sleep", func(t *testutil.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		t.Override(&newRetryBackOff, func(ctx context.Context) backoff.BackOff {
			return backoff.WithContext(&cancelOnSleepBackOff{cancel: cancel}, ctx)
		})

		err := deleteResourceWithRetry(
			ctx,
			"Service",
			"projects/testProject/locations/us-central1/services/test-service",
			proto.StatusCode_DEPLOY_CLOUD_RUN_DELETE_SERVICE_ERR,
			func(context.Context) error {
				return &googleapi.Error{Code: http.StatusServiceUnavailable, Message: "unavailable"}
			},
		)
		if err == nil {
			t.Fatalf("expected error on context cancellation, got nil")
		}
		var sErr sErrors.Error
		if !errors.As(err, &sErr) {
			t.Fatalf("expected sErrors.Error, got %T: %v", err, err)
		}
		if sErr.StatusCode() != proto.StatusCode_DEPLOY_CLOUD_RUN_DELETE_SERVICE_ERR {
			t.Fatalf("expected status code %v, got %v", proto.StatusCode_DEPLOY_CLOUD_RUN_DELETE_SERVICE_ERR, sErr.StatusCode())
		}
		if want := "unable to delete Cloud Run Service"; errors.Unwrap(err).Error() != want {
			t.Fatalf("expected unwrapped error %q, got %q", want, errors.Unwrap(err).Error())
		}
	})
}
