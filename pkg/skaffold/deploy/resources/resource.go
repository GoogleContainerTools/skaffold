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
		return util.Trim(rs.err.Error())
	}
	return rs.details
}


func NewStatus(msg string, reason string, err error) Status {
	return Status{
		details: msg,
		reason:  reason,
		err:     err,
	}
}
