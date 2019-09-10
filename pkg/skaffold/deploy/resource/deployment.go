/*
Copyright 2019 The Skaffold Authors

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

package resource

import (
	"fmt"
	"time"
)

const (
	deploymentType = "deployment"
)

type Deployment struct {
	name      string
	namespace string
	rType     string
	deadline  time.Duration
	status    Status
}

func (d *Deployment) String() string {
	return fmt.Sprintf("%s:%s/%s", d.namespace, d.rType, d.name)
}

func (d *Deployment) Name() string {
	return d.name
}

func (d *Deployment) Deadline() time.Duration {
	return d.deadline
}

func (d *Deployment) Status() Status {
	return d.status
}

func (d *Deployment) UpdateStatus(details string, err error) {
	d.status = newStatus(details, err)
}

func NewDeployment(name string, ns string, deadline time.Duration) *Deployment {
	return &Deployment{
		name:      name,
		namespace: ns,
		rType:     deploymentType,
		deadline:  deadline,
		status:    newStatus("", nil),
	}
}

// For testing
func (d *Deployment) WithStatus(details string, err error) *Deployment {
	d.UpdateStatus(details, err)
	return d
}
