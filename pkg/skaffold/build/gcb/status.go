/*
Copyright 2021 The Skaffold Authors

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

package gcb

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"google.golang.org/api/cloudbuild/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/gcp"
)

var (
	// maintain single instance of `statusReporter` per skaffold process
	reporter     *statusManagerImpl
	reporterOnce sync.Once
	// maintain single instance of the GCB client per skaffold process
	client     *cloudbuild.Service
	clientOnce sync.Once
)

// statusManager provides an interface for getting the status of GCB jobs.
// It reduces the number of requests made to the GCB API by using a single `List` call for
// all concurrently running builds for a project instead of multiple per-job `Get` calls.
// The statuses are polled with an independent exponential backoff strategy per project until success, cancellation or failure.
// The backoff duration for each `projectID` status query is reset with every new build request.
type statusManager interface {
	getStatus(ctx context.Context, projectID string, buildID string) <-chan result
}

// statusManagerImpl implements the `statusManager` interface
type statusManagerImpl struct {
	results     map[jobID]chan result
	resultMutex sync.RWMutex
	requests    chan jobRequest
}

// jobID represents a single build job
type jobID struct {
	projectID string
	buildID   string
}

// jobRequest encapsulates a single build job and it's context.
type jobRequest struct {
	jobID
	ctx context.Context
}

// result represents a single build job status
type result struct {
	jobID
	status *cloudbuild.Build
	err    error
}

func getStatusManager() statusManager {
	reporterOnce.Do(func() {
		reporter = &statusManagerImpl{
			results:  make(map[jobID]chan result),
			requests: make(chan jobRequest, 1),
		}
		// start processing all job requests in new goroutine.
		go reporter.run()
	})
	return reporter
}

// poller sends the `projectID` on channel `C` with an exponentially increasing period for each project.
type poller struct {
	timers   map[string]*time.Timer   // timers keep the next `Timer` keyed on the projectID
	backoffs map[string]*wait.Backoff // backoffs keeps the current `Backoff` keyed on the projectID
	reset    chan string              // reset channel receives any new `projectID` and resets the timer and backoff for that projectID
	remove   chan string              // remove channel receives a `projectID` that has no more running jobs and deletes its timer and backoff
	trans    chan string              // trans channel pushes to channel `C` and sets up the next trigger
	C        chan string              // C channel triggers with the `projectID`
}

func newPoller() *poller {
	p := &poller{
		timers:   make(map[string]*time.Timer),
		backoffs: make(map[string]*wait.Backoff),
		reset:    make(chan string),
		remove:   make(chan string),
		trans:    make(chan string),
		C:        make(chan string),
	}

	go func() {
		for {
			select {
			case projectID := <-p.remove:
				if b, found := p.timers[projectID]; found {
					b.Stop()
				}
				delete(p.timers, projectID)
				delete(p.backoffs, projectID)
			case projectID := <-p.reset:
				if b, found := p.timers[projectID]; found {
					b.Stop()
				}
				p.backoffs[projectID] = NewStatusBackoff()
				p.timers[projectID] = time.AfterFunc(p.backoffs[projectID].Step(), func() {
					p.trans <- projectID
				})
			case projectID := <-p.trans:
				go func() {
					p.C <- projectID
				}()
				b, found := p.backoffs[projectID]
				if !found {
					continue
				}
				p.timers[projectID] = time.AfterFunc(b.Step(), func() {
					p.trans <- projectID
				})
			}
		}
	}()
	return p
}

func (p *poller) resetTimer(projectID string) {
	p.reset <- projectID
}

func (p *poller) removeTimer(projectID string) {
	p.remove <- projectID
}

func (r *statusManagerImpl) run() {
	poll := newPoller()
	jobsByProjectID := make(map[string]map[jobID]jobRequest)
	jobCancelled := make(chan jobRequest)
	retryCount := make(map[jobID]int)
	for {
		select {
		case req := <-r.requests:
			// setup for each new build status request
			if _, found := jobsByProjectID[req.projectID]; !found {
				jobsByProjectID[req.projectID] = make(map[jobID]jobRequest)
			}
			jobsByProjectID[req.projectID][req.jobID] = req
			go poll.resetTimer(req.projectID)
			go func() {
				// setup cancellation for each job
				r := req
				<-r.ctx.Done()
				jobCancelled <- r
			}()
		case projectID := <-poll.C:
			// get status of all active jobs for `projectID`
			jobs := jobsByProjectID[projectID]
			if len(jobs) == 0 {
				poll.removeTimer(projectID)
				continue
			}
			statuses, err := getStatuses(projectID, jobs)
			if err != nil {
				for id := range jobs {
					// if the GCB API is throttling, ignore error and retry `MaxRetryCount` number of times.
					if strings.Contains(err.Error(), "Error 429: Quota exceeded for quota metric 'cloudbuild.googleapis.com/get_requests'") {
						retryCount[id]++
						if retryCount[id] < MaxRetryCount {
							continue
						}
					}
					r.setResult(result{jobID: id, err: err})
					delete(jobs, id)
				}
				continue
			}
			for id := range jobs {
				cb := statuses[id]
				switch cb.Status {
				case StatusQueued, StatusWorking, StatusUnknown:
				case StatusSuccess:
					r.setResult(result{jobID: id, status: cb})
					delete(jobs, id)
				case StatusFailure, StatusInternalError, StatusTimeout, StatusCancelled:
					r.setResult(result{jobID: id, err: fmt.Errorf("cloud build failed: %s", cb.Status)})
					delete(jobs, id)
				default:
					r.setResult(result{jobID: id, err: fmt.Errorf("unhandled status: %s", cb.Status)})
					delete(jobs, id)
				}
			}
		case job := <-jobCancelled:
			r.setResult(result{jobID: job.jobID, err: job.ctx.Err()})
			delete(jobsByProjectID[job.projectID], job.jobID)
		}
	}
}

func getStatuses(projectID string, jobs map[jobID]jobRequest) (map[jobID]*cloudbuild.Build, error) {
	var err error
	clientOnce.Do(func() {
		client, err = cloudbuild.NewService(context.Background(), gcp.ClientOptions()...)
	})
	if err != nil {
		clientOnce = sync.Once{} // reset on failure
		return nil, fmt.Errorf("getting cloudbuild client: %w", err)
	}
	cb, err := client.Projects.Builds.List(projectID).Filter(getFilterQuery(jobs)).Do()
	if err != nil {
		return nil, fmt.Errorf("getting build status: %w", err)
	}
	if cb == nil {
		return nil, errors.New("getting build status")
	}

	m := make(map[jobID]*cloudbuild.Build)
	for _, job := range jobs {
		found := false
		for _, b := range cb.Builds {
			if b.Id != job.buildID {
				continue
			}
			found = true
			m[job.jobID] = b
			break
		}
		if !found {
			return nil, errors.New("getting build status")
		}
	}
	return m, nil
}

func getFilterQuery(jobs map[jobID]jobRequest) string {
	var sl []string
	for job := range jobs {
		sl = append(sl, fmt.Sprintf("build_id=%s", job.buildID))
	}
	return strings.Join(sl, " OR ")
}

func (r *statusManagerImpl) setResult(result result) {
	r.resultMutex.RLock()
	r.results[result.jobID] <- result
	r.resultMutex.RUnlock()
}

func (r *statusManagerImpl) getStatus(ctx context.Context, projectID string, buildID string) <-chan result {
	id := jobID{projectID, buildID}
	res := make(chan result, 1)
	r.resultMutex.Lock()
	r.results[id] = res
	r.resultMutex.Unlock()
	r.requests <- jobRequest{id, ctx}
	return res
}
