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

package resources

import (
	"fmt"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

const (
	TabHeader      = " -"
	DeploymentType = "deployment"
)

type ResourceObj struct {
	name      string
	namespace string
	rType     string
	status    Status
}

func (r *ResourceObj) String() string {
	return fmt.Sprintf("%s:%s/%s", r.namespace, r.rType, r.name)
}

func (r *ResourceObj) Type() string {
	return r.rType
}

func (r *ResourceObj) UpdateStatus(msg string, reason string, err error) {
	newStatus := Status{details: msg, reason: reason, err: err}
	if !r.status.Equals(&newStatus) {
		r.status.err = err
		r.status.details = util.Trim(msg)
		r.status.reason = util.Trim(reason)
		r.status.updated = true
	}
}

func (r *ResourceObj) Status() *Status {
	return &r.status
}

func (r *ResourceObj) Namespace() string {
	return r.namespace
}

func (r *ResourceObj) Name() string {
	return r.name
}

func (r *ResourceObj) MarkCheckComplete() *ResourceObj {
	r.status.completed = true
	return r
}

func (r *ResourceObj) IsStatusCheckComplete() bool {
	return r.status.completed
}

func (r *ResourceObj) ReportSinceLastUpdated(out io.Writer) {
	if !r.status.updated {
		return
	}
	r.status.updated = false
	color.Default.Fprintln(out, fmt.Sprintf("%s %s %s", TabHeader, r.String(), r.status.String()))
}

type Status struct {
	err       error
	reason    string
	details   string
	updated   bool
	completed bool
}

func (rs *Status) Error() error {
	return rs.err
}

func (rs *Status) Equals(other *Status) bool {
	return util.Trim(rs.reason) == util.Trim(other.reason)
}

func (rs *Status) String() string {
	if rs.err != nil {
		return fmt.Sprintf("%s", util.Trim(rs.err.Error()))
	}
	return fmt.Sprintf("%s", rs.details)
}

func NewResource(name string, ns string) *ResourceObj {
	return &ResourceObj{
		name:      name,
		namespace: ns,
		rType:     DeploymentType,
	}
}

func (r *ResourceObj) WithStatus(status Status) *ResourceObj {
	r.status = status
	return r
}

// For testing, mimics when a ResourceObj status is updated.
func (r *ResourceObj) WithUpdatedStatus(status Status) *ResourceObj {
	r.status = status
	r.status.updated = true
	return r
}

func NewStatus(msg string, reason string, err error) Status {
	return Status{
		details: msg,
		reason:  reason,
		err:     err,
	}
}
