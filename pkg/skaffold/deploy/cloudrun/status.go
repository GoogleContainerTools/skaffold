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
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
	"google.golang.org/api/option"
	"google.golang.org/api/run/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	eventV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/gcp"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	proto "github.com/GoogleContainerTools/skaffold/proto/v2"
)

var (
	defaultStatusCheckDeadline = 10 * time.Minute
	defaultPollPeriod          = 1000 * time.Millisecond
	defaultReportStatusTime    = 5 * time.Second
)

type Monitor struct {
	Resources     []ResourceName
	clientOptions []option.ClientOption
	singleRun     singleflight.Group
	labeller      *label.DefaultLabeller

	statusCheckDeadline time.Duration
	pollPeriod          time.Duration
	reportStatusTime    time.Duration
}

type ResourceName struct {
	path string
	name string
}

func NewMonitor(labeller *label.DefaultLabeller, clientOptions []option.ClientOption) *Monitor {
	return &Monitor{
		labeller:            labeller,
		clientOptions:       clientOptions,
		statusCheckDeadline: defaultStatusCheckDeadline,
		pollPeriod:          defaultPollPeriod,
		reportStatusTime:    defaultReportStatusTime,
	}
}

func (s *Monitor) Reset() {
	s.Resources = nil
}

func (s *Monitor) Check(ctx context.Context, out io.Writer) error {
	_, err, _ := s.singleRun.Do(s.labeller.GetRunID(), func() (interface{}, error) {
		return struct{}{}, s.check(ctx, out)
	})
	return err
}
func (s *Monitor) check(ctx context.Context, out io.Writer) error {
	resources := make([]*runResource, len(s.Resources))
	for i, resource := range s.Resources {
		resources[i] = &runResource{path: resource.path, name: resource.name}
	}
	c := newCounter(len(resources))
	cctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var wg sync.WaitGroup
	var exitStatusOnce sync.Once
	exitStatus := proto.StatusCode_STATUSCHECK_SUCCESS
	for _, resource := range resources {
		wg.Add(1)
		go func(resource *runResource) {
			defer wg.Done()
			resource.pollResourceStatus(cctx, s.statusCheckDeadline, s.pollPeriod, s.clientOptions, true)
			c.markComplete()
			res := resource.status
			if res.ae.ErrCode != proto.StatusCode_STATUSCHECK_SUCCESS {
				exitStatusOnce.Do(func() { exitStatus = res.ae.ErrCode })
				cancel()
			}
			s.printStatusCheckSummary(out, c, resource)
		}(resource)
	}
	// Retrieve pending resource statuses
	go func() {
		s.printResourceStatus(ctx, out, resources)
	}()

	wg.Wait()
	return checkResults(c, exitStatus)
}

func checkResults(c *counter, exitStatus proto.StatusCode) error {
	if exitStatus != proto.StatusCode_STATUSCHECK_SUCCESS {
		return fmt.Errorf("skaffold deployment failed. %d/%d failed to complete", c.pending, c.total)
	}
	return nil
}

type counter struct {
	total   int32
	pending int32
}

func newCounter(i int) *counter {
	return &counter{
		total:   int32(i),
		pending: int32(i),
	}
}

func (c *counter) markComplete() {
	atomic.AddInt32(&c.pending, int32(-1))
}

func (c *counter) remaining() string {
	return fmt.Sprintf("%d/%d deployment(s) still pending", c.pending, c.total)
}

type runResource struct {
	path      string
	name      string
	completed bool
	status    Status
	url       string
	revision  string
}

type Status struct {
	ae       *proto.ActionableErr
	reported bool
}

func (r *runResource) pollResourceStatus(ctx context.Context, deadline time.Duration, pollPeriod time.Duration, clientOptions []option.ClientOption, useGcpOptions bool) {
	ticker := time.NewTicker(pollPeriod)
	defer ticker.Stop()
	timeoutContext, cancel := context.WithTimeout(ctx, deadline+pollPeriod)
	defer cancel()
	options := clientOptions
	if useGcpOptions {
		options = append(gcp.ClientOptions(ctx), options...)
	}
	crClient, err := run.NewService(ctx, options...)
	if err != nil {
		r.status = Status{ae: &proto.ActionableErr{
			ErrCode: proto.StatusCode_STATUSCHECK_KUBECTL_CLIENT_FETCH_ERR,
			Message: fmt.Sprintf("Unable to connect to Cloud Run: %v", err),
		}}
		return
	}
	for {
		select {
		case <-timeoutContext.Done():
			switch c := timeoutContext.Err(); c {
			case context.Canceled:
				r.updateStatus(&proto.ActionableErr{
					ErrCode: proto.StatusCode_STATUSCHECK_USER_CANCELLED,
					Message: "check cancelled\n",
				})
			case context.DeadlineExceeded:
				r.updateStatus(&proto.ActionableErr{
					ErrCode: proto.StatusCode_STATUSCHECK_DEADLINE_EXCEEDED,
					Message: fmt.Sprintf("Resource failed to become ready in %v", deadline),
				})
			}
			return
		case <-ticker.C:
			r.checkStatus(crClient)
			if r.completed {
				return
			}
		}
	}
}

