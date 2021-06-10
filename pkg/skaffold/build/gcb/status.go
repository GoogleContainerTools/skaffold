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
	reporter     *statusReporterImpl
	reporterOnce sync.Once
	client       *cloudbuild.Service
	clientOnce   sync.Once
)

type statusReporter interface {
	getStatus(ctx context.Context, projectID string, buildID string) <-chan result
}

type statusReporterImpl struct {
	results         map[jobID]chan result
	resultMutex     sync.RWMutex
	requests        chan jobRequest
	pollingInterval time.Duration
}

type jobID struct {
	projectID string
	buildID   string
}

type jobRequest struct {
	jobID
	ctx context.Context
}

type result struct {
	jobID
	status *cloudbuild.Build
	err    error
}

func getStatusReporter() statusReporter {
	reporterOnce.Do(func() {
		reporter = &statusReporterImpl{
			pollingInterval: PollingInterval,
			results:         make(map[jobID]chan result),
			requests:        make(chan jobRequest, 100),
		}

		go reporter.start()
	})
	return reporter
}

func (r *statusReporterImpl) start() {
	poll := time.NewTicker(r.pollingInterval)
	jobsByProjectID := make(map[string]map[jobID]jobRequest)
	for {
		select {
		case req := <-r.requests:
			if _, found := jobsByProjectID[req.projectID]; !found {
				jobsByProjectID[req.projectID] = make(map[jobID]jobRequest)
			}
			jobsByProjectID[req.projectID][req.jobID] = req
		case <-poll.C:
			for projectID, jobs := range jobsByProjectID {
				for id, req := range jobs {
					if req.ctx.Err() != nil {
						r.setResult(id, result{jobID: id, err: req.ctx.Err()})
						delete(jobs, id)
					}
				}
				statuses, err := getStatuses(projectID, jobs)
				if err != nil {
					for id := range jobs {
						r.setResult(id, result{jobID: id, err: err})
						delete(jobs, id)
					}
					continue
				}
				for id := range jobs {
					cb := statuses[id]
					switch cb.Status {
					case StatusQueued, StatusWorking, StatusUnknown:
					case StatusSuccess:
						r.setResult(id, result{jobID: id, status: cb})
						delete(jobs, id)
					case StatusFailure, StatusInternalError, StatusTimeout, StatusCancelled:
						r.setResult(id, result{jobID: id, err: fmt.Errorf("cloud build failed: %s", cb.Status)})
						delete(jobs, id)
					default:
						r.setResult(id, result{jobID: id, err: fmt.Errorf("unknown status: %s", cb.Status)})
						delete(jobs, id)
					}
				}
			}
		}
	}
}

func getStatuses(projectID string, jobs map[jobID]jobRequest) (map[jobID]*cloudbuild.Build, error) {
	var err error
	clientOnce.Do(func() {
		client, err = cloudbuild.NewService(context.Background(), gcp.ClientOptions()...)
	})
	if err != nil {
		return nil, fmt.Errorf("getting cloudbuild client: %w", err)
	}
	backoff := NewStatusBackoff()
	var cb *cloudbuild.ListBuildsResponse
	if waitErr := wait.Poll(backoff.Duration, RetryTimeout, func() (bool, error) {
		backoff.Step()
		cb, err = client.Projects.Builds.List(projectID).Filter(getFilterQuery(jobs)).Do()
		if err == nil {
			return true, nil
		}
		if strings.Contains(err.Error(), "Error 429: Quota exceeded for quota metric 'cloudbuild.googleapis.com/get_requests'") {
			// if we hit the rate limit, continue to retry
			return false, nil
		}
		return false, err
	}); waitErr != nil {
		return nil, fmt.Errorf("getting build status: %w", waitErr)
	}
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

func (r *statusReporterImpl) setResult(jobID jobID, result result) {
	r.resultMutex.RLock()
	r.results[jobID] <- result
	r.resultMutex.RUnlock()
}

func (r *statusReporterImpl) getStatus(ctx context.Context, projectID string, buildID string) <-chan result {
	id := jobID{projectID, buildID}
	res := make(chan result, 1)
	r.resultMutex.Lock()
	r.results[id] = res
	r.resultMutex.Unlock()
	r.requests <- jobRequest{id, ctx}
	return res
}
