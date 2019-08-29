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
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
)

const (
	TabHeader = " -"
)

type ResourceObj struct {
	name      string
	namespace string
	rType     string
	status    Status
}

func (r *ResourceObj) String() string {
	return fmt.Sprintf("%s/%s", r.rType, r.name)
}

func (r *ResourceObj) Type() string {
	return r.rType
}

func (r *ResourceObj) UpdateStatus(msg string, reason string, err error) {
	newStatus := Status{details: msg, reason: reason, err: err}
	if !r.status.Equals(&newStatus) {
		r.status.err = err
		r.status.details = strings.TrimSuffix(msg, "\n")
		r.status.reason = strings.TrimSuffix(reason, "\n")
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

func (r *ResourceObj) checkComplete() {
	r.status.completed = true
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
	return strings.TrimSuffix(rs.reason, "\n") == strings.TrimSuffix(other.reason, "\n")
}

func (rs *Status) String() string {
	if rs.err != nil {
		return fmt.Sprintf("is pending due to %s", strings.TrimSuffix(rs.err.Error(), "\n"))
	}
	return fmt.Sprintf("is pending due to %s", rs.details)
}