func (r *runResource) updateStatus(ae *proto.ActionableErr) {
	curStatus := r.status
	if curStatus.ae != nil && ae.ErrCode == curStatus.ae.ErrCode && ae.Message == curStatus.ae.Message {
		return
	}
	r.status = Status{ae: ae}
}

func (r *runResource) ReportSinceLastUpdated() string {
	curStatus := r.status
	if curStatus.reported {
		return ""
	}
	curStatus.reported = true
	if curStatus.ae == nil {
		return ""
	}
	return fmt.Sprintf("%s: %s", r.name, curStatus.ae.Message)
}

func (r *runResource) checkStatus(crClient *run.APIService) {
	call := crClient.Projects.Locations.Services.Get(r.path)
	res, err := call.Do()
	if err != nil {
		r.updateStatus(&proto.ActionableErr{
			ErrCode: proto.StatusCode_STATUSCHECK_KUBECTL_CLIENT_FETCH_ERR,
			Message: fmt.Sprintf("Unable to check Cloud Run status: %v", err),
		})
		return
	}
	// find the ready condition
	var ready *run.GoogleCloudRunV1Condition

	// If the status is still showing the old generation, treat it the
	// same as no status being set.
	if res.Status.ObservedGeneration == res.Metadata.Generation {
		for _, cond := range res.Status.Conditions {
			if cond.Type == "Ready" {
				ready = cond
				break
			}
		}
	}
	if ready == nil {
		// No ready condition found, must not have started reconciliation yet
		r.updateStatus(&proto.ActionableErr{
			ErrCode: proto.StatusCode_STATUSCHECK_CONTAINER_WAITING_UNKNOWN,
			Message: "Waiting for service to start",
		})
		return
	}
	switch ready.Status {
	case "True":
		r.completed = true
		r.updateStatus(&proto.ActionableErr{
			ErrCode: proto.StatusCode_STATUSCHECK_SUCCESS,
			Message: "Service started",
		})
		r.url = res.Status.Url
		r.revision = res.Status.LatestReadyRevisionName
	case "False":
		r.completed = true
		r.updateStatus(&proto.ActionableErr{
			ErrCode: proto.StatusCode_STATUSCHECK_UNHEALTHY,
			Message: fmt.Sprintf("Service failed to start: %v", ready.Message),
		})
	default:
		// status is unknown
		r.updateStatus(&proto.ActionableErr{
			ErrCode: proto.StatusCode_STATUSCHECK_UNKNOWN,
			Message: fmt.Sprintf("Service starting: %v", ready.Message),
		})
	}
}

// printResourceStatus prints resource statuses until all status check are completed or context is cancelled.
func (s *Monitor) printResourceStatus(ctx context.Context, out io.Writer, resources []*runResource) {
	ticker := time.NewTicker(s.reportStatusTime)
	defer ticker.Stop()
	for {
		var allDone bool
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			allDone = s.printStatus(resources, out)
		}
		if allDone {
			return
		}
	}
}

func (s *Monitor) printStatus(resources []*runResource, out io.Writer) bool {
	allDone := true
	for _, res := range resources {
		if res.completed {
			continue
		}
		allDone = false
		if status := res.ReportSinceLastUpdated(); status != "" {
			eventV2.ResourceStatusCheckEventUpdated(res.path, res.status.ae)
			fmt.Fprintln(out, status)
		}
	}
	return allDone
}

func (s *Monitor) printStatusCheckSummary(out io.Writer, c *counter, r *runResource) {
	curStatus := r.status
	if curStatus.ae.ErrCode == proto.StatusCode_STATUSCHECK_USER_CANCELLED {
		// Don't print the status summary if the user ctrl-C or
		// another deployment failed
		return
	}
	eventV2.ResourceStatusCheckEventCompleted(r.path, curStatus.ae)
	if r.url != "" {
		eventV2.CloudRunServiceReady(r.path, r.url, r.revision)
	}
	if curStatus.ae.ErrCode != proto.StatusCode_STATUSCHECK_SUCCESS {
		output.Default.Fprintln(out, fmt.Sprintf("Cloud Run Service %s failed with error: %s", r.name, curStatus.ae.Message))
	} else {
		output.Default.Fprintln(out, fmt.Sprintf("Cloud Run Service %s finished: %s. %s", r.name, curStatus.ae.Message, c.remaining()))
	}
}
