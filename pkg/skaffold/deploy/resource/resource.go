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
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

const (
	tabHeader      = " -"
	deploymentType = "deployment"
)

type Resource struct {
	name      string
	namespace string
	rType     string
	status    Status
}

func (r *Resource) String() string {
	return fmt.Sprintf("%s:%s/%s", r.namespace, r.rType, r.name)
}

func (r *Resource) Type() string {
	return r.rType
}

func (r *Resource) UpdateStatus(msg string, err error) {
	newStatus := Status{details: msg, err: err}
	if !r.status.Equals(&newStatus) {
		r.status.err = err
		r.status.details = util.Trim(msg)
		r.status.updated = true
	}
}

func (r *Resource) Status() *Status {
	return &r.status
}

func (r *Resource) Namespace() string {
	return r.namespace
}

func (r *Resource) Name() string {
	return r.name
}

func (r *Resource) MarkCheckComplete() *Resource {
	r.status.completed = true
	return r
}

func (r *Resource) IsStatusCheckComplete() bool {
	return r.status.completed
}

func (r *Resource) ReportSinceLastUpdated(out io.Writer) {
	if !r.status.updated {
		return
	}
	r.status.updated = false
	color.Default.Fprintln(out, fmt.Sprintf("%s %s %s", tabHeader, r.String(), r.status.String()))
}

type Status struct {
	err       error
	details   string
	updated   bool
	completed bool
}

func (rs *Status) Error() error {
	return rs.err
}

func (rs *Status) Equals(other *Status) bool {
	if util.Trim(rs.details) != util.Trim(other.details) {
		return false
	}
	if rs.err == other.err {
		return true
	}
	return rs.err.Error() == other.err.Error()
}

func (rs *Status) String() string {
	if rs.err != nil {
		return util.Trim(rs.err.Error())
	}
	return rs.details
}

func NewResource(name string, ns string) *Resource {
	return &Resource{
		name:      name,
		namespace: ns,
		rType:     deploymentType,
	}
}

func (r *Resource) WithStatus(status Status) *Resource {
	r.status = status
	return r
}

// For testing, mimics when a Resource status is updated.
func (r *Resource) WithUpdatedStatus(status Status) *Resource {
	r.status = status
	r.status.updated = true
	return r
}

func NewStatus(msg string, err error) Status {
	return Status{
		details: msg,
		err:     err,
	}
}
