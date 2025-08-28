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

import "fmt"

const (
	typeService    = "Service"
	typeJob        = "Job"
	typeWorkerPool = "WorkerPool"
)

type ResourceType string

// RunResourceName represents a Cloud Run Service
type RunResourceName struct {
	Project    string
	Region     string
	Service    string
	Job        string
	WorkerPool string
}

// String returns the path representation of a Cloud Run Service.
func (n RunResourceName) String() string {
	// only one of Job, Service or WorkerPool should be specified
	if n.Service != "" {
		return fmt.Sprintf("projects/%s/locations/%s/services/%s", n.Project, n.Region, n.Service)
	}
	if n.WorkerPool != "" {
		return fmt.Sprintf("namespaces/%s/workerpools/%s", n.Project, n.WorkerPool)
	}
	return fmt.Sprintf("namespaces/%s/jobs/%s", n.Project, n.Job)
}
func (n RunResourceName) Name() string {
	if n.Service != "" {
		return n.Service
	}
	if n.WorkerPool != "" {
		return n.WorkerPool
	}
	return n.Job
}

func (n RunResourceName) Type() ResourceType {
	if n.Service != "" {
		return typeService
	}
	if n.WorkerPool != "" {
		return typeWorkerPool
	}
	return typeJob
}
