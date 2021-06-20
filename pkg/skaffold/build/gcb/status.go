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
	results     map[jobID]chan result
	resultMutex sync.RWMutex
	requests    chan jobRequest
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
			results:  make(map[jobID]chan result),
			requests: make(chan jobRequest, 1),
		}

		go reporter.start()
	})
	return reporter
}

type poller struct {
	timers   map[string]*time.Timer
	backoffs map[string]*wait.Backoff
	add      chan string
	rm       chan string
	response chan string
	C        chan string
}

func newPoller() *poller {
	p := &poller{
		timers:   make(map[string]*time.Timer),
		backoffs: make(map[string]*wait.Backoff),
		add:      make(chan string, 1),
		response: make(chan string),
		C:        make(chan string),
	}

	go func() {
		for {
			select {
			case projectID := <-p.rm:
				if b, found := p.timers[projectID]; found {
					b.Stop()
				}
				delete(p.timers, projectID)
				delete(p.backoffs, projectID)
			case projectID := <-p.add:
				if b, found := p.timers[projectID]; found {
					b.Stop()
				}
				p.backoffs[projectID] = NewStatusBackoff()
				p.timers[projectID] = time.AfterFunc(p.backoffs[projectID].Step(), func() {
					p.response <- projectID
				})
			case projectID := <-p.response:
				go func() {
					p.C <- projectID
				}()
				b, found := p.backoffs[projectID]
				if !found {
					continue
				}
				p.timers[projectID] = time.AfterFunc(b.Step(), func() {
					p.response <- projectID
				})
			}
		}
	}()
	return p
}

func (p *poller) addOrReset(projectID string) {
	p.add <- projectID
}

func (p *poller) remove(projectID string) {
	p.rm <- projectID
}

func (r *statusReporterImpl) start() {
	poll := newPoller()
	jobsByProjectID := make(map[string]map[jobID]jobRequest)
	jobCancelled := make(chan jobRequest)
	retryCount := make(map[jobID]int)
	for {
		select {
		case job := <-jobCancelled:
			r.setResult(result{jobID: job.jobID, err: job.ctx.Err()})
			delete(jobsByProjectID[job.projectID], job.jobID)
		case req := <-r.requests:
			if _, found := jobsByProjectID[req.projectID]; !found {
				jobsByProjectID[req.projectID] = make(map[jobID]jobRequest)
			}
			jobsByProjectID[req.projectID][req.jobID] = req
			poll.addOrReset(req.projectID)
			go func() {
				<-req.ctx.Done()
				jobCancelled <- req
			}()
		case projectID := <-poll.C:
			jobs := jobsByProjectID[projectID]
			if len(jobs) == 0 {
				poll.remove(projectID)
				continue
			}
			statuses, err := getStatuses(projectID, jobs)
			if err != nil {
				for id := range jobs {
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
					r.setResult(result{jobID: id, err: fmt.Errorf("unknown status: %s", cb.Status)})
					delete(jobs, id)
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

func (r *statusReporterImpl) setResult(result result) {
	r.resultMutex.RLock()
	r.results[result.jobID] <- result
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
